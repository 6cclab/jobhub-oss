package eval

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCheckAITellsNoMatches(t *testing.T) {
	resumeText := `SUMMARY
I'm a Go developer with 5 years of experience.
EXPERIENCE
### Acme Corp
- Built REST API endpoints
- Wrote unit tests
`

	result := checkAITells(resumeText, nil)

	assert.Equal(t, Pass, result.Score)
	assert.Empty(t, result.Tells)
}

func TestCheckAITellsEmptyText(t *testing.T) {
	result := checkAITells("", nil)

	assert.Equal(t, Pass, result.Score)
	assert.Empty(t, result.Tells)
}

func TestCheckAITellsWhitespaceOnly(t *testing.T) {
	result := checkAITells("   \n\n   ", nil)

	assert.Equal(t, Pass, result.Score)
	assert.Empty(t, result.Tells)
}
