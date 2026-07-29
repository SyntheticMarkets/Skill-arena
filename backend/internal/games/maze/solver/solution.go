package solver

import (
	"bytes"
	"encoding/binary"

	"skill-arena/internal/games/maze/generator"
)

type Step struct {
	Sequence       int
	ArrowID        string
	OpenChoices    int
	EscapeDistance int
}

type Result struct {
	Accepted       bool
	Classification string
	Steps          []Step
	DependencyHash string
	SolutionHash   string
	FinalChecksum  string
	Metrics        generator.VerificationMetrics
}

func canonicalSolution(version int, classification string, steps []Step) []byte {
	buffer := &bytes.Buffer{}
	writeUint32(buffer, uint32(version))
	writeBytes(buffer, []byte(classification))
	writeUint32(buffer, uint32(len(steps)))
	for _, step := range steps {
		writeUint32(buffer, uint32(step.Sequence))
		writeBytes(buffer, []byte(step.ArrowID))
		writeUint32(buffer, uint32(step.OpenChoices))
		writeUint32(buffer, uint32(step.EscapeDistance))
	}
	return buffer.Bytes()
}

func canonicalFinalState(board generator.Board, steps []Step) ([]byte, error) {
	boardBytes, err := generator.CanonicalBoard(board)
	if err != nil {
		return nil, err
	}
	buffer := &bytes.Buffer{}
	writeBytes(buffer, boardBytes)
	writeUint32(buffer, uint32(len(steps)))
	for _, step := range steps {
		writeBytes(buffer, []byte(step.ArrowID))
	}
	return buffer.Bytes(), nil
}

func uint64Bytes(value uint64) []byte {
	var encoded [8]byte
	binary.BigEndian.PutUint64(encoded[:], value)
	return encoded[:]
}
