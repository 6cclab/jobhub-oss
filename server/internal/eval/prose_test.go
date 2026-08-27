package eval

import "testing"

func typesOf(issues []ProseIssue) map[string]int {
	m := map[string]int{}
	for _, i := range issues {
		m[i.Type]++
	}
	return m
}

// The fragment that shipped in the Headway resume on 2026-08-07 and was caught
// by hand rather than by the engine. This is the regression test for it.
func TestCheckProse_CatchesRealFragment(t *testing.T) {
	resume := `## Experience

### General Assembly — New York, NY

- Taught full-stack development and Python to 75+ students, 90% placed into engineering roles within six months. Mentoring cohorts of ~28 through 1:1 support and workshops.
`
	got := typesOf(checkProse(resume, 1, longSentenceWords))
	if got[ProseFragment] != 1 {
		t.Fatalf("expected 1 fragment, got %d (%+v)", got[ProseFragment], checkProse(resume, 1, longSentenceWords))
	}
}

func TestCheckProse_AcceptsTheFix(t *testing.T) {
	resume := `## Experience

- Taught full-stack development and Python to 75+ students, 90% placed into engineering roles within six months, and built the React Native curriculum from scratch, mentoring cohorts of ~28 through 1:1 support and group workshops.
`
	if issues := checkProse(resume, 1, longSentenceWords); len(issues) != 0 {
		t.Fatalf("expected clean, got %+v", issues)
	}
}

func TestCheckProse_NoFalsePositivesOnGoodBullets(t *testing.T) {
	resume := `## Summary

I am a fullstack engineer who likes end-to-end ownership of a problem, from the database schema through to the thing a person actually clicks.

## Skills

- **Languages:** TypeScript, Python, Node.js
- **Infrastructure:** AWS, Kubernetes, CI/CD, Datadog

## Experience

### LegalZoom — Remote

**Staff Software Engineer (SE IV)** — Aug 2025 - Aug 2026

- Built a developer-experience observability platform from zero, giving a 300+ engineer monorepo its first real data.
- Co-led the React Router 7 upgrade across every contributing team in the monorepo, a 454-file production change.
- Found a class of bug where an empty upstream entitlements response with no error was read by the UI as entitled.
- Designed AWS infrastructure and built social analytics tooling on a microservices architecture.

## Education

**General Assembly** — Software Engineering Immersive, New York, NY
`
	issues := checkProse(resume, 3, longSentenceWords)
	for _, i := range issues {
		if i.Type == ProseFragment {
			t.Errorf("false positive fragment: %q", i.Text)
		}
	}
}

func TestCheckProse_EmDashLimit(t *testing.T) {
	resume := "## Summary\n\nI built one thing — and another — and a third.\n"
	if got := typesOf(checkProse(resume, 1, longSentenceWords))[ProseEmDash]; got != 1 {
		t.Errorf("expected em_dash issue, got %d", got)
	}
	if got := typesOf(checkProse(resume, 2, longSentenceWords))[ProseEmDash]; got != 0 {
		t.Errorf("expected no em_dash issue at limit 2, got %d", got)
	}
}

func TestCheckProse_RepeatedWord(t *testing.T) {
	resume := "## Experience\n\n- Built the the observability platform from zero for a large monorepo.\n"
	if got := typesOf(checkProse(resume, 1, longSentenceWords))[ProseRepeatedWord]; got != 1 {
		t.Errorf("expected repeated_word, got %d", got)
	}
}

func TestCheckProse_LongSentence(t *testing.T) {
	long := "- Built a platform that "
	for range 50 {
		long += "word "
	}
	long += "shipped.\n"
	if got := typesOf(checkProse("## Experience\n\n"+long, 1, longSentenceWords))[ProseLongSentence]; got != 1 {
		t.Errorf("expected long_sentence, got %d", got)
	}
}

func TestCheckProse_SkipsSkillsAndEducation(t *testing.T) {
	resume := `## Skills

- **Backend & Data:** REST, GraphQL, PostgreSQL, Redis, Snowflake, SQS

## Education

**General Assembly** — Software Engineering Immersive, New York, NY
`
	if issues := checkProse(resume, 1, longSentenceWords); len(issues) != 0 {
		t.Fatalf("skills/education must not be checked as prose, got %+v", issues)
	}
}

// The reported count must match what the em-dash rule gates on. Counting the
// raw document made a compliant resume report 10 em-dashes because every job
// header carries one as a structural separator.
func TestCountProseEmDashes_IgnoresStructuralHeaders(t *testing.T) {
	resume := `## Summary

I built one thing — and shipped it.

## Experience

### LegalZoom — Remote

**Staff Software Engineer (SE IV)** — Aug 2025 - Aug 2026
**Senior Software Engineer (SE III)** — Aug 2023 - Aug 2025

- Built the Tasks API from zero, then shipped it to production.

## Education

**General Assembly** — Software Engineering Immersive, New York, NY
`
	if got := countProseEmDashes(resume); got != 1 {
		t.Errorf("expected 1 prose em-dash, got %d", got)
	}
	// And the gate agrees: 1 prose em-dash is within a limit of 1.
	if got := typesOf(checkProse(resume, 1, longSentenceWords))[ProseEmDash]; got != 0 {
		t.Errorf("expected no em_dash issue, got %d", got)
	}
}

func TestSplitSentences_DoesNotSplitDecimals(t *testing.T) {
	got := splitSentences("Cut build times by 1.5x across the fleet. Then shipped it.")
	if len(got) != 2 {
		t.Fatalf("expected 2 sentences, got %d: %q", len(got), got)
	}
}

func TestWordsOf_SplitsHyphens(t *testing.T) {
	got := wordsOf("Co-led the 50-62% rollout")
	want := map[string]bool{"co": true, "led": true, "the": true, "rollout": true}
	for _, w := range got {
		if !want[w] {
			t.Errorf("unexpected token %q in %q", w, got)
		}
	}
	if !hasFiniteVerb(got) {
		t.Error("co-led should register a finite verb")
	}
}
