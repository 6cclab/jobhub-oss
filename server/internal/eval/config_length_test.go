package eval

import "testing"

// The length band used to be a single hard-coded 1500-4500 range, which meant a
// resume deliberately built as two pages — something resume-style.md explicitly
// sanctions for broad postings — could never clear the style dimension. These
// cases are the real character counts of resumes that were actually sent.
func TestLengthBoundsByTargetPages(t *testing.T) {
	cfg := DefaultConfig()

	cases := []struct {
		name  string
		chars int
		pages int
		want  bool
	}{
		{"crowdstrike one page", 3382, 1, true},
		{"seeq one page", 3515, 1, true},
		{"affirm as the two-pager it is", 5112, 2, true},
		{"affirm scored as one page", 5112, 1, false},
		{"roadie is too long even for two", 9400, 2, false},
		{"a stub is too short", 400, 1, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			min, max := cfg.LengthBounds(tc.pages)
			got := tc.chars >= min && tc.chars <= max
			if got != tc.want {
				t.Errorf("%d chars at %d page(s): lengthOK = %v, want %v (band %d-%d)",
					tc.chars, tc.pages, got, tc.want, min, max)
			}
		})
	}
}

// An unset TargetPages must keep the previous one-page behaviour rather than
// defaulting to the more permissive band, so an untailored caller cannot
// accidentally opt out of the length check entirely.
func TestLengthBoundsDefaultsToOnePage(t *testing.T) {
	cfg := DefaultConfig()
	zeroMin, zeroMax := cfg.LengthBounds(0)
	oneMin, oneMax := cfg.LengthBounds(1)
	if zeroMin != oneMin || zeroMax != oneMax {
		t.Errorf("unset target pages gave %d-%d, want the one-page band %d-%d",
			zeroMin, zeroMax, oneMin, oneMax)
	}
}

// The thresholds are only "synced" to resume-style.md if the guide's numbers are
// what the engine actually starts from. These assert the documented values.
func TestDefaultsMatchTheStyleGuide(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.MaxSentenceWords != 45 {
		t.Errorf("MaxSentenceWords = %d, want 45 (resume-style.md: \"Reject anything over 45 words\")",
			cfg.MaxSentenceWords)
	}
	if cfg.OnePageMaxChars != 4500 {
		t.Errorf("OnePageMaxChars = %d, want 4500", cfg.OnePageMaxChars)
	}
	if cfg.TwoPageMinChars < cfg.OnePageMaxChars {
		t.Errorf("two-page floor %d sits below the one-page ceiling %d, which leaves a resume "+
			"that is too long for one page and too short for two with no valid band",
			cfg.TwoPageMinChars, cfg.OnePageMaxChars)
	}
}
