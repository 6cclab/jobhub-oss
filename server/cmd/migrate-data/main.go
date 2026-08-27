package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"
)

func main() {
	if len(os.Args) < 3 {
		log.Fatal("usage: migrate-data <sqlite-path> <postgres-url>")
	}

	src, err := sql.Open("sqlite", os.Args[1])
	if err != nil {
		log.Fatalf("open sqlite: %v", err)
	}
	defer src.Close()

	dst, err := sql.Open("pgx", os.Args[2])
	if err != nil {
		log.Fatalf("open postgres: %v", err)
	}
	defer dst.Close()

	tables := []string{
		"companies",
		"boards",
		"research_briefs",
		"fit_reports",
		"fit_signals",
		"search_batches",
		"search_results",
		"applications",
		"application_events",
		"eval_results",
	}

	for _, table := range tables {
		if err := migrateTable(src, dst, table); err != nil {
			log.Fatalf("migrate %s: %v", table, err)
		}
	}

	log.Println("migration complete")
}

func migrateTable(src, dst *sql.DB, table string) error {
	rows, err := src.Query(fmt.Sprintf("SELECT * FROM %s", table))
	if err != nil {
		return fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("columns: %w", err)
	}

	count := 0
	for rows.Next() {
		values := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range values {
			ptrs[i] = &values[i]
		}

		if err := rows.Scan(ptrs...); err != nil {
			return fmt.Errorf("scan: %w", err)
		}

		placeholders := ""
		colNames := ""
		for i, c := range cols {
			if i > 0 {
				placeholders += ", "
				colNames += ", "
			}
			placeholders += fmt.Sprintf("$%d", i+1)
			colNames += c
		}

		query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s) ON CONFLICT DO NOTHING", table, colNames, placeholders)
		if _, err := dst.Exec(query, values...); err != nil {
			return fmt.Errorf("insert into %s: %w (values: %v)", table, err, values)
		}
		count++
	}

	log.Printf("%s: migrated %d rows", table, count)
	return nil
}
