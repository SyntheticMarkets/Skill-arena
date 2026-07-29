package solver

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"sort"

	"skill-arena/internal/games/maze/generator"
)

type dependencyGraph struct {
	edges      map[string][]string
	dependents map[string][]string
}

func deriveGraph(ctx context.Context, board generator.Board, index occupancyIndex) (dependencyGraph, error) {
	graph := dependencyGraph{
		edges:      make(map[string][]string, len(board.Arrows)),
		dependents: make(map[string][]string, len(board.Arrows)),
	}
	for _, arrow := range board.Arrows {
		if err := ctx.Err(); err != nil {
			return dependencyGraph{}, err
		}
		graph.edges[arrow.ID] = []string{}
		graph.dependents[arrow.ID] = []string{}
	}
	for _, arrow := range board.Arrows {
		blockers := map[string]struct{}{}
		dx, dy, ok := directionVector(arrow.Direction)
		if !ok {
			return dependencyGraph{}, errors.New("invalid arrow direction")
		}
		for cell := move(arrow.Head(), dx, dy); inside(board, cell); cell = move(cell, dx, dy) {
			if owner, occupied := index.owners[cellKey(cell)]; occupied && owner != arrow.ID {
				blockers[owner] = struct{}{}
			}
		}
		for blocker := range blockers {
			graph.edges[arrow.ID] = append(graph.edges[arrow.ID], blocker)
			graph.dependents[blocker] = append(graph.dependents[blocker], arrow.ID)
		}
		sort.Strings(graph.edges[arrow.ID])
	}
	for id := range graph.dependents {
		sort.Strings(graph.dependents[id])
	}
	return graph, nil
}

func (graph dependencyGraph) metrics(ctx context.Context) (generator.VerificationMetrics, error) {
	metrics := generator.VerificationMetrics{ArrowCount: len(graph.edges)}
	for id, blockers := range graph.edges {
		if err := ctx.Err(); err != nil {
			return generator.VerificationMetrics{}, err
		}
		metrics.DependencyEdges += len(blockers)
		if len(blockers) == 0 {
			metrics.InitiallyOpen++
		}
		if len(blockers) > 1 {
			metrics.CrossDependencies += len(blockers) - 1
		}
		if len(blockers) == 0 && len(graph.dependents[id]) == 0 {
			metrics.IsolatedArrows++
		}
		if len(graph.dependents[id]) > metrics.Branching {
			metrics.Branching = len(graph.dependents[id])
		}
	}
	depth, err := graph.depth(ctx)
	if err != nil {
		return generator.VerificationMetrics{}, err
	}
	metrics.DependencyDepth = depth
	return metrics, nil
}

func (graph dependencyGraph) depth(ctx context.Context) (int, error) {
	memo := make(map[string]int, len(graph.edges))
	visiting := make(map[string]bool, len(graph.edges))
	var visit func(string) (int, bool, error)
	visit = func(id string) (int, bool, error) {
		if err := ctx.Err(); err != nil {
			return 0, false, err
		}
		if depth, ok := memo[id]; ok {
			return depth, true, nil
		}
		if visiting[id] {
			return 0, false, nil
		}
		visiting[id] = true
		depth := 1
		for _, blocker := range graph.edges[id] {
			childDepth, ok, err := visit(blocker)
			if err != nil {
				return 0, false, err
			}
			if !ok {
				return 0, false, nil
			}
			if childDepth+1 > depth {
				depth = childDepth + 1
			}
		}
		visiting[id] = false
		memo[id] = depth
		return depth, true, nil
	}
	maximum := 0
	for id := range graph.edges {
		depth, ok, err := visit(id)
		if err != nil {
			return 0, err
		}
		if !ok {
			return 0, nil
		}
		if depth > maximum {
			maximum = depth
		}
	}
	return maximum, nil
}

func (graph dependencyGraph) canonical() []byte {
	ids := make([]string, 0, len(graph.edges))
	for id := range graph.edges {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	buffer := &bytes.Buffer{}
	writeUint32(buffer, uint32(len(ids)))
	for _, id := range ids {
		writeBytes(buffer, []byte(id))
		blockers := append([]string(nil), graph.edges[id]...)
		sort.Strings(blockers)
		writeUint32(buffer, uint32(len(blockers)))
		for _, blocker := range blockers {
			writeBytes(buffer, []byte(blocker))
		}
	}
	return buffer.Bytes()
}

func move(cell generator.Cell, dx, dy int) generator.Cell {
	return generator.Cell{Column: cell.Column + dx, Row: cell.Row + dy}
}

func writeUint32(buffer *bytes.Buffer, value uint32) {
	var encoded [4]byte
	binary.BigEndian.PutUint32(encoded[:], value)
	buffer.Write(encoded[:])
}

func writeBytes(buffer *bytes.Buffer, value []byte) {
	writeUint32(buffer, uint32(len(value)))
	buffer.Write(value)
}
