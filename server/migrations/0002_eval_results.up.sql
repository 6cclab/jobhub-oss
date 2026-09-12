CREATE TABLE eval_results (
    id UUID PRIMARY KEY,
    resume_file TEXT NOT NULL,
    company TEXT NOT NULL,
    role_title TEXT NOT NULL,
    posting_url TEXT,

    keyword_score TEXT NOT NULL CHECK(keyword_score IN ('pass','warn','fail')),
    gap_fill_score TEXT NOT NULL CHECK(gap_fill_score IN ('pass','warn','fail')),
    style_score TEXT NOT NULL CHECK(style_score IN ('pass','warn','fail')),
    skills_score TEXT NOT NULL CHECK(skills_score IN ('pass','warn','fail')),
    structural_score TEXT NOT NULL CHECK(structural_score IN ('pass','warn','fail')),

    verdict TEXT NOT NULL CHECK(verdict IN ('strong','acceptable','needs_rework','critical')),
    primary_issue TEXT,

    keyword_details TEXT NOT NULL,
    gap_fill_details TEXT NOT NULL,
    style_details TEXT NOT NULL,
    adversarial_details TEXT,

    eval_mode TEXT NOT NULL CHECK(eval_mode IN ('gate','manual','batch')),
    attempt_number INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_eval_results_company ON eval_results(company);
CREATE INDEX idx_eval_results_verdict ON eval_results(verdict);
CREATE INDEX idx_eval_results_created ON eval_results(created_at);
