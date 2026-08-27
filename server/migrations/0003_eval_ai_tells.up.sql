-- Adds the ai_tells dimension: a score for how much the resume prose reads as
-- machine-generated. Existing rows predate the check and are backfilled to
-- 'pass' so they do not retroactively appear to have failed a gate that did not
-- exist when they ran.
ALTER TABLE eval_results
    ADD COLUMN ai_tells_score TEXT NOT NULL DEFAULT 'pass'
        CHECK(ai_tells_score IN ('pass','warn','fail')),
    ADD COLUMN ai_tells_details TEXT NOT NULL DEFAULT '{}';

CREATE INDEX idx_eval_results_ai_tells ON eval_results(ai_tells_score);
