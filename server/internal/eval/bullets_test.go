package eval

import "testing"

// The heading block of a real resume: bold role/date lines sit between the
// company heading and the bullets. They start with '*' and must not be counted
// as bullets — a naive prefix check reads "**Staff Software Engineer**" as one
// and silently inflates every count by the number of titles held.
const resumeFixture = `# Andre Pato

## Summary

I build things.

## Experience

### LegalZoom — Remote

**Staff Software Engineer (SE IV)** — Aug 2025 - Aug 2026
**Senior Software Engineer (SE III)** — Aug 2023 - Aug 2025

- Built the Tasks API from zero.
- Diagnosed pod liveness failures under load.
- Authored the ADR for a Relay Proxy.
- Built a developer-experience observability platform.

### General Assembly — New York, NY

- Taught full-stack development.

## Skills

- TypeScript
- Go
`

func TestCountRoleBulletsIgnoresBoldHeadingLines(t *testing.T) {
	roles := countRoleBullets(resumeFixture)

	if len(roles) != 2 {
		t.Fatalf("found %d roles, want 2: %+v", len(roles), roles)
	}
	if roles[0].Heading != "LegalZoom — Remote" || !roles[0].Lead {
		t.Errorf("first role should be the lead: %+v", roles[0])
	}
	if roles[0].Bullets != 4 {
		t.Errorf("lead role counted %d bullets, want 4 — bold role/date lines must not count",
			roles[0].Bullets)
	}
	if roles[1].Bullets != 1 || roles[1].Lead {
		t.Errorf("prior role: got %+v, want 1 bullet and Lead=false", roles[1])
	}
}

// Skills is also a bullet list. Counting it would attribute its entries to
// whichever role happened to be last.
func TestCountRoleBulletsStopsAtEndOfExperience(t *testing.T) {
	roles := countRoleBullets(resumeFixture)
	total := 0
	for _, r := range roles {
		total += r.Bullets
	}
	if total != 5 {
		t.Errorf("counted %d bullets across all roles, want 5 — Skills must be excluded", total)
	}
}

func TestCheckBulletCountsFlagsThinAndFatLeadRoles(t *testing.T) {
	cfg := DefaultConfig()

	// 4 bullets is under the one-page floor of 5.
	_, issues := checkBulletCounts(resumeFixture, 1, cfg)
	if len(issues) != 1 {
		t.Fatalf("want 1 issue for a 4-bullet lead role on one page, got %d: %v", len(issues), issues)
	}

	// The same resume judged as two pages is further out of range, not closer.
	_, twoPage := checkBulletCounts(resumeFixture, 2, cfg)
	if len(twoPage) != 1 {
		t.Errorf("want 1 issue at two pages too, got %d: %v", len(twoPage), twoPage)
	}
}

func TestCheckBulletCountsFlagsOverlongPriorRole(t *testing.T) {
	cfg := DefaultConfig()
	fixture := `## Experience

### Current Co

- one
- two
- three
- four
- five

### Prior Co

- a
- b
- c
`
	_, issues := checkBulletCounts(fixture, 1, cfg)
	if len(issues) != 1 {
		t.Fatalf("want 1 issue for a 3-bullet prior role, got %d: %v", len(issues), issues)
	}
}

func TestCheckBulletCountsSilentOnAConformingResume(t *testing.T) {
	cfg := DefaultConfig()
	fixture := `## Experience

### Current Co

- one
- two
- three
- four
- five

### Prior Co

- a
`
	if _, issues := checkBulletCounts(fixture, 1, cfg); len(issues) != 0 {
		t.Errorf("conforming resume produced issues: %v", issues)
	}
}

func TestCheckBulletCountsNoExperienceSection(t *testing.T) {
	cfg := DefaultConfig()
	roles, issues := checkBulletCounts("# Name\n\n## Summary\n\nHello.\n", 1, cfg)
	if roles != nil || issues != nil {
		t.Errorf("a resume with no Experience section should produce nothing, got %v / %v", roles, issues)
	}
}
