package eval

import (
	"fmt"
	"strings"
)

type Engine struct {
	cfg Config
}

func New(cfg Config) *Engine {
	return &Engine{cfg: cfg}
}

func (e *Engine) Run(req EvalRequest) EvalResult {
	termResults := matchTerms(req)
	gapFill := checkGapFill(termResults, req.ProjectGaps)
	style := e.checkStyle(req.StyleInput)
	skills := e.checkSkillsRelevance(termResults, req.SkillsList, req.PostingTerms)
	structural := checkStructural(req.StyleInput)
	keyword := scoreKeywords(termResults)
	aiTells := checkAITells(req.StyleInput.FullResumeText, e.cfg.AITellPhrases)

	scores := Scores{
		KeywordCoverage: keyword.Score,
		GapFill:         gapFill.Score,
		Style:           style.Score,
		SkillsRelevance: skills.Score,
		Structural:      structural.Score,
		AITells:         aiTells.Score,
	}

	verdict, primaryIssue := deriveVerdict(scores, gapFill, aiTells)

	return EvalResult{
		ResumeFile:      req.ResumeFile,
		Company:         req.Company,
		RoleTitle:       req.RoleTitle,
		PostingURL:      req.PostingURL,
		EvalMode:        req.EvalMode,
		KeywordCoverage: keyword,
		GapFill:         gapFill,
		Style:           style,
		SkillsRelevance: skills,
		Structural:      structural,
		AITells:         aiTells,
		Scores:          scores,
		Verdict:         verdict,
		PrimaryIssue:    primaryIssue,
	}
}

func matchTerms(req EvalRequest) []TermResult {
	resumeLower := strings.ToLower(req.ResumeText)
	skillsLower := make([]string, len(req.SkillsList))
	for i, s := range req.SkillsList {
		skillsLower[i] = strings.ToLower(s)
	}
	skillsText := strings.Join(skillsLower, " ")

	results := make([]TermResult, 0, len(req.PostingTerms))

	for _, pt := range req.PostingTerms {
		tr := TermResult{
			Term:     pt.Term,
			Category: pt.Category,
			Priority: pt.Priority,
			Verdict:  NotApplicable,
		}

		termLower := strings.ToLower(pt.Term)

		foundInBody := containsTerm(resumeLower, termLower)
		foundInSkills := containsTerm(skillsText, termLower)

		if foundInBody {
			tr.Status = Covered
			matched := pt.Term
			tr.MatchedVia = &matched
		} else if foundInSkills {
			tr.Status = SkillsOnly
			matched := pt.Term + " (skills section only)"
			tr.MatchedVia = &matched
		} else {
			tr.Status = Missing
		}

		results = append(results, tr)
	}

	return results
}

func containsTerm(text, term string) bool {
	if strings.Contains(text, term) {
		return true
	}
	syns := Synonyms(term)
	for _, syn := range syns {
		if strings.Contains(text, strings.ToLower(syn)) {
			return true
		}
	}
	return false
}

func checkGapFill(terms []TermResult, projectGaps map[string][]string) GapFillResult {
	var details []GapFillDetail
	top3Failures := 0

	projectGapsLower := make(map[string][]string, len(projectGaps))
	for k, v := range projectGaps {
		projectGapsLower[strings.ToLower(k)] = v
	}

	for i, tr := range terms {
		if tr.Status == Covered {
			continue
		}

		termLower := strings.ToLower(tr.Term)
		projects := findGapFillProjects(termLower, projectGapsLower)

		if len(projects) > 0 {
			terms[i].Verdict = GapFillFailure
			terms[i].GapFillProject = &projects[0]
			details = append(details, GapFillDetail{
				Term:     tr.Term,
				Project:  projects[0],
				Priority: tr.Priority,
			})
			if tr.Priority == Top3 {
				top3Failures++
			}
		} else {
			terms[i].Verdict = GenuineGap
		}
	}

	score := Pass
	failures := len(details)
	if failures >= 3 || top3Failures > 0 {
		score = Fail
	} else if failures > 0 {
		score = Warn
	}

	return GapFillResult{
		Failures:     failures,
		Top3Failures: top3Failures,
		Score:        score,
		Details:      details,
	}
}

func findGapFillProjects(term string, gaps map[string][]string) []string {
	if projects, ok := gaps[term]; ok {
		return projects
	}

	syns := Synonyms(term)
	for _, syn := range syns {
		if projects, ok := gaps[strings.ToLower(syn)]; ok {
			return projects
		}
	}

	for key, projects := range gaps {
		if strings.Contains(key, term) || strings.Contains(term, key) {
			return projects
		}
	}

	return nil
}

func (e *Engine) checkStyle(input StyleInput) StyleResult {
	summaryLower := strings.ToLower(input.SummaryText)
	resumeLower := strings.ToLower(input.FullResumeText)

	firstPerson := strings.HasPrefix(input.SummaryText, "I ") ||
		strings.HasPrefix(input.SummaryText, "I'")

	var foundBanned []string
	for _, phrase := range e.cfg.BannedPhrases {
		if strings.Contains(resumeLower, strings.ToLower(phrase)) {
			foundBanned = append(foundBanned, phrase)
		}
	}

	var foundMetrics []string
	for _, metric := range e.cfg.UnverifiedMetrics {
		if strings.Contains(resumeLower, strings.ToLower(metric)) {
			foundMetrics = append(foundMetrics, metric)
		}
	}

	_ = summaryLower

	chars := len(input.FullResumeText)
	pages := input.TargetPages
	if pages == 0 {
		pages = 1
	}
	lengthMin, lengthMax := e.cfg.LengthBounds(pages)
	lengthOK := chars >= lengthMin && chars <= lengthMax

	maxEmDashes := e.cfg.MaxEmDashes
	if maxEmDashes == 0 {
		maxEmDashes = 1
	}
	maxSentenceWords := e.cfg.MaxSentenceWords
	if maxSentenceWords == 0 {
		maxSentenceWords = longSentenceWords
	}
	proseIssues := checkProse(input.FullResumeText, maxEmDashes, maxSentenceWords)
	roleBullets, bulletIssues := checkBulletCounts(input.FullResumeText, pages, e.cfg)
	emDashCount := countProseEmDashes(input.FullResumeText)

	score := Pass
	issues := 0
	if !firstPerson {
		score = Fail
	}
	if len(foundBanned) > 0 {
		score = Fail
	}
	if len(foundMetrics) > 0 {
		if score != Fail {
			score = Fail
		}
	}
	if !lengthOK {
		issues++
	}
	issues += len(proseIssues)
	issues += len(bulletIssues)
	if score == Pass && issues > 0 {
		score = Warn
	}

	return StyleResult{
		SummaryFirstPerson: firstPerson,
		BannedPhrasesFound: foundBanned,
		UnverifiedMetrics:  foundMetrics,
		EstimatedChars:     chars,
		LengthOK:           lengthOK,
		TargetPages:        pages,
		LengthMin:          lengthMin,
		LengthMax:          lengthMax,
		EmDashCount:        emDashCount,
		ProseIssues:        proseIssues,
		RoleBullets:        roleBullets,
		BulletIssues:       bulletIssues,
		Score:              score,
	}
}

func (e *Engine) checkSkillsRelevance(terms []TermResult, skillsList []string, postingTerms []PostingTerm) SkillsRelevance {
	var skillsOnlyTerms []string
	for _, tr := range terms {
		if tr.Status == SkillsOnly {
			skillsOnlyTerms = append(skillsOnlyTerms, tr.Term)
		}
	}

	postingTermsLower := make(map[string]bool, len(postingTerms))
	for _, pt := range postingTerms {
		postingTermsLower[strings.ToLower(pt.Term)] = true
		for _, syn := range Synonyms(pt.Term) {
			postingTermsLower[strings.ToLower(syn)] = true
		}
	}

	var unmatchedSkills []string
	for _, skill := range skillsList {
		skillLower := strings.ToLower(skill)
		if postingTermsLower[skillLower] {
			continue
		}
		matched := false
		for _, syn := range Synonyms(skill) {
			if postingTermsLower[strings.ToLower(syn)] {
				matched = true
				break
			}
		}
		if matched {
			continue
		}
		if e.cfg.CommonStackSkills[skillLower] {
			continue
		}
		unmatchedSkills = append(unmatchedSkills, skill)
	}

	soCount := len(skillsOnlyTerms)
	umCount := len(unmatchedSkills)

	score := Pass
	if soCount >= 3 || umCount >= 5 {
		score = Fail
	} else if soCount > 0 || umCount > 2 {
		score = Warn
	}

	return SkillsRelevance{
		SkillsOnlyTerms: skillsOnlyTerms,
		UnmatchedSkills: unmatchedSkills,
		Score:           score,
	}
}

func checkStructural(input StyleInput) StructuralResult {
	titleLower := strings.ToLower(input.TitleLine)
	levelLower := strings.ToLower(input.ExpectedLevel)

	titleMatches := strings.Contains(titleLower, levelLower)

	score := Pass
	if !titleMatches {
		score = Fail
	}
	issues := 0
	if !input.HasEducation {
		issues++
	}
	if !input.HasPriorRoles {
		issues++
	}
	if score == Pass && issues > 0 {
		score = Warn
	}

	return StructuralResult{
		TitleMatches:      titleMatches,
		EducationPresent:  input.HasEducation,
		PriorRolesPresent: input.HasPriorRoles,
		Score:             score,
	}
}

func scoreKeywords(terms []TermResult) KeywordCoverage {
	total := len(terms)
	covered := 0
	skillsOnly := 0
	missing := 0

	for _, tr := range terms {
		switch tr.Status {
		case Covered:
			covered++
		case SkillsOnly:
			skillsOnly++
		case Missing:
			missing++
		}
	}

	pct := 0
	if total > 0 {
		pct = (covered * 100) / total
	}

	score := Pass
	if pct < 60 {
		score = Fail
	} else if pct < 80 {
		score = Warn
	}

	return KeywordCoverage{
		Total:      total,
		Covered:    covered,
		SkillsOnly: skillsOnly,
		Missing:    missing,
		Percentage: pct,
		Score:      score,
		Terms:      terms,
	}
}

func deriveVerdict(scores Scores, gapFill GapFillResult, aiTells AITellsResult) (Verdict, string) {
	if gapFill.Score == Fail && gapFill.Top3Failures > 0 {
		return Critical, fmt.Sprintf(
			"%d gap-fill failures including %d top-3 requirement(s)",
			gapFill.Failures, gapFill.Top3Failures,
		)
	}

	hasFail := scores.KeywordCoverage == Fail ||
		scores.GapFill == Fail ||
		scores.Style == Fail ||
		scores.SkillsRelevance == Fail ||
		scores.Structural == Fail ||
		scores.AITells == Fail

	if hasFail {
		issue := "one or more dimensions failed"
		if scores.GapFill == Fail {
			issue = fmt.Sprintf("%d gap-fill failures", gapFill.Failures)
		} else if scores.Style == Fail {
			issue = "style guide violations"
		} else if scores.KeywordCoverage == Fail {
			issue = "keyword coverage below 60%"
		} else if scores.Structural == Fail {
			issue = "structural issues (title level mismatch)"
		} else if scores.AITells == Fail {
			issue = fmt.Sprintf("reads as AI-generated (%d tell(s), weighted %d)", len(aiTells.Tells), aiTells.WeightedScore)
		}
		return NeedsRework, issue
	}

	warnCount := 0
	for _, s := range []Score{
		scores.KeywordCoverage, scores.GapFill, scores.Style,
		scores.SkillsRelevance, scores.Structural, scores.AITells,
	} {
		if s == Warn {
			warnCount++
		}
	}

	if warnCount >= 3 {
		return NeedsRework, fmt.Sprintf("%d dimensions at warn level", warnCount)
	}
	if warnCount > 0 {
		return Acceptable, ""
	}
	return Strong, ""
}

func (e *Engine) ApplyAdversarialDowngrades(result *EvalResult, refutedTerms []string) {
	refuted := make(map[string]bool, len(refutedTerms))
	for _, t := range refutedTerms {
		refuted[strings.ToLower(t)] = true
	}

	for i, tr := range result.KeywordCoverage.Terms {
		if tr.Status == Covered && refuted[strings.ToLower(tr.Term)] {
			result.KeywordCoverage.Terms[i].Status = SkillsOnly
			via := tr.Term + " (adversarial refuted)"
			result.KeywordCoverage.Terms[i].MatchedVia = &via
		}
	}

	result.KeywordCoverage = scoreKeywords(result.KeywordCoverage.Terms)

	projectGapsFromDetails := make(map[string][]string)
	for _, d := range result.GapFill.Details {
		projectGapsFromDetails[strings.ToLower(d.Term)] = []string{d.Project}
	}
	result.GapFill = checkGapFill(result.KeywordCoverage.Terms, projectGapsFromDetails)

	var skillsOnlyTerms []string
	for _, tr := range result.KeywordCoverage.Terms {
		if tr.Status == SkillsOnly {
			skillsOnlyTerms = append(skillsOnlyTerms, tr.Term)
		}
	}
	result.SkillsRelevance.SkillsOnlyTerms = skillsOnlyTerms
	soCount := len(skillsOnlyTerms)
	umCount := len(result.SkillsRelevance.UnmatchedSkills)
	result.SkillsRelevance.Score = Pass
	if soCount >= 3 || umCount >= 5 {
		result.SkillsRelevance.Score = Fail
	} else if soCount > 0 || umCount > 2 {
		result.SkillsRelevance.Score = Warn
	}

	result.Scores = Scores{
		KeywordCoverage: result.KeywordCoverage.Score,
		GapFill:         result.GapFill.Score,
		Style:           result.Style.Score,
		SkillsRelevance: result.SkillsRelevance.Score,
		Structural:      result.Structural.Score,
		AITells:         result.AITells.Score,
	}
	result.Verdict, result.PrimaryIssue = deriveVerdict(result.Scores, result.GapFill, result.AITells)
}
