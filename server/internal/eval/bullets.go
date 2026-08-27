package eval

import (
	"fmt"
	"strings"
)

// RoleBullets is the bullet count for one role heading under ## Experience.
type RoleBullets struct {
	Heading string `json:"heading"`
	Bullets int    `json:"bullets"`
	Lead    bool   `json:"lead"`
}

// countRoleBullets walks the ## Experience section and counts bullets under each
// ### role heading, in document order.
//
// The first role encountered is treated as the lead role — the most recent one,
// which carries the bulk of the evidence and has its own count range in
// resume-style.md. This is positional rather than name-matched on purpose: the
// engine should not know which company anyone works for.
func countRoleBullets(fullText string) []RoleBullets {
	var roles []RoleBullets
	inExperience := false

	for _, raw := range strings.Split(fullText, "\n") {
		line := strings.TrimSpace(raw)

		switch {
		case strings.HasPrefix(line, "## "):
			section := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(line, "## ")))
			inExperience = section == "experience"
			continue
		case strings.HasPrefix(line, "### "):
			if inExperience {
				roles = append(roles, RoleBullets{
					Heading: strings.TrimSpace(strings.TrimPrefix(line, "### ")),
					Lead:    len(roles) == 0,
				})
			}
			continue
		case strings.HasPrefix(line, "#"):
			continue
		}

		if inExperience && len(roles) > 0 &&
			(strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ")) {
			roles[len(roles)-1].Bullets++
		}
	}

	return roles
}

// checkBulletCounts enforces resume-style.md: "5-7 LegalZoom bullets on one page,
// 10-13 on two (reordered by relevance), 1-2 per prior role".
//
// This is a separate check from length because the two catch different failures.
// A resume can sit inside the character band and still be badly apportioned —
// nineteen bullets on the current role and one line for everything before it
// reads as a single-job career no matter how many characters it is.
func checkBulletCounts(fullText string, targetPages int, cfg Config) ([]RoleBullets, []string) {
	roles := countRoleBullets(fullText)
	if len(roles) == 0 {
		return nil, nil
	}

	leadMin, leadMax := cfg.LeadRoleBullets(targetPages)
	var issues []string

	for _, r := range roles {
		if r.Lead {
			if r.Bullets < leadMin || r.Bullets > leadMax {
				issues = append(issues, fmt.Sprintf(
					"%q has %d bullets; resume-style.md wants %d-%d for the lead role on a %d-page resume",
					r.Heading, r.Bullets, leadMin, leadMax, pagesOrOne(targetPages)))
			}
			continue
		}
		if r.Bullets > cfg.PriorRoleBulletsMax {
			issues = append(issues, fmt.Sprintf(
				"%q has %d bullets; resume-style.md wants at most %d per prior role",
				r.Heading, r.Bullets, cfg.PriorRoleBulletsMax))
		}
	}

	return roles, issues
}

func pagesOrOne(p int) int {
	if p == 2 {
		return 2
	}
	return 1
}
