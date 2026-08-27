package eval

import (
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config holds every threshold the eval engine scores against.
//
// These are the machine-readable half of user/resume-style.md. Any numeric rule
// stated in that guide belongs here rather than as a constant in the engine —
// a hard-coded threshold silently stops matching the guide the moment the guide
// changes, and the resume is then graded against a rule nobody wrote down.
// The one-page-only length check that scored every deliberate two-page resume
// as a style violation was exactly this failure.
type Config struct {
	CommonStackSkills map[string]bool
	BannedPhrases     []string
	UnverifiedMetrics []string
	MaxEmDashes       int
	AITellPhrases     []string

	// resume-style.md "Formatting & Length"
	MaxSentenceWords int
	OnePageMinChars  int
	OnePageMaxChars  int
	TwoPageMinChars  int
	TwoPageMaxChars  int

	// resume-style.md: "5-7 LegalZoom bullets on one page, 10-13 on two
	// (reordered by relevance), 1-2 per prior role"
	LeadBulletsOnePageMin int
	LeadBulletsOnePageMax int
	LeadBulletsTwoPageMin int
	LeadBulletsTwoPageMax int
	PriorRoleBulletsMax   int
}

// LeadRoleBullets returns the acceptable bullet count for the most recent role
// at the given page target.
func (c Config) LeadRoleBullets(targetPages int) (int, int) {
	if targetPages == 2 {
		return c.LeadBulletsTwoPageMin, c.LeadBulletsTwoPageMax
	}
	return c.LeadBulletsOnePageMin, c.LeadBulletsOnePageMax
}

type yamlConfig struct {
	CommonStackSkills     []string `yaml:"common_stack_skills"`
	BannedPhrases         []string `yaml:"banned_phrases"`
	UnverifiedMetrics     []string `yaml:"unverified_metrics"`
	MaxEmDashes           *int     `yaml:"max_em_dashes"`
	AITellPhrases         []string `yaml:"ai_tell_phrases"`
	MaxSentenceWords      *int     `yaml:"max_sentence_words"`
	LeadBulletsOnePageMin *int     `yaml:"lead_bullets_one_page_min"`
	LeadBulletsOnePageMax *int     `yaml:"lead_bullets_one_page_max"`
	LeadBulletsTwoPageMin *int     `yaml:"lead_bullets_two_page_min"`
	LeadBulletsTwoPageMax *int     `yaml:"lead_bullets_two_page_max"`
	PriorRoleBulletsMax   *int     `yaml:"prior_role_bullets_max"`
	OnePageMinChars       *int     `yaml:"one_page_min_chars"`
	OnePageMaxChars       *int     `yaml:"one_page_max_chars"`
	TwoPageMinChars       *int     `yaml:"two_page_min_chars"`
	TwoPageMaxChars       *int     `yaml:"two_page_max_chars"`
}

// LengthBounds returns the acceptable character range for the given page target.
// Anything other than 2 is treated as one page, which keeps the previous
// behaviour for callers that do not set TargetPages.
func (c Config) LengthBounds(targetPages int) (int, int) {
	if targetPages == 2 {
		return c.TwoPageMinChars, c.TwoPageMaxChars
	}
	return c.OnePageMinChars, c.OnePageMaxChars
}

func DefaultConfig() Config {
	return Config{
		CommonStackSkills: map[string]bool{
			"typescript": true, "javascript": true, "python": true, "go": true,
			"node.js": true, "nodejs": true, "node": true, "ruby": true,
			"java": true, "rust": true, "scala": true, "c++": true, "c#": true,
			"react": true, "angular": true, "vue": true, "svelte": true,
			"css": true, "tailwind": true, "html": true, "es6": true,
			"postgresql": true, "postgres": true, "mysql": true, "mongodb": true,
			"mongo": true, "redis": true, "dynamodb": true, "sqlite": true,
			"snowflake": true,
			"graphql":   true, "rest": true, "grpc": true, "sqs": true,
			"kafka": true, "rabbitmq": true, "apollo server": true, "prisma": true,
			"aws": true, "gcp": true, "azure": true, "docker": true,
			"kubernetes": true, "k8s": true, "terraform": true, "ansible": true,
			"ci/cd": true, "github actions": true, "gitlab ci": true,
			"argo rollouts": true, "argocd": true, "gitops": true,
			"datadog": true, "grafana": true, "prometheus": true,
			"opentelemetry": true, "otel": true, "newrelic": true,
			"vault": true, "oauth": true, "oidc": true,
			"claude code": true, "mcp": true,
		},
		BannedPhrases: []string{
			"track record of",
			"proven ability to",
			"demonstrated experience in",
			"leveraging",
			"leverage",
			"synergies",
			"passionate about",
			"also builds",
			"results-driven",
			"self-motivated",
			"detail-oriented",
		},
		UnverifiedMetrics: []string{},
		MaxEmDashes:       1,

		// resume-style.md: "Reject anything over 45 words."
		MaxSentenceWords: 45,
		// resume-style.md: "One page by default. Two pages when the posting is
		// broad enough to warrant it." One page holds ~8 items; two pages carry
		// 10-13 LegalZoom bullets and must be at least ~60% full, which puts the
		// floor well above the one-page ceiling rather than adjacent to it.
		OnePageMinChars: 1500,
		OnePageMaxChars: 4500,
		TwoPageMinChars: 4500,
		TwoPageMaxChars: 9000,

		LeadBulletsOnePageMin: 5,
		LeadBulletsOnePageMax: 7,
		LeadBulletsTwoPageMin: 10,
		LeadBulletsTwoPageMax: 13,
		PriorRoleBulletsMax:   2,
	}
}

func LoadConfig(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return DefaultConfig(), err
	}

	var yc yamlConfig
	if err := yaml.Unmarshal(data, &yc); err != nil {
		return DefaultConfig(), err
	}

	cfg := DefaultConfig()

	if len(yc.CommonStackSkills) > 0 {
		cfg.CommonStackSkills = make(map[string]bool, len(yc.CommonStackSkills))
		for _, s := range yc.CommonStackSkills {
			cfg.CommonStackSkills[strings.ToLower(s)] = true
		}
	}
	if len(yc.BannedPhrases) > 0 {
		cfg.BannedPhrases = yc.BannedPhrases
	}
	if yc.UnverifiedMetrics != nil {
		cfg.UnverifiedMetrics = yc.UnverifiedMetrics
	}
	if yc.MaxEmDashes != nil {
		cfg.MaxEmDashes = *yc.MaxEmDashes
	}
	if len(yc.AITellPhrases) > 0 {
		cfg.AITellPhrases = yc.AITellPhrases
	}
	for _, o := range []struct {
		src *int
		dst *int
	}{
		{yc.MaxSentenceWords, &cfg.MaxSentenceWords},
		{yc.OnePageMinChars, &cfg.OnePageMinChars},
		{yc.OnePageMaxChars, &cfg.OnePageMaxChars},
		{yc.TwoPageMinChars, &cfg.TwoPageMinChars},
		{yc.TwoPageMaxChars, &cfg.TwoPageMaxChars},
		{yc.LeadBulletsOnePageMin, &cfg.LeadBulletsOnePageMin},
		{yc.LeadBulletsOnePageMax, &cfg.LeadBulletsOnePageMax},
		{yc.LeadBulletsTwoPageMin, &cfg.LeadBulletsTwoPageMin},
		{yc.LeadBulletsTwoPageMax, &cfg.LeadBulletsTwoPageMax},
		{yc.PriorRoleBulletsMax, &cfg.PriorRoleBulletsMax},
	} {
		if o.src != nil {
			*o.dst = *o.src
		}
	}

	return cfg, nil
}
