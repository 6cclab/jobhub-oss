package eval

import (
	"strings"
	"unicode"
)

// ProseIssue is a single sentence-level defect found in the resume body.
type ProseIssue struct {
	Type    string `json:"type"`
	Section string `json:"section"`
	Text    string `json:"text"`
	Detail  string `json:"detail"`
}

// Prose issue types.
const (
	ProseFragment     = "fragment"
	ProseEmDash       = "em_dash"
	ProseRepeatedWord = "repeated_word"
	ProseDoubleSpace  = "double_space"
	ProseLongSentence = "long_sentence"
)

// longSentenceWords is the word count above which a sentence is flagged as hard
// to scan. Recruiters skim; a 45-word sentence in a bullet does not get read.
const longSentenceWords = 45

// minWordsForFragmentCheck skips headers and stub lines, which are not prose and
// legitimately have no verb.
const minWordsForFragmentCheck = 4

// finiteVerbs are verbs that carry tense. A resume bullet with none of these and
// no -ed form is almost always a fragment. Bare present forms ("support",
// "design") are deliberately excluded: resume bullets are written in past tense,
// so a bare form is far more likely to be a noun ("1:1 support") than a verb.
var finiteVerbs = map[string]bool{
	// auxiliaries and copulas
	"is": true, "are": true, "was": true, "were": true, "be": true, "been": true,
	"am": true, "has": true, "have": true, "had": true, "can": true, "could": true,
	// stative/mental present-tense verbs, common in first-person summaries and
	// almost never nouns, so they do not weaken fragment detection in bullets
	"like": true, "likes": true, "love": true, "loves": true, "enjoy": true,
	"enjoys": true, "care": true, "cares": true, "tend": true, "tends": true,
	"prefer": true, "prefers": true, "want": true, "wants": true, "know": true,
	"knows": true, "think": true, "thinks": true, "believe": true,
	"believes": true, "seem": true, "seems": true, "become": true,
	"becomes": true, "remain": true, "remains": true,
	"will": true, "would": true, "should": true, "may": true, "might": true,
	"must": true, "do": true, "does": true, "did": true,
	// irregular past tenses common in engineering resumes
	"built": true, "led": true, "ran": true, "wrote": true, "made": true,
	"took": true, "drove": true, "found": true, "taught": true, "cut": true,
	"put": true, "set": true, "got": true, "gave": true, "held": true,
	"kept": true, "left": true, "sent": true, "spent": true, "told": true,
	"won": true, "brought": true, "bought": true, "caught": true, "chose": true,
	"came": true, "went": true, "grew": true, "knew": true, "met": true,
	"paid": true, "read": true, "said": true, "saw": true, "sold": true,
	"showed": true, "sat": true, "stood": true, "understood": true,
	"began": true, "broke": true, "brought ": true, "dealt": true, "drew": true,
	"fell": true, "felt": true, "flew": true, "forgot": true, "hit": true,
	"lost": true, "rose": true, "sought": true, "spoke": true, "struck": true,
	"swept": true, "threw": true, "wound": true,
}

// bodySections are the markdown sections whose bullets are real prose. Skills
// and Education are label-style lists and are correctly verbless.
var bodySections = map[string]bool{
	"experience":        true,
	"selected projects": true,
	"projects":          true,
	"summary":           true,
}

// checkProse runs sentence-level quality checks over the resume body. It is
// deliberately conservative: every rule here is meant to have near-zero false
// positives on well-written resumes, because a noisy prose check trains the
// author to ignore it.
func checkProse(fullText string, maxEmDashes, maxSentenceWords int) []ProseIssue {
	var issues []ProseIssue
	proseEmDashes := 0

	section := ""
	for _, raw := range strings.Split(fullText, "\n") {
		line := strings.TrimSpace(raw)

		// Only level-2 headings name a section. Level 3+ are company/role
		// headers inside a section and must not clobber it.
		if strings.HasPrefix(line, "#") {
			if strings.HasPrefix(line, "## ") && !strings.HasPrefix(line, "### ") {
				section = strings.ToLower(strings.TrimSpace(strings.TrimLeft(line, "# ")))
			}
			continue
		}
		if line == "" || strings.HasPrefix(line, "---") {
			continue
		}
		if !bodySections[section] {
			continue
		}

		text := line
		if strings.HasPrefix(text, "- ") || strings.HasPrefix(text, "* ") {
			text = text[2:]
		} else if section != "summary" {
			// Non-bullet lines inside Experience are job headers and dates.
			continue
		}
		text = stripMarkdown(text)
		if isLabelLine(text) {
			continue
		}

		// Count em-dashes only in prose. Job headers use them as structural
		// separators ("LegalZoom — Remote") and the rendered PDF lays those out
		// in columns, so they never reach the reader as punctuation.
		proseEmDashes += strings.Count(text, "—")

		issues = append(issues, checkSentences(text, section, maxSentenceWords)...)
	}

	if proseEmDashes > maxEmDashes {
		issues = append(issues, ProseIssue{
			Type:   ProseEmDash,
			Detail: plural(proseEmDashes, "em-dash", "em-dashes") + " in prose; style allows " + itoa(maxEmDashes) + ". More than that reads as a crutch.",
		})
	}

	return issues
}

// countProseEmDashes counts em-dashes in prose only, matching what the em-dash
// rule in checkProse actually gates on. The whole-document count is misleading:
// job headers use em-dashes as structural separators ("LegalZoom — Remote") and
// the rendered PDF lays those out in columns, so a resume with zero em-dashes in
// its writing still reports eight or ten if you count the raw text.
func countProseEmDashes(fullText string) int {
	n := 0
	for _, line := range extractProse(fullText) {
		n += strings.Count(line, "—")
	}
	return n
}

func checkSentences(text, section string, maxSentenceWords int) []ProseIssue {
	var issues []ProseIssue

	if strings.Contains(text, "  ") {
		issues = append(issues, ProseIssue{
			Type:    ProseDoubleSpace,
			Section: section,
			Text:    snippet(text),
			Detail:  "Double space inside a line.",
		})
	}

	for _, sentence := range splitSentences(text) {
		words := wordsOf(sentence)
		if len(words) == 0 {
			continue
		}

		if dup := repeatedWord(words); dup != "" {
			issues = append(issues, ProseIssue{
				Type:    ProseRepeatedWord,
				Section: section,
				Text:    snippet(sentence),
				Detail:  "Repeated word: \"" + dup + " " + dup + "\".",
			})
		}

		if len(words) > maxSentenceWords {
			issues = append(issues, ProseIssue{
				Type:    ProseLongSentence,
				Section: section,
				Text:    snippet(sentence),
				Detail:  itoa(len(words)) + " words. Long sentences do not survive a skim; split it.",
			})
		}

		if len(words) >= minWordsForFragmentCheck && !hasFiniteVerb(words) {
			issues = append(issues, ProseIssue{
				Type:    ProseFragment,
				Section: section,
				Text:    snippet(sentence),
				Detail:  "No finite verb — reads as a sentence fragment.",
			})
		}
	}

	return issues
}

// hasFiniteVerb reports whether any word carries tense: an -ed form, or a known
// irregular past tense or auxiliary.
func hasFiniteVerb(words []string) bool {
	for _, w := range words {
		if finiteVerbs[w] {
			return true
		}
		if len(w) > 3 && strings.HasSuffix(w, "ed") {
			return true
		}
	}
	return false
}

func repeatedWord(words []string) string {
	for i := 1; i < len(words); i++ {
		if words[i] == words[i-1] && len(words[i]) > 2 {
			return words[i]
		}
	}
	return ""
}

// splitSentences splits on sentence-final punctuation. It avoids splitting on
// decimals and common abbreviations by requiring the following character to be a
// space and the next word to look like a new sentence.
func splitSentences(text string) []string {
	var out []string
	start := 0
	runes := []rune(text)
	for i := 0; i < len(runes); i++ {
		if runes[i] != '.' && runes[i] != '!' && runes[i] != '?' {
			continue
		}
		if i+1 < len(runes) && !unicode.IsSpace(runes[i+1]) {
			continue // decimal, version number, or abbreviation like "1.5x"
		}
		s := strings.TrimSpace(string(runes[start : i+1]))
		if s != "" {
			out = append(out, s)
		}
		start = i + 1
	}
	if tail := strings.TrimSpace(string(runes[start:])); tail != "" {
		out = append(out, tail)
	}
	return out
}

// wordsOf lowercases and splits on any non-letter, so "co-led" yields "co","led"
// and "50-62%" drops out entirely.
func wordsOf(s string) []string {
	fields := strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return !unicode.IsLetter(r)
	})
	out := fields[:0]
	for _, f := range fields {
		if f != "" {
			out = append(out, f)
		}
	}
	return out
}

func stripMarkdown(s string) string {
	s = strings.ReplaceAll(s, "**", "")
	s = strings.ReplaceAll(s, "`", "")
	return strings.TrimSpace(s)
}

// isLabelLine matches "Languages: TypeScript, Python" style entries, which are
// lists rather than prose and are correctly verbless.
func isLabelLine(s string) bool {
	i := strings.Index(s, ":")
	return i > 0 && i <= 30
}

func snippet(s string) string {
	const max = 90
	s = strings.TrimSpace(s)
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}

func plural(n int, one, many string) string {
	if n == 1 {
		return "1 " + one
	}
	return itoa(n) + " " + many
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	if neg {
		return "-" + string(b)
	}
	return string(b)
}
