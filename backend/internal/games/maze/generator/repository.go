package generator

import "context"

type Repository interface {
	RegisterVersion(context.Context, GeneratorVersion) error
	GetVersion(context.Context, VersionKey) (GeneratorVersion, error)
	SaveDifficultyProfile(context.Context, DifficultyProfile) error
	GetDifficultyProfile(context.Context, string) (DifficultyProfile, error)
	CreatePuzzle(context.Context, PuzzleMetadata) error
	GetPuzzle(context.Context, string) (PuzzleMetadata, error)
	GetPuzzleByRequestHash(context.Context, string) (PuzzleMetadata, error)
	FinalizeAndAssign(context.Context, Finalization) (Assignment, error)
	GetAssignment(context.Context, string, string) (Assignment, error)
}
