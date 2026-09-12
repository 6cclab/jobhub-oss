// Package migrations embeds the SQL migration files so they can be shipped
// inside the compiled binary and applied via golang-migrate's iofs source.
package migrations

import "embed"

// FS embeds all migration files (*.sql) in this directory.
//
//go:embed *.sql
var FS embed.FS
