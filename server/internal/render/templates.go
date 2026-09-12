// Package render provides HTML template rendering for the job search
// dashboard, built on Go's html/template and goldmark for markdown-to-HTML
// conversion of research briefs.
package render

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"path"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
)

// PageData wraps per-page data with common layout information so
// layout.html can render nav state and page titles.
type PageData struct {
	Title       string
	CurrentPath string
	Data        interface{}
}

// Renderer holds parsed templates and a markdown renderer.
//
// Page templates (dashboard.html, fitreports/list.html, etc.) each define
// "title" and "content" blocks that layout.html references via
// {{block "title" .}} / {{block "content" .}}. Because html/template
// resolves same-named templates as "last definition wins" within a single
// template set, each page must be parsed into its OWN template set together
// with layout.html and the shared partials — never all page templates
// together, or they'd stomp on each other's "title"/"content" definitions.
type Renderer struct {
	templatesFS embed.FS
	funcMap     template.FuncMap
	partials    *template.Template // layout.html + partials/*.html, shared base
	pages       map[string]*template.Template
	markdown    goldmark.Markdown
}

// NewRenderer parses all templates from the embedded FS and registers
// template funcs used across the dashboard's views.
func NewRenderer(templatesFS embed.FS) *Renderer {
	funcMap := template.FuncMap{
		"formatDate":     formatDate,
		"upper":          strings.ToUpper,
		"lower":          strings.ToLower,
		"verdictClass":   verdictClass,
		"stabilityClass": stabilityClass,
		"formatSalary":   formatSalary,
		"safeHTML":       safeHTML,
		"splitTags":      splitTagsFn,
		"dict":           dictFn,
	}

	// Base: layout.html + all partials, shared by every page.
	base, err := template.New("").Funcs(funcMap).ParseFS(templatesFS,
		"layout.html",
		"partials/*.html",
	)
	if err != nil {
		panic(fmt.Sprintf("render: parse base templates: %v", err))
	}

	r := &Renderer{
		templatesFS: templatesFS,
		funcMap:     funcMap,
		partials:    base,
		pages:       make(map[string]*template.Template),
		markdown:    goldmark.New(goldmark.WithExtensions(extension.GFM)),
	}

	pageGlobs := []string{
		"dashboard.html",
		"fitreports/*.html",
		"searchresults/*.html",
		"applications/*.html",
		"research/*.html",
		"boards/*.html",
		"evalresults/*.html",
	}
	for _, glob := range pageGlobs {
		matches, err := fsGlob(templatesFS, glob)
		if err != nil {
			panic(fmt.Sprintf("render: glob %q: %v", glob, err))
		}
		for _, name := range matches {
			clone, err := base.Clone()
			if err != nil {
				panic(fmt.Sprintf("render: clone base for %q: %v", name, err))
			}
			if _, err := clone.ParseFS(templatesFS, name); err != nil {
				panic(fmt.Sprintf("render: parse page template %q: %v", name, err))
			}
			r.pages[name] = clone
		}
	}

	return r
}

// fsGlob returns the embed.FS paths matching a glob pattern like
// "fitreports/*.html".
func fsGlob(fsys embed.FS, pattern string) ([]string, error) {
	dir := path.Dir(pattern)
	base := path.Base(pattern)
	entries, err := fsys.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var matches []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ok, err := path.Match(base, e.Name())
		if err != nil {
			return nil, err
		}
		if ok {
			matches = append(matches, path.Join(dir, e.Name()))
		}
	}
	return matches, nil
}

// Page renders a full page: layout.html wraps the named page template's
// "title" and "content" blocks. name is relative to the templates dir, e.g.
// "dashboard.html" or "fitreports/detail.html".
func (r *Renderer) Page(c *fiber.Ctx, name string, data interface{}) error {
	tmpl, ok := r.pages[name]
	if !ok {
		return fmt.Errorf("render: page template %q not found", name)
	}

	pageData := PageData{
		CurrentPath: c.Path(),
		Data:        data,
	}

	buf := &bytes.Buffer{}
	if err := tmpl.ExecuteTemplate(buf, "layout.html", pageData); err != nil {
		return fmt.Errorf("render: execute page template %q: %w", name, err)
	}

	c.Set(fiber.HeaderContentType, fiber.MIMETextHTMLCharsetUTF8)
	return c.SendStream(buf)
}

// Partial renders just the named partial template, with no layout wrapper.
// Used for HTMX fragment responses. name is the template's defined name,
// e.g. "partials/_search_result_row.html".
func (r *Renderer) Partial(c *fiber.Ctx, name string, data interface{}) error {
	buf := &bytes.Buffer{}
	if err := r.partials.ExecuteTemplate(buf, name, data); err != nil {
		return fmt.Errorf("render: execute partial template %q: %w", name, err)
	}

	c.Set(fiber.HeaderContentType, fiber.MIMETextHTMLCharsetUTF8)
	return c.SendStream(buf)
}

// PartialEach renders the named partial template once per element of items
// (a slice), concatenating the output into a single response. Useful for
// HTMX endpoints that return a set of repeated fragments (e.g. table rows)
// using a partial template designed to be invoked once per item.
func (r *Renderer) PartialEach(c *fiber.Ctx, name string, items interface{}) error {
	buf := &bytes.Buffer{}

	v := reflectValueOf(items)
	for i := 0; i < v.Len(); i++ {
		if err := r.partials.ExecuteTemplate(buf, name, v.Index(i).Interface()); err != nil {
			return fmt.Errorf("render: execute partial template %q (item %d): %w", name, i, err)
		}
	}

	c.Set(fiber.HeaderContentType, fiber.MIMETextHTMLCharsetUTF8)
	return c.SendStream(buf)
}

// RenderMarkdown converts markdown to HTML via goldmark.
func (r *Renderer) RenderMarkdown(md string) string {
	var buf bytes.Buffer
	if err := r.markdown.Convert([]byte(md), &buf); err != nil {
		return template.HTMLEscapeString(md)
	}
	return buf.String()
}

// formatDate converts an ISO-8601 timestamp (e.g. "2026-07-08T15:04:05.000Z"
// or "2026-07-08") to a human-friendly "Jan 2, 2006" format. If parsing
// fails, the original string is returned unchanged.
func formatDate(iso string) string {
	if iso == "" {
		return ""
	}
	layouts := []string{
		"2006-01-02T15:04:05.000Z",
		"2006-01-02T15:04:05Z",
		time.RFC3339,
		"2006-01-02",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, iso); err == nil {
			return t.Format("Jan 2, 2006")
		}
	}
	return iso
}

// verdictClass maps a fit report verdict (one of: strong, worth, stretch,
// skip — see fit_reports.verdict CHECK constraint) to a CSS class suffix.
func verdictClass(verdict string) string {
	v := strings.ToLower(strings.TrimSpace(verdict))
	switch v {
	case "strong", "worth", "stretch", "skip":
		return v
	default:
		return strings.ReplaceAll(v, " ", "_")
	}
}

// stabilityClass maps a research brief stability verdict (one of: strong,
// stable, caution, avoid — see research_briefs.stability_verdict CHECK
// constraint) to a CSS class suffix.
func stabilityClass(verdict string) string {
	v := strings.ToLower(strings.TrimSpace(verdict))
	switch v {
	case "strong", "stable", "caution", "avoid":
		return v
	default:
		return strings.ReplaceAll(v, " ", "_")
	}
}

// formatSalary converts a cents-denominated integer amount to a compact
// "$XXXk" style string. Values are treated as whole-dollar cents, i.e. the
// same unit as salary_min/salary_max in the DB (dollars), rounded to the
// nearest thousand.
func formatSalary(cents int) string {
	dollars := cents
	k := dollars / 1000
	return fmt.Sprintf("$%sk", strconv.Itoa(k))
}

// safeHTML marks a string as safe HTML, bypassing html/template's
// auto-escaping. Only use with trusted, already-sanitized content (e.g.
// goldmark output).
func safeHTML(s string) template.HTML {
	return template.HTML(s)
}

// splitTagsFn splits a comma-separated string into a slice of trimmed tags.
func splitTagsFn(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	tags := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			tags = append(tags, p)
		}
	}
	return tags
}

// reflectValueOf returns a reflect.Value guaranteed to support Len/Index,
// panicking with a clear message if items isn't a slice or array.
func reflectValueOf(items interface{}) reflect.Value {
	v := reflect.ValueOf(items)
	if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
		panic(fmt.Sprintf("render: PartialEach requires a slice or array, got %T", items))
	}
	return v
}

// dictFn builds a map[string]interface{} from an alternating key/value
// argument list, for passing multiple values into a sub-template.
func dictFn(values ...interface{}) (map[string]interface{}, error) {
	if len(values)%2 != 0 {
		return nil, fmt.Errorf("dict: odd number of arguments")
	}
	d := make(map[string]interface{}, len(values)/2)
	for i := 0; i < len(values); i += 2 {
		key, ok := values[i].(string)
		if !ok {
			return nil, fmt.Errorf("dict: keys must be strings")
		}
		d[key] = values[i+1]
	}
	return d, nil
}
