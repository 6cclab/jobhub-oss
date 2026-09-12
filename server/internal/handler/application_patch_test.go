package handler

import (
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
)

// The bug these cover, found 2026-08-24: PATCH /api/applications/:id accepted a
// `source` update, returned 200, and discarded it. The field was missing from
// UpdateApplicationRequest and Fiber's BodyParser drops unknown keys silently,
// so the response was indistinguishable from a real write. A follow-up GET was
// the only way to notice. Every case below is about making a non-write loud.

func statusOf(t *testing.T, err error) int {
	t.Helper()
	var fe *fiber.Error
	if !asFiberError(err, &fe) {
		t.Fatalf("expected *fiber.Error, got %T: %v", err, err)
	}
	return fe.Code
}

func asFiberError(err error, target **fiber.Error) bool {
	fe, ok := err.(*fiber.Error)
	if ok {
		*target = fe
	}
	return ok
}

func TestPatchAcceptsEveryRepoUpdatableColumn(t *testing.T) {
	// Mirrors ApplicationRepo.Update's allowed map. If a column is added there
	// and not here, this test is the reminder that the API cannot reach it.
	cases := map[string]string{
		"status":        `{"status":"phone_screen"}`,
		"notes":         `{"notes":"n"}`,
		"source":        `{"source":"Recruiter inbound"}`,
		"role_title":    `{"role_title":"Senior Software Engineer"}`,
		"applied_at":    `{"applied_at":"2026-08-24"}`,
		"resume_file":   `{"resume_file":"a.md"}`,
		"fit_report_id": `{"fit_report_id":"abc"}`,
	}

	for col, body := range cases {
		t.Run(col, func(t *testing.T) {
			req, err := decodeUpdateApplication([]byte(body))
			if err != nil {
				t.Fatalf("decode %s: %v", body, err)
			}
			if col == "status" {
				if req.Status == nil || *req.Status != "phone_screen" {
					t.Fatalf("status did not survive decode: %+v", req)
				}
				return
			}
			if _, ok := req.Fields()[col]; !ok {
				t.Fatalf("%q decoded but is absent from Fields(): %v", col, req.Fields())
			}
		})
	}
}

func TestPatchSourceIsNotSilentlyDropped(t *testing.T) {
	// The exact regression. Imprint was PATCHed with a corrected source, got a
	// 200, and kept the old value -- which then miscounted it as a cold
	// application in the funnel breakdown.
	req, err := decodeUpdateApplication([]byte(`{"source":"Recruiter inbound (Ashby posting)"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, ok := req.Fields()["source"]
	if !ok {
		t.Fatal("source missing from Fields(); this is the original bug")
	}
	if got != "Recruiter inbound (Ashby posting)" {
		t.Fatalf("source = %v, want the posted value", got)
	}
}

func TestPatchRejectsUnknownFieldInsteadOf200(t *testing.T) {
	_, err := decodeUpdateApplication([]byte(`{"scource":"typo"}`))
	if err == nil {
		t.Fatal("a misspelled key was accepted; it would return 200 and write nothing")
	}
	if code := statusOf(t, err); code != fiber.StatusBadRequest {
		t.Fatalf("status = %d, want 400", code)
	}
}

func TestPatchRejectsBodyWithNoUpdatableField(t *testing.T) {
	for _, body := range []string{`{}`, `{"status":""}`} {
		if _, err := decodeUpdateApplication([]byte(body)); err == nil {
			t.Fatalf("%s was accepted but updates nothing", body)
		} else if code := statusOf(t, err); code != fiber.StatusBadRequest {
			t.Fatalf("%s: status = %d, want 400", body, code)
		}
	}
}

func TestPatchErrorNamesTheUpdatableFields(t *testing.T) {
	_, err := decodeUpdateApplication([]byte(`{}`))
	if err == nil {
		t.Fatal("expected an error")
	}
	for _, want := range []string{"status", "notes", "source", "role_title", "fit_report_id"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error does not name %q, so a caller cannot learn the vocabulary: %v", want, err)
		}
	}
}

func TestPatchRejectsMalformedJSON(t *testing.T) {
	if _, err := decodeUpdateApplication([]byte(`{"status":`)); err == nil {
		t.Fatal("malformed JSON was accepted")
	} else if code := statusOf(t, err); code != fiber.StatusBadRequest {
		t.Fatalf("status = %d, want 400", code)
	}
}

func TestPatchOmittedFieldsStayAbsent(t *testing.T) {
	// A partial update must not blank the columns it does not mention.
	req, err := decodeUpdateApplication([]byte(`{"notes":"only notes"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	f := req.Fields()
	if len(f) != 1 {
		t.Fatalf("Fields() = %v, want only notes", f)
	}
	if req.Source != nil || req.RoleTitle != nil || req.AppliedAt != nil {
		t.Fatalf("omitted fields became non-nil: %+v", req)
	}
}

func TestPatchDistinguishesExplicitEmptyStringFromOmitted(t *testing.T) {
	// Clearing a field is a legitimate update and must reach the repo.
	req, err := decodeUpdateApplication([]byte(`{"notes":""}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, ok := req.Fields()["notes"]
	if !ok {
		t.Fatal(`{"notes":""} should clear notes, not be treated as absent`)
	}
	if got != "" {
		t.Fatalf("notes = %v, want empty string", got)
	}
}
