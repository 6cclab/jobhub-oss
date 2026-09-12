// Command backfill migrates existing flat-file job-search data
// (applications.md, boards.md, research/*.md) into a running JobHub
// server via its HTTP API.
//
// This is a one-time migration script, not production code. It is
// deliberately simple and best-effort: it prints progress as it goes
// and keeps going on individual failures so a single bad row doesn't
// abort the whole run.
//
// Usage:
//
//	go run ./cmd/backfill
//
// The target server URL defaults to http://localhost:8080 and can be
// overridden with the JOBHUB_URL environment variable.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var (
	apiToken     string
	jobSearchDir string
)

func main() {
	jobSearchDir = os.Getenv("JOBHUB_DATA_DIR")
	if jobSearchDir == "" {
		jobSearchDir = "."
	}

	baseURL := os.Getenv("JOBHUB_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	baseURL = strings.TrimRight(baseURL, "/")

	apiToken = os.Getenv("JOBHUB_API_TOKEN")
	client := &http.Client{Timeout: 15 * time.Second}

	fmt.Printf("Backfilling into %s\n\n", baseURL)

	boardsOK := backfillBoards(client, baseURL)
	researchOK := backfillResearch(client, baseURL)
	appsOK := backfillApplications(client, baseURL)

	fmt.Println()
	fmt.Printf("Summary: %d boards, %d research briefs, %d applications imported\n",
		boardsOK, researchOK, appsOK)
}

// postJSON POSTs the given payload as JSON to path and returns the status
// code and raw response body (for error reporting).
func postJSON(client *http.Client, baseURL, path string, payload interface{}) (int, []byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return 0, nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, baseURL+path, bytes.NewReader(body))
	if err != nil {
		return 0, nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if apiToken != "" {
		req.Header.Set("Authorization", "Bearer "+apiToken)
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(resp.Body); err != nil {
		return resp.StatusCode, nil, fmt.Errorf("read response: %w", err)
	}

	return resp.StatusCode, buf.Bytes(), nil
}

// ---------------------------------------------------------------------
// Boards
// ---------------------------------------------------------------------

type createBoardRequest struct {
	Slug   string   `json:"slug"`
	Name   string   `json:"name"`
	ATS    string   `json:"ats"`
	Tags   []string `json:"tags"`
	Status string   `json:"status"`
}

func backfillBoards(client *http.Client, baseURL string) int {
	path := filepath.Join(jobSearchDir, "boards.md")
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Printf("[boards] FAILED to read %s: %v\n", path, err)
		return 0
	}

	sections := map[string]string{
		"### Greenhouse": "greenhouse",
		"### Ashby":      "ashby",
		"### Lever":      "lever",
	}

	lines := strings.Split(string(data), "\n")

	var currentATS string
	count := 0

	for _, rawLine := range lines {
		line := strings.TrimSpace(rawLine)

		// Stop parsing boards once we hit the Discovery List section.
		if strings.HasPrefix(line, "## Discovery List") {
			break
		}

		if ats, ok := sections[line]; ok {
			currentATS = ats
			continue
		}
		// Any other heading (## or ###) that isn't a tracked ATS section
		// resets the current ATS so we don't misparse stray lines.
		if strings.HasPrefix(line, "#") {
			if _, ok := sections[line]; !ok {
				currentATS = ""
			}
			continue
		}

		if currentATS == "" || line == "" {
			continue
		}

		// Expected format: slug | Company Name | tag1, tag2, tag3
		parts := strings.Split(line, "|")
		if len(parts) < 2 {
			continue
		}
		slug := strings.TrimSpace(parts[0])
		name := strings.TrimSpace(parts[1])
		if slug == "" || name == "" {
			continue
		}

		var tags []string
		if len(parts) >= 3 {
			tagsRaw := strings.TrimSpace(parts[2])
			for _, t := range strings.Split(tagsRaw, ",") {
				t = strings.TrimSpace(t)
				if t != "" {
					tags = append(tags, t)
				}
			}
		}

		req := createBoardRequest{
			Slug:   slug,
			Name:   name,
			ATS:    currentATS,
			Tags:   tags,
			Status: "tracked",
		}

		status, body, err := postJSON(client, baseURL, "/api/boards", req)
		if err != nil {
			fmt.Printf("[board] FAILED %s (%s): %v\n", name, slug, err)
			continue
		}
		if status >= 200 && status < 300 {
			fmt.Printf("[board] OK %s (%s, %s)\n", name, slug, currentATS)
			count++
		} else {
			fmt.Printf("[board] FAILED %s (%s): HTTP %d: %s\n", name, slug, status, strings.TrimSpace(string(body)))
		}
	}

	return count
}

// ---------------------------------------------------------------------
// Research briefs
// ---------------------------------------------------------------------

type createResearchRequest struct {
	Company          string   `json:"company"`
	StabilityVerdict *string  `json:"stability_verdict"`
	RemotePolicy     *string  `json:"remote_policy"`
	SalaryRangeText  *string  `json:"salary_range_text"`
	TechStack        []string `json:"tech_stack,omitempty"`
	ResearchedAt     string   `json:"researched_at"`
	RawMarkdown      string   `json:"raw_markdown"`
}

var (
	stabilityVerdictRe = regexp.MustCompile(`(?i)verdict:?\s*(Strong|Stable|Caution|Avoid)`)
	remotePolicyLineRe = regexp.MustCompile(`(?im)^\*{0,2}remote policy\*{0,2}:?\*{0,2}\s*:?\s*(.+)$`)
	remoteMentionRe    = regexp.MustCompile(`(?i)([^.\n]*\bremote\b[^.\n]*\.)`)
	dollarAmountRe     = regexp.MustCompile(`\$[\d][\d,]*[Kk]?(?:\s*-\s*\$?[\d][\d,]*[Kk]?)?`)
)

func backfillResearch(client *http.Client, baseURL string) int {
	dir := filepath.Join(jobSearchDir, "research")
	entries, err := os.ReadDir(dir)
	if err != nil {
		fmt.Printf("[research] FAILED to read dir %s: %v\n", dir, err)
		return 0
	}

	count := 0

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		fullPath := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(fullPath)
		if err != nil {
			fmt.Printf("[research] FAILED to read %s: %v\n", entry.Name(), err)
			continue
		}
		content := string(data)

		slug := strings.TrimSuffix(entry.Name(), ".md")
		companyName := slugToTitle(slug)

		req := createResearchRequest{
			Company:          companyName,
			StabilityVerdict: extractStabilityVerdict(content),
			RemotePolicy:     extractRemotePolicy(content),
			SalaryRangeText:  extractSalaryRange(content),
			ResearchedAt:     time.Now().UTC().Format("2006-01-02"),
			RawMarkdown:      content,
		}

		status, body, err := postJSON(client, baseURL, "/api/research", req)
		if err != nil {
			fmt.Printf("[research] FAILED %s: %v\n", companyName, err)
			continue
		}
		if status >= 200 && status < 300 {
			fmt.Printf("[research] OK %s\n", companyName)
			count++
		} else {
			fmt.Printf("[research] FAILED %s: HTTP %d: %s\n", companyName, status, strings.TrimSpace(string(body)))
		}
	}

	return count
}

// extractStabilityVerdict finds a verdict word (Strong/Stable/Caution/Avoid)
// near a "Stability" heading, best-effort.
func extractStabilityVerdict(content string) *string {
	idx := strings.Index(strings.ToLower(content), "## stability")
	section := content
	if idx >= 0 {
		end := idx + 800
		if end > len(content) {
			end = len(content)
		}
		section = content[idx:end]
	}
	m := stabilityVerdictRe.FindStringSubmatch(section)
	if len(m) < 2 {
		return nil
	}
	v := strings.ToLower(m[1])
	return &v
}

// extractRemotePolicy looks for a "Remote policy:" line, falling back to
// the first sentence mentioning "remote" anywhere in the doc.
func extractRemotePolicy(content string) *string {
	if m := remotePolicyLineRe.FindStringSubmatch(content); len(m) >= 2 {
		v := strings.TrimSpace(m[1])
		if v != "" {
			return &v
		}
	}
	if m := remoteMentionRe.FindStringSubmatch(content); len(m) >= 2 {
		v := strings.TrimSpace(m[1])
		if v != "" {
			return &v
		}
	}
	return nil
}

// extractSalaryRange looks for dollar amounts near a Compensation/Salary
// heading, best-effort.
func extractSalaryRange(content string) *string {
	lower := strings.ToLower(content)
	idx := strings.Index(lower, "## compensation")
	if idx < 0 {
		idx = strings.Index(lower, "## salary")
	}
	if idx < 0 {
		return nil
	}
	end := idx + 1500
	if end > len(content) {
		end = len(content)
	}
	section := content[idx:end]

	matches := dollarAmountRe.FindAllString(section, -1)
	if len(matches) == 0 {
		return nil
	}
	// Join a handful of the found amounts/ranges for a best-effort summary.
	limit := len(matches)
	if limit > 6 {
		limit = 6
	}
	joined := strings.Join(matches[:limit], ", ")
	return &joined
}

// slugToTitle converts a hyphenated filename slug into a Title Case company
// name, e.g. "grafana-labs" -> "Grafana Labs".
func slugToTitle(slug string) string {
	parts := strings.Split(slug, "-")
	for i, p := range parts {
		if p == "" {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	return strings.Join(parts, " ")
}

// ---------------------------------------------------------------------
// Applications
// ---------------------------------------------------------------------

type createApplicationRequest struct {
	Company    string  `json:"company"`
	RoleTitle  string  `json:"role_title"`
	Source     *string `json:"source"`
	Status     string  `json:"status"`
	AppliedAt  *string `json:"applied_at"`
	ResumeFile *string `json:"resume_file"`
	Notes      *string `json:"notes"`
}

func backfillApplications(client *http.Client, baseURL string) int {
	path := filepath.Join(jobSearchDir, "applications.md")
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Printf("[applications] FAILED to read %s: %v\n", path, err)
		return 0
	}

	lines := strings.Split(string(data), "\n")
	count := 0
	headerSeen := false

	for _, rawLine := range lines {
		line := strings.TrimSpace(rawLine)
		if !strings.HasPrefix(line, "|") {
			continue
		}

		cols := splitTableRow(line)
		if len(cols) < 7 {
			continue
		}

		// Skip header row.
		if !headerSeen {
			headerSeen = true
			continue
		}
		// Skip separator row, e.g. |------|------|...
		if isSeparatorRow(cols) {
			continue
		}

		date := cols[0]
		company := cols[1]
		role := cols[2]
		source := cols[3]
		status := cols[4]
		resume := cols[5]
		notes := cols[6]

		if company == "" || role == "" {
			continue
		}

		req := createApplicationRequest{
			Company:   company,
			RoleTitle: role,
			Status:    status,
		}
		if source != "" && source != "—" {
			s := source
			req.Source = &s
		}
		if date != "" && strings.ToLower(date) != "unknown" {
			d := date
			req.AppliedAt = &d
		}
		if resume != "" && resume != "—" {
			r := resume
			req.ResumeFile = &r
		}
		if notes != "" {
			n := notes
			req.Notes = &n
		}

		status2, body, err := postJSON(client, baseURL, "/api/applications", req)
		if err != nil {
			fmt.Printf("[application] FAILED %s - %s: %v\n", company, role, err)
			continue
		}
		if status2 >= 200 && status2 < 300 {
			fmt.Printf("[application] OK %s - %s\n", company, role)
			count++
		} else {
			fmt.Printf("[application] FAILED %s - %s: HTTP %d: %s\n", company, role, status2, strings.TrimSpace(string(body)))
		}
	}

	return count
}

// splitTableRow splits a markdown table row like "| a | b | c |" into
// trimmed cell values, dropping the leading/trailing empty cells produced
// by the outer pipes.
func splitTableRow(line string) []string {
	trimmed := strings.Trim(line, "|")
	rawCols := strings.Split(trimmed, "|")
	cols := make([]string, len(rawCols))
	for i, c := range rawCols {
		cols[i] = strings.TrimSpace(c)
	}
	return cols
}

// isSeparatorRow reports whether a table row is a markdown header separator
// row, i.e. every cell consists only of dashes (and optional colons).
func isSeparatorRow(cols []string) bool {
	for _, c := range cols {
		if c == "" {
			continue
		}
		if strings.Trim(c, "-: ") != "" {
			return false
		}
	}
	return true
}
