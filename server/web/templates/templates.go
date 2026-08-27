// Package templates embeds the HTML templates so they can be shipped
// inside the compiled binary and parsed by internal/render.
package templates

import "embed"

// FS embeds all template files (*.html) in this directory and its
// subdirectories (fitreports/, searchresults/, applications/, research/,
// boards/, partials/).
//
//go:embed *.html fitreports/*.html searchresults/*.html applications/*.html research/*.html boards/*.html partials/*.html evalresults/*.html
var FS embed.FS
