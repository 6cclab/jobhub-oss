DROP INDEX IF EXISTS idx_eval_results_ai_tells;

ALTER TABLE eval_results
    DROP COLUMN IF EXISTS ai_tells_score,
    DROP COLUMN IF EXISTS ai_tells_details;
