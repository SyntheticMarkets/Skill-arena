package migrations

import _ "embed"

// FinancialPlatform is the immutable Sprint 3 schema migration.
//
//go:embed 004_financial_platform.sql
var FinancialPlatform string

// FinancialCompletion adds provider, evidence, artifact, and reserve records.
//
//go:embed 005_financial_completion.sql
var FinancialCompletion string

// AdminCRM is the normalized Sprint 4 operations and compliance schema.
//
//go:embed 006_admin_crm.sql
var AdminCRM string

// RealtimeArena is the normalized Sprint 5 session, queue, presence, and replay schema.
//
//go:embed 007_realtime_arena.sql
var RealtimeArena string

// GamesPuzzleService is the additive Sprint 6 Phase 2 puzzle metadata schema.
//
//go:embed 008_games_puzzle_service.sql
var GamesPuzzleService string

// GamesRuntime is the additive Sprint 6 Phase 7 participant-state and action schema.
//
//go:embed 009_games_runtime.sql
var GamesRuntime string

// BackgroundJobsAuthoritative moves the production worker queue off snapshots.
//
//go:embed 010_background_jobs_authoritative.sql
var BackgroundJobsAuthoritative string
