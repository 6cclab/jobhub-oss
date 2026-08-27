package handler

import (
	"strings"
	"testing"
)

// The allowed set must stay in lockstep with the fit_reports.verdict CHECK
// constraint in migrations/0001_init.up.sql. If that migration changes and this
// list does not, valid input starts 400ing (or invalid input starts 500ing).
func TestValidFitVerdictMatchesSchemaConstraint(t *testing.T) {
	fromSchema := []string{"strong", "worth", "stretch", "skip"}

	if len(fitVerdicts) != len(fromSchema) {
		t.Fatalf("fitVerdicts has %d entries, schema allows %d: %v vs %v",
			len(fitVerdicts), len(fromSchema), fitVerdicts, fromSchema)
	}
	for _, v := range fromSchema {
		if !validFitVerdict(v) {
			t.Errorf("schema allows %q but validFitVerdict rejects it", v)
		}
	}
}

func TestValidFitVerdictRejects(t *testing.T) {
	// "acceptable" and "needs_rework" are eval_results verdicts, not fit
	// report verdicts — an easy mix-up worth guarding against.
	rejects := []string{"", "Strong", "STRONG", "acceptable", "needs_rework", "stable", "maybe", " skip"}

	for _, v := range rejects {
		if validFitVerdict(v) {
			t.Errorf("expected %q to be rejected", v)
		}
	}
}

func TestFitVerdictErrorNamesTheOptions(t *testing.T) {
	msg := fitVerdictError("maybe")

	if !strings.Contains(msg, `"maybe"`) {
		t.Errorf("error should quote the bad value, got: %s", msg)
	}
	for _, v := range fitVerdicts {
		if !strings.Contains(msg, v) {
			t.Errorf("error should list valid option %q, got: %s", v, msg)
		}
	}
}
