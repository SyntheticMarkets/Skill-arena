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
