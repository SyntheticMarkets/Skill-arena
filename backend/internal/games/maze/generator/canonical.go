package generator

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
)

type canonicalDifficultyProfile struct {
	GameID                 string `json:"gameId"`
	SchemaVersion          int    `json:"schemaVersion"`
	Source                 string `json:"source"`
	ComplexityMin          int64  `json:"complexityMin"`
	ComplexityMax          int64  `json:"complexityMax"`
	LineCountMin           int    `json:"lineCountMin"`
	LineCountMax           int    `json:"lineCountMax"`
	DependencyDepthMin     int    `json:"dependencyDepthMin"`
	DependencyDepthMax     int    `json:"dependencyDepthMax"`
	BranchingMin           int    `json:"branchingMin"`
	BranchingMax           int    `json:"branchingMax"`
	FalseRoutesMin         int    `json:"falseRoutesMin"`
	FalseRoutesMax         int    `json:"falseRoutesMax"`
	DensityMinBPS          int    `json:"densityMinBps"`
	DensityMaxBPS          int    `json:"densityMaxBps"`
	PatternBias            string `json:"patternBias"`
	ExpectedSolveTimeMinMS int64  `json:"expectedSolveTimeMinMs"`
	ExpectedSolveTimeMaxMS int64  `json:"expectedSolveTimeMaxMs"`
	VisualComplexityMin    int    `json:"visualComplexityMin"`
	VisualComplexityMax    int    `json:"visualComplexityMax"`
}

func CanonicalProfileHash(profile DifficultyProfile) (string, error) {
	// The profile hash is derived from the remaining immutable profile fields,
	// so callers must be able to calculate it before persistence.
	validated := profile
	validated.ProfileHash = HashFields("skill-arena:difficulty-profile:validation", "computed-profile-hash")
	if err := validated.Validate(); err != nil {
		return "", err
	}
	canonical, err := json.Marshal(canonicalDifficultyProfile{
		GameID: profile.GameID, SchemaVersion: profile.SchemaVersion, Source: profile.Source,
		ComplexityMin: profile.ComplexityMin, ComplexityMax: profile.ComplexityMax,
		LineCountMin: profile.LineCountMin, LineCountMax: profile.LineCountMax,
		DependencyDepthMin: profile.DependencyDepthMin, DependencyDepthMax: profile.DependencyDepthMax,
		BranchingMin: profile.BranchingMin, BranchingMax: profile.BranchingMax,
		FalseRoutesMin: profile.FalseRoutesMin, FalseRoutesMax: profile.FalseRoutesMax,
		DensityMinBPS: profile.DensityMinBPS, DensityMaxBPS: profile.DensityMaxBPS,
		PatternBias:            profile.PatternBias,
		ExpectedSolveTimeMinMS: profile.ExpectedSolveTimeMinMS,
		ExpectedSolveTimeMaxMS: profile.ExpectedSolveTimeMaxMS,
		VisualComplexityMin:    profile.VisualComplexityMin,
		VisualComplexityMax:    profile.VisualComplexityMax,
	})
	if err != nil {
		return "", err
	}
	return HashBytes("skill-arena:difficulty-profile:v1", canonical), nil
}

func CanonicalBoard(board Board) ([]byte, error) {
	if err := ValidateGeometry(board); err != nil {
		return nil, err
	}
	buffer := &bytes.Buffer{}
	writeUint32(buffer, uint32(board.GeometryVersion))
	writeUint32(buffer, uint32(board.RulesVersion))
	writeUint32(buffer, uint32(board.Columns))
	writeUint32(buffer, uint32(board.Rows))
	arrows := append([]Arrow(nil), board.Arrows...)
	sort.Slice(arrows, func(i, j int) bool { return arrows[i].ID < arrows[j].ID })
	writeUint32(buffer, uint32(len(arrows)))
	for _, arrow := range arrows {
		writeBytes(buffer, []byte(arrow.ID))
		buffer.WriteByte(byte(arrow.Direction))
		writeUint32(buffer, uint32(len(arrow.Cells)))
		for _, cell := range arrow.Cells {
			writeUint32(buffer, uint32(cell.Column))
			writeUint32(buffer, uint32(cell.Row))
		}
	}
	return buffer.Bytes(), nil
}

func CanonicalGraph(graph DependencyGraph) ([]byte, error) {
	if len(graph.Edges) == 0 {
		return nil, errors.New("dependency graph is empty")
	}
	buffer := &bytes.Buffer{}
	ids := make([]string, 0, len(graph.Edges))
	for id := range graph.Edges {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	writeUint32(buffer, uint32(len(ids)))
	for _, id := range ids {
		writeBytes(buffer, []byte(id))
		blockers := append([]string(nil), graph.Edges[id]...)
		sort.Strings(blockers)
		writeUint32(buffer, uint32(len(blockers)))
		for _, blocker := range blockers {
			writeBytes(buffer, []byte(blocker))
		}
	}
	return buffer.Bytes(), nil
}

func HashBytes(domain string, values ...[]byte) string {
	target := sha256.New()
	writeBinaryField(target, []byte(domain))
	for _, value := range values {
		writeBinaryField(target, value)
	}
	return "sha256:" + hex.EncodeToString(target.Sum(nil))
}

func writeUint32(buffer *bytes.Buffer, value uint32) {
	var data [4]byte
	binary.BigEndian.PutUint32(data[:], value)
	buffer.Write(data[:])
}

func writeBytes(buffer *bytes.Buffer, value []byte) {
	writeUint32(buffer, uint32(len(value)))
	buffer.Write(value)
}

type binaryWriter interface {
	Write([]byte) (int, error)
}

func writeBinaryField(target binaryWriter, value []byte) {
	var size [8]byte
	binary.BigEndian.PutUint64(size[:], uint64(len(value)))
	_, _ = target.Write(size[:])
	_, _ = target.Write(value)
}
