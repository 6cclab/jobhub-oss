// Command eval runs the resume eval engine standalone: request JSON on stdin,
// result JSON on stdout.
//
// This exists because the eval engine was the one part of the JobHub server
// that was genuinely being used. The server was retired on 2026-09-11 (see
// docs/sqlite-funnel-design.md) and everything else it did moved to
// scripts/jobhub_db.py, but eval.Engine is real work: keyword coverage, gap
// fill, prose checks, AI tells, structural limits. Reimplementing it in Python
// would fork the logic and its test suite.
//
// eval.Engine.Run is pure -- no database, no network, no repository. Only the
// HTTP handler around it needed Postgres. So the engine needs no server, just
// an entry point.
//
//	go run ./cmd/eval < request.json > result.json
//	go run ./cmd/eval --config ../user/eval-config.yaml < request.json
//
// Exit codes: 0 the eval ran (whatever its verdict), 1 bad input or unreadable
// config. A `needs_rework` verdict is NOT a non-zero exit -- per
// prompts/rules/job-eval-gate.md the eval is advisory, and making it an exit
// code would rebuild the scored loop that rule exists to remove.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/6cclab/jobhub/internal/eval"
)

func main() {
	cfgPath := flag.String("config", "../user/eval-config.yaml",
		"path to eval-config.yaml; falls back to built-in defaults if unreadable")
	indent := flag.Bool("indent", true, "pretty-print the result")
	flag.Parse()

	cfg, err := eval.LoadConfig(*cfgPath)
	if err != nil {
		// LoadConfig returns usable defaults alongside the error. Say so and
		// continue rather than refusing to run: a missing user config is a
		// normal state for a fresh checkout, not a failure.
		fmt.Fprintf(os.Stderr, "eval: using built-in defaults (%s: %v)\n", *cfgPath, err)
	}

	var req eval.EvalRequest
	dec := json.NewDecoder(os.Stdin)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		fmt.Fprintf(os.Stderr, "eval: could not read request JSON from stdin: %v\n", err)
		os.Exit(1)
	}

	result := eval.New(cfg).Run(req)

	enc := json.NewEncoder(os.Stdout)
	if *indent {
		enc.SetIndent("", "  ")
	}
	if err := enc.Encode(result); err != nil {
		fmt.Fprintf(os.Stderr, "eval: could not write result: %v\n", err)
		os.Exit(1)
	}
}
