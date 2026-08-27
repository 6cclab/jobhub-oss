package eval

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestScoreTypes(t *testing.T) {
	assert.Equal(t, Pass, Pass)
	assert.Equal(t, Warn, Warn)
	assert.Equal(t, Fail, Fail)
}

func TestVerdictStrong(t *testing.T) {
	assert.Equal(t, Strong, Strong)
}

func TestVerdictAcceptable(t *testing.T) {
	assert.Equal(t, Acceptable, Acceptable)
}

func TestVerdictNeedsRework(t *testing.T) {
	assert.Equal(t, NeedsRework, NeedsRework)
}

func TestVerdictCritical(t *testing.T) {
	assert.Equal(t, Critical, Critical)
}

func TestTermCategoryExplicit(t *testing.T) {
	assert.Equal(t, ExplicitSkill, ExplicitSkill)
}

func TestTermCategoryImplicit(t *testing.T) {
	assert.Equal(t, ImplicitCapability, ImplicitCapability)
}

func TestTermCategoryDomain(t *testing.T) {
	assert.Equal(t, DomainKeyword, DomainKeyword)
}

func TestTermCategorySeniority(t *testing.T) {
	assert.Equal(t, SenioritySignal, SenioritySignal)
}

func TestTermPriorityTop3(t *testing.T) {
	assert.Equal(t, Top3, Top3)
}

func TestTermPriorityStandard(t *testing.T) {
	assert.Equal(t, Standard, Standard)
}

func TestTermStatusCovered(t *testing.T) {
	assert.Equal(t, Covered, Covered)
}

func TestTermStatusSkillsOnly(t *testing.T) {
	assert.Equal(t, SkillsOnly, SkillsOnly)
}

func TestTermStatusMissing(t *testing.T) {
	assert.Equal(t, Missing, Missing)
}
