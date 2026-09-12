// Package static embeds the CSS/JS static assets so they can be shipped
// inside the compiled binary and served directly by the HTTP server.
package static

import "embed"

// FS embeds all static asset files under css/ and js/.
//
//go:embed css/*.css js/*.js
var FS embed.FS
