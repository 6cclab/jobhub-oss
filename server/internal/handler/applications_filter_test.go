package handler

import (
	"testing"

	"github.com/6cclab/jobhub/internal/models"
)

func ptr(s string) *string { return &s }

// The sample deliberately includes a row with a nil status. The column is NOT
// NULL with a default, but the model carries *string and List() scans whatever
// is there — so a nil is representable in Go even when the schema says it
// should not happen, and a filter that dereferences it panics in production
// rather than in a test.
func sampleApplications() []models.Application {
	return []models.Application{
		{ID: "1", CompanyName: "Headway", RoleTitle: "Senior Fullstack Software Engineer", Status: ptr("rejected")},
		{ID: "2", CompanyName: "CrowdStrike", RoleTitle: "Sr SWE, Cloud Asset Platform", Status: ptr("phone_screen")},
		{ID: "3", CompanyName: "Affirm", RoleTitle: "Staff SWE, CI", Status: ptr("applied")},
		{ID: "4", CompanyName: "Roadie", RoleTitle: "Senior Full Stack Engineer", Status: ptr("applied")},
		{ID: "5", CompanyName: "Seeq", RoleTitle: "Staff SWE, Developer Experience", Status: nil},
	}
}

func ids(apps []models.Application) []string {
	out := make([]string, 0, len(apps))
	for _, a := range apps {
		out = append(out, a.ID)
	}
	return out
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestFilterApplications(t *testing.T) {
	cases := []struct {
		name    string
		company string
		status  string
		want    []string
	}{
		{"no filters returns everything", "", "", []string{"1", "2", "3", "4", "5"}},
		{"company matches exactly", "Headway", "", []string{"1"}},
		// The reason the match is case-insensitive: a caller should not have to
		// know that the row says "CrowdStrike" and not "Crowdstrike".
		{"company ignores case", "headway", "", []string{"1"}},
		{"company ignores case the other way", "CROWDSTRIKE", "", []string{"2"}},
		{"company matches a prefix", "crowd", "", []string{"2"}},
		{"company matches an interior substring", "adie", "", []string{"4"}},
		{"company matching nothing returns empty", "netflix", "", nil},
		// Row 5 has a nil status and is included: nil means the column default,
		// which is 'applied'. See TestFilterApplicationsTreatsNilStatusAsApplied.
		{"status filters exactly", "", "applied", []string{"3", "4", "5"}},
		{"status with no matches returns empty", "", "offer", nil},
		{"company and status together", "Headway", "rejected", []string{"1"}},
		// Both filters apply; matching one is not enough.
		{"company and status must both match", "Headway", "applied", nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ids(filterApplications(sampleApplications(), tc.company, tc.status))
			if !equal(got, tc.want) && !(len(got) == 0 && len(tc.want) == 0) {
				t.Errorf("filterApplications(%q, %q) = %v, want %v", tc.company, tc.status, got, tc.want)
			}
		})
	}
}

// A nil status means the row takes the column default, which is 'applied'.
// Treating nil as "matches nothing" would silently hide rows from exactly the
// filter most likely to be used.
func TestFilterApplicationsTreatsNilStatusAsApplied(t *testing.T) {
	got := ids(filterApplications(sampleApplications(), "Seeq", "applied"))

	if !equal(got, []string{"5"}) {
		t.Errorf("nil status should match the 'applied' default, got %v", got)
	}
}

// Never nil, so a caller can range over the result and the handler marshals
// `[]` rather than `null`. A JSON `null` where an array was documented is the
// kind of thing that breaks a client months later.
func TestFilterApplicationsReturnsEmptyNotNil(t *testing.T) {
	got := filterApplications(sampleApplications(), "nobody", "")

	if got == nil {
		t.Fatal("expected an empty slice, got nil")
	}
	if len(got) != 0 {
		t.Errorf("expected no matches, got %d", len(got))
	}
}

// Order is the repository's (created_at DESC) and the filter must not disturb
// it — the dashboard and the API should agree on what "most recent" means.
func TestFilterApplicationsPreservesOrder(t *testing.T) {
	got := ids(filterApplications(sampleApplications(), "", ""))

	if !equal(got, []string{"1", "2", "3", "4", "5"}) {
		t.Errorf("filter reordered the list: %v", got)
	}
}

// Same lockstep contract as TestValidFitVerdictMatchesSchemaConstraint: this
// list mirrors the applications.status CHECK constraint in
// migrations/0001_init.up.sql. If the migration gains a status and this does
// not, filtering by the new one 400s despite being valid in the database.
func TestApplicationStatusesMatchSchemaConstraint(t *testing.T) {
	fromSchema := []string{"applied", "phone_screen", "onsite", "offer", "rejected", "ghosted", "withdrawn"}

	if len(applicationStatuses) != len(fromSchema) {
		t.Fatalf("applicationStatuses has %d entries, schema allows %d: %v vs %v",
			len(applicationStatuses), len(fromSchema), applicationStatuses, fromSchema)
	}
	for _, s := range fromSchema {
		if !validApplicationStatus(s) {
			t.Errorf("schema allows %q but validApplicationStatus rejects it", s)
		}
	}
}

func TestValidApplicationStatusRejects(t *testing.T) {
	// "interviewing" is the one that actually bit: it reads like a status this
	// system would have, and the database rejected it with a constraint error.
	rejects := []string{"", "interviewing", "Applied", "APPLIED", "strong", "accepted", " applied"}

	for _, s := range rejects {
		if validApplicationStatus(s) {
			t.Errorf("expected %q to be rejected", s)
		}
	}
}
