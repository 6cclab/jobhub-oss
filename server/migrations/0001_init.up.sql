-- 0001_init.up.sql
-- Initial schema for job search dashboard (Postgres).

CREATE TABLE companies (
    id         UUID PRIMARY KEY,
    name       TEXT NOT NULL UNIQUE,
    slug       TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE boards (
    id             UUID PRIMARY KEY,
    slug           TEXT NOT NULL UNIQUE,
    name           TEXT NOT NULL,
    ats            TEXT NOT NULL CHECK (ats IN ('greenhouse', 'lever', 'ashby')),
    tags           TEXT NOT NULL DEFAULT '',
    status         TEXT NOT NULL CHECK (status IN ('tracked', 'discovery', 'dead')),
    last_probed_at TIMESTAMPTZ,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE research_briefs (
    id                 UUID PRIMARY KEY,
    company_id         UUID NOT NULL REFERENCES companies(id),
    stability_verdict  TEXT NOT NULL CHECK (stability_verdict IN ('strong', 'stable', 'caution', 'avoid')),
    stability_notes    TEXT,
    stage              TEXT,
    headcount          TEXT,
    founded            TEXT,
    remote_policy      TEXT,
    culture_notes      TEXT,
    tech_stack         TEXT,
    salary_range_text  TEXT,
    salary_source      TEXT,
    raw_markdown       TEXT NOT NULL,
    researched_at      TIMESTAMPTZ NOT NULL,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_research_briefs_company_id ON research_briefs(company_id);
CREATE INDEX idx_research_briefs_created_at ON research_briefs(created_at);

CREATE TABLE fit_reports (
    id                 UUID PRIMARY KEY,
    company_id         UUID NOT NULL REFERENCES companies(id),
    role_title         TEXT NOT NULL,
    location           TEXT,
    level              TEXT,
    posting_url        TEXT,
    verdict            TEXT NOT NULL CHECK (verdict IN ('strong', 'worth', 'stretch', 'skip')),
    verdict_summary    TEXT NOT NULL,
    why_apply          TEXT,
    research_brief_id  UUID REFERENCES research_briefs(id),
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_fit_reports_company_id ON fit_reports(company_id);
CREATE INDEX idx_fit_reports_research_brief_id ON fit_reports(research_brief_id);
CREATE INDEX idx_fit_reports_created_at ON fit_reports(created_at);

CREATE TABLE fit_signals (
    id             UUID PRIMARY KEY,
    fit_report_id  UUID NOT NULL REFERENCES fit_reports(id) ON DELETE CASCADE,
    kind           TEXT NOT NULL CHECK (kind IN ('match', 'gap', 'flag')),
    requirement    TEXT NOT NULL,
    evidence       TEXT NOT NULL,
    source         TEXT,
    sort_order     INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX idx_fit_signals_fit_report_id ON fit_signals(fit_report_id);

CREATE TABLE search_batches (
    id               UUID PRIMARY KEY,
    ran_at           TIMESTAMPTZ NOT NULL,
    board_count      INTEGER NOT NULL DEFAULT 0,
    raw_count        INTEGER NOT NULL DEFAULT 0,
    location_filter  TEXT,
    notes            TEXT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE search_results (
    id                UUID PRIMARY KEY,
    search_batch_id   UUID NOT NULL REFERENCES search_batches(id) ON DELETE CASCADE,
    company_id        UUID NOT NULL REFERENCES companies(id),
    role_title        TEXT NOT NULL,
    location          TEXT,
    is_remote         BOOLEAN NOT NULL DEFAULT FALSE,
    salary_min        INTEGER,
    salary_max        INTEGER,
    salary_disclosed  BOOLEAN NOT NULL DEFAULT FALSE,
    below_floor       BOOLEAN NOT NULL DEFAULT FALSE,
    posting_url       TEXT NOT NULL,
    fit_tier          TEXT NOT NULL CHECK (fit_tier IN ('strong', 'good')),
    tags              TEXT NOT NULL DEFAULT '',
    level_tag         TEXT,
    domain_tag        TEXT,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_search_results_company_id ON search_results(company_id);
CREATE INDEX idx_search_results_batch_id ON search_results(search_batch_id);
CREATE INDEX idx_search_results_batch_id_fit_tier ON search_results(search_batch_id, fit_tier);

CREATE TABLE applications (
    id              UUID PRIMARY KEY,
    company_id      UUID NOT NULL REFERENCES companies(id),
    role_title      TEXT NOT NULL,
    source          TEXT,
    status          TEXT NOT NULL DEFAULT 'applied' CHECK (
                        status IN ('applied', 'phone_screen', 'onsite', 'offer', 'rejected', 'ghosted', 'withdrawn')
                    ),
    applied_at      TEXT,
    resume_file     TEXT,
    resume_data     BYTEA,
    fit_report_id   UUID REFERENCES fit_reports(id),
    notes           TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_applications_company_id ON applications(company_id);
CREATE INDEX idx_applications_fit_report_id ON applications(fit_report_id);
CREATE INDEX idx_applications_status ON applications(status);

CREATE TABLE application_events (
    id              UUID PRIMARY KEY,
    application_id  UUID NOT NULL REFERENCES applications(id) ON DELETE CASCADE,
    from_status     TEXT,
    to_status       TEXT NOT NULL,
    note            TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_application_events_application_id ON application_events(application_id);
