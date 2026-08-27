-- 0001_init.down.sql
-- Drops all tables in reverse dependency order.

DROP TABLE IF EXISTS application_events;
DROP TABLE IF EXISTS applications;
DROP TABLE IF EXISTS search_results;
DROP TABLE IF EXISTS search_batches;
DROP TABLE IF EXISTS fit_signals;
DROP TABLE IF EXISTS fit_reports;
DROP TABLE IF EXISTS research_briefs;
DROP TABLE IF EXISTS boards;
DROP TABLE IF EXISTS companies;
