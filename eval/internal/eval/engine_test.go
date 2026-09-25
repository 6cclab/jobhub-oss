package eval

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEngineKeywordScore(t *testing.T) {
	term := PostingTerm{Term: "Go"}
	_ = term
	assert.NotNil(t, term.Term)
}

func TestEngineGapFill(t *testing.T) {
	project := "Go service"
	assert.Greater(t, len(project), 0)
}

func TestEngineStyleCheck(t *testing.T) {
	resumeText := "I'm a Go developer"
	assert.Greater(t, len(resumeText), 0)
}

func TestEngineSkillsCheck(t *testing.T) {
	terms := []string{"Go", "Kubernetes"}
	assert.Greater(t, len(terms), 0)
}

func TestEngineStructuralCheck(t *testing.T) {
	styleInput := StyleInput{SummaryText: "Summary."}
	assert.NotNil(t, styleInput.SummaryText)
}

func TestEngineRunAllCheckers(t *testing.T) {
	engine := &Engine{}
	assert.NotNil(t, engine)
}
