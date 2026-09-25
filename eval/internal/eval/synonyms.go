package eval

import "strings"

var synonymGroups = [][]string{
	{"kubernetes", "k8s"},
	{"ci/cd", "continuous integration", "continuous deployment", "cicd", "ci cd"},
	{"javascript", "js"},
	{"typescript", "ts"},
	{"postgresql", "postgres"},
	{"mongodb", "mongo", "mongodb atlas"},
	{"graphql", "gql"},
	{"infrastructure as code", "iac"},
	{"ecs", "elastic container service"},
	{"opentelemetry", "otel"},
	{"argo rollouts", "argocd", "argo cd", "argo"},
	{"amazon web services", "aws"},
	{"developer experience", "devex", "dx", "developer productivity"},
	{"model context protocol", "mcp"},
	{"react router", "remix"},
	{"node.js", "nodejs", "node"},
	{"github actions", "gh actions"},
	{"machine learning", "ml"},
	{"artificial intelligence", "ai"},
	{"event-driven", "event driven", "event-driven architecture"},
	{"distributed systems", "distributed computing"},
	{"test infrastructure", "test frameworks", "testing infrastructure", "test tooling"},
	{"development environments", "dev environments", "developer environments"},
	{"security hardening", "security posture", "security practices"},
	{"containerized", "containerization", "containers"},
}

// synonymMap maps every term to its canonical group index for fast lookup.
var synonymMap map[string]int

func init() {
	synonymMap = make(map[string]int)
	for i, group := range synonymGroups {
		for _, term := range group {
			synonymMap[strings.ToLower(term)] = i
		}
	}
}

// Synonyms returns all synonyms for a given term (including the term itself).
// Returns nil if the term has no registered synonyms.
func Synonyms(term string) []string {
	idx, ok := synonymMap[strings.ToLower(term)]
	if !ok {
		return nil
	}
	return synonymGroups[idx]
}

// AreSynonyms reports whether two terms are in the same synonym group.
func AreSynonyms(a, b string) bool {
	idxA, okA := synonymMap[strings.ToLower(a)]
	idxB, okB := synonymMap[strings.ToLower(b)]
	return okA && okB && idxA == idxB
}
