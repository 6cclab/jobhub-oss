package eval

import (
	"math"
	"regexp"
	"strings"
)

// AITell is a single detected marker of machine-generated prose.
type AITell struct {
	Type   string `json:"type"`
	Text   string `json:"text"`
	Detail string `json:"detail"`
	Weight int    `json:"weight"`
}

// AI tell types.
const (
	AITellPhrase        = "llm_phrase"
	AITellParticiple    = "participle_cascade"
	AITellNotJust       = "not_just_construction"
	AITellTriad         = "rule_of_three"
	AITellAdverbPadding = "adverb_padding"
	AITellVagueMetric   = "vague_quantifier"
	AITellUniformity    = "sentence_uniformity"
)

// AITellsResult summarizes how machine-generated the prose reads.
type AITellsResult struct {
	Tells         []AITell `json:"tells"`
	WeightedScore int      `json:"weighted_score"`
	SentenceCount int      `json:"sentence_count"`
	LengthCV      float64  `json:"sentence_length_cv"`
	Score         Score    `json:"score"`
}

// Thresholds for the weighted tell score. Tuned so a resume written by a person
// lands at 0-2, light AI polish lands in the warn band, and unedited model
// output lands in fail.
const (
	aiWarnThreshold = 4
	aiFailThreshold = 9
)

// uniformityMinSentences is the sample size below which sentence-length variance
// is statistical noise rather than signal.
const uniformityMinSentences = 6

// uniformityCVThreshold is the coefficient of variation below which sentence
// lengths are suspiciously even. Human resume prose typically sits at 0.35-0.6;
// model output clusters tightly because it targets a consistent rhythm.
const uniformityCVThreshold = 0.22

// llmPhrases are lexical markers that appear far more often in generated prose
// than in writing by an engineer describing their own work. Kept separate from
// BannedPhrases, which is about corporate filler the user personally dislikes.
var llmPhrases = []string{
	"delve", "seamless", "seamlessly", "robust", "cutting-edge",
	"state-of-the-art", "best-in-class", "world-class", "spearheaded",
	"pivotal", "meticulous", "meticulously", "underscore", "underscores",
	"testament", "tapestry", "realm", "landscape of", "showcase",
	"showcasing", "navigate the complexities", "in today's",
	"fast-paced", "ever-evolving", "game-changer", "game-changing",
	"unlock", "unlocking", "empower", "empowering", "foster", "fostering",
	"holistic", "synergy", "paradigm", "innovative solutions",
	"wide range of", "deep dive", "at the forefront", "poised to",
	"commitment to excellence", "drive success", "elevate",
}

// participleTails are the "-ing" clause endings that model output chains onto
// bullets: "Built X, enabling Y, resulting in Z, driving W." One is fine and
// idiomatic; three or more in a single bullet is a strong signal.
var participleTails = []string{
	"enabling", "resulting in", "driving", "ensuring", "allowing",
	"empowering", "fostering", "streamlining", "leading to", "culminating in",
	"paving the way", "facilitating", "leveraging", "highlighting",
}

// paddingAdverbs add emphasis without information. Models reach for them to
// signal impact when no number is available.
var paddingAdverbs = []string{
	"successfully", "effectively", "efficiently", "significantly",
	"substantially", "dramatically", "seamlessly", "robustly",
	"consistently", "proactively", "strategically", "meticulously",
}

// vagueQuantifiers claim magnitude without a figure.
var vagueQuantifiers = []string{
	"significantly improved", "substantially reduced", "greatly increased",
	"dramatically improved", "vastly improved", "considerably reduced",
	"markedly improved", "notably increased",
}

var notJustRe = regexp.MustCompile(`(?i)\bnot (just|merely|only)\b[^.]{0,60}\bbut\b`)

// triadRe matches "a, b, and c" lists of single or compound words — the rule of
// three that model prose falls into.
var triadRe = regexp.MustCompile(`(?i)\b[\w-]+, [\w-]+,? and [\w-]+\b`)

// checkAITells scores how much the resume reads as machine-generated.
//
// This detects *stylistic* markers, not provenance. A human can write this way
// and a model can be prompted out of it. It exists because recruiters and ATS
// screens increasingly flag these patterns, so the cost of carrying them is
// real regardless of who typed them.
func checkAITells(fullText string, extraPhrases []string) AITellsResult {
	var tells []AITell

	proseLines := extractProse(fullText)
	prose := strings.Join(proseLines, " ")
	proseLower := strings.ToLower(prose)

	phrases := append(append([]string{}, llmPhrases...), extraPhrases...)
	seen := map[string]bool{}
	for _, p := range phrases {
		lp := strings.ToLower(p)
		if lp == "" || seen[lp] || !containsWord(proseLower, lp) {
			continue
		}
		seen[lp] = true
		tells = append(tells, AITell{
			Type:   AITellPhrase,
			Text:   p,
			Detail: "Phrase strongly associated with generated prose.",
			Weight: 2,
		})
	}

	for _, line := range proseLines {
		ll := strings.ToLower(line)
		n := 0
		for _, t := range participleTails {
			n += strings.Count(ll, t)
		}
		if n >= 3 {
			tells = append(tells, AITell{
				Type:   AITellParticiple,
				Text:   snippet(line),
				Detail: itoa(n) + " trailing \"-ing\" clauses in one bullet. Chained result clauses are the most recognisable generated-resume pattern.",
				Weight: 3,
			})
		}
	}

	if m := notJustRe.FindString(prose); m != "" {
		tells = append(tells, AITell{
			Type:   AITellNotJust,
			Text:   snippet(m),
			Detail: "\"Not just X but Y\" construction.",
			Weight: 2,
		})
	}

	if triads := triadRe.FindAllString(prose, -1); len(triads) >= 3 {
		tells = append(tells, AITell{
			Type:   AITellTriad,
			Text:   snippet(triads[0]),
			Detail: itoa(len(triads)) + " three-item lists. Repeated triads read as generated rhythm.",
			Weight: 2,
		})
	}

	adverbHits := 0
	var firstAdverb string
	for _, a := range paddingAdverbs {
		if c := strings.Count(proseLower, a); c > 0 {
			adverbHits += c
			if firstAdverb == "" {
				firstAdverb = a
			}
		}
	}
	if adverbHits >= 2 {
		tells = append(tells, AITell{
			Type:   AITellAdverbPadding,
			Text:   firstAdverb,
			Detail: itoa(adverbHits) + " emphasis adverbs. They add no information; a number does the work instead.",
			Weight: 2,
		})
	}

	for _, v := range vagueQuantifiers {
		if strings.Contains(proseLower, v) {
			tells = append(tells, AITell{
				Type:   AITellVagueMetric,
				Text:   v,
				Detail: "Claims magnitude without a figure.",
				Weight: 2,
			})
		}
	}

	var lengths []int
	for _, line := range proseLines {
		for _, s := range splitSentences(line) {
			if w := len(wordsOf(s)); w >= 4 {
				lengths = append(lengths, w)
			}
		}
	}
	cv := coefficientOfVariation(lengths)
	if len(lengths) >= uniformityMinSentences && cv < uniformityCVThreshold {
		tells = append(tells, AITell{
			Type:   AITellUniformity,
			Detail: "Sentence lengths are unusually uniform (CV " + formatCV(cv) + "). Human writing varies more; even rhythm is a generated-text signature.",
			Weight: 3,
		})
	}

	total := 0
	for _, t := range tells {
		total += t.Weight
	}

	score := Pass
	switch {
	case total >= aiFailThreshold:
		score = Fail
	case total >= aiWarnThreshold:
		score = Warn
	}

	return AITellsResult{
		Tells:         tells,
		WeightedScore: total,
		SentenceCount: len(lengths),
		LengthCV:      cv,
		Score:         score,
	}
}

// extractProse returns the summary paragraphs and bullet text — the parts a
// reader judges as writing. Skills lists and headers are excluded.
func extractProse(fullText string) []string {
	var out []string
	section := ""
	for _, raw := range strings.Split(fullText, "\n") {
		line := strings.TrimSpace(raw)
		if strings.HasPrefix(line, "#") {
			if strings.HasPrefix(line, "## ") && !strings.HasPrefix(line, "### ") {
				section = strings.ToLower(strings.TrimSpace(strings.TrimLeft(line, "# ")))
			}
			continue
		}
		if line == "" || strings.HasPrefix(line, "---") || !bodySections[section] {
			continue
		}
		text := line
		if strings.HasPrefix(text, "- ") || strings.HasPrefix(text, "* ") {
			text = text[2:]
		} else if section != "summary" {
			continue
		}
		text = stripMarkdown(text)
		if isLabelLine(text) {
			continue
		}
		out = append(out, text)
	}
	return out
}

// containsWord matches on word boundaries so "elevate" does not fire inside
// "elevated" and "realm" does not fire inside "realms of".
func containsWord(haystack, needle string) bool {
	if strings.ContainsAny(needle, " -'") {
		return strings.Contains(haystack, needle)
	}
	idx := 0
	for {
		i := strings.Index(haystack[idx:], needle)
		if i < 0 {
			return false
		}
		i += idx
		beforeOK := i == 0 || !isWordByte(haystack[i-1])
		end := i + len(needle)
		afterOK := end >= len(haystack) || !isWordByte(haystack[end])
		if beforeOK && afterOK {
			return true
		}
		idx = i + 1
	}
}

func isWordByte(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9')
}

func coefficientOfVariation(xs []int) float64 {
	if len(xs) < 2 {
		return 0
	}
	sum := 0
	for _, x := range xs {
		sum += x
	}
	mean := float64(sum) / float64(len(xs))
	if mean == 0 {
		return 0
	}
	varsum := 0.0
	for _, x := range xs {
		d := float64(x) - mean
		varsum += d * d
	}
	return math.Sqrt(varsum/float64(len(xs))) / mean
}

func formatCV(f float64) string {
	scaled := int(math.Round(f * 100))
	return "0." + pad2(scaled)
}

func pad2(n int) string {
	if n < 10 {
		return "0" + itoa(n)
	}
	return itoa(n)
}
