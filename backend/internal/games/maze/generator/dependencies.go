package generator

import (
	"errors"
	"sort"
)

type DependencyGraph struct {
	Edges      map[string][]string `json:"edges"`
	Dependents map[string][]string `json:"dependents"`
}

func DeriveDependencies(board Board) (DependencyGraph, error) {
	if err := ValidateGeometry(board); err != nil {
		return DependencyGraph{}, err
	}
	occupancy := make(map[uint64]string, len(board.Arrows)*3)
	for _, arrow := range board.Arrows {
		for _, cell := range arrow.Cells {
			occupancy[cell.key()] = arrow.ID
		}
	}
	graph := DependencyGraph{
		Edges:      make(map[string][]string, len(board.Arrows)),
		Dependents: make(map[string][]string, len(board.Arrows)),
	}
	for _, arrow := range board.Arrows {
		seen := map[string]struct{}{}
		dx, dy := arrow.Direction.vector()
		for cell := stepCell(arrow.Head(), dx, dy); inBoard(board, cell); cell = stepCell(cell, dx, dy) {
			blocker, occupied := occupancy[cell.key()]
			if !occupied || blocker == arrow.ID {
				continue
			}
			if _, exists := seen[blocker]; exists {
				continue
			}
			seen[blocker] = struct{}{}
			graph.Edges[arrow.ID] = append(graph.Edges[arrow.ID], blocker)
			graph.Dependents[blocker] = append(graph.Dependents[blocker], arrow.ID)
		}
		if graph.Edges[arrow.ID] == nil {
			graph.Edges[arrow.ID] = []string{}
		}
		if graph.Dependents[arrow.ID] == nil {
			graph.Dependents[arrow.ID] = []string{}
		}
	}
	for id := range graph.Edges {
		sort.Strings(graph.Edges[id])
		sort.Strings(graph.Dependents[id])
	}
	return graph, nil
}

func ValidateDependencyGraph(board Board, graph DependencyGraph, allowIsolated bool) error {
	if len(graph.Edges) != len(board.Arrows) || len(graph.Dependents) != len(board.Arrows) {
		return errors.New("dependency graph does not cover every arrow")
	}
	indegree := make(map[string]int, len(graph.Edges))
	for id, blockers := range graph.Edges {
		indegree[id] = len(blockers)
	}
	queue := make([]string, 0, len(indegree))
	for id, count := range indegree {
		if count == 0 {
			queue = append(queue, id)
		}
	}
	sort.Strings(queue)
	visited := 0
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		visited++
		for _, dependent := range graph.Dependents[id] {
			indegree[dependent]--
			if indegree[dependent] == 0 {
				queue = append(queue, dependent)
				sort.Strings(queue)
			}
		}
	}
	if visited != len(board.Arrows) {
		return errors.New("dependency graph contains a cycle")
	}
	if len(initiallyOpen(graph)) == 0 {
		return errors.New("dependency graph has no initial open arrow")
	}
	if !allowIsolated {
		for _, arrow := range board.Arrows {
			if len(graph.Edges[arrow.ID]) == 0 && len(graph.Dependents[arrow.ID]) == 0 {
				return errors.New("competitive puzzle contains an isolated arrow")
			}
		}
	}
	return nil
}

func initiallyOpen(graph DependencyGraph) []string {
	open := make([]string, 0)
	for id, blockers := range graph.Edges {
		if len(blockers) == 0 {
			open = append(open, id)
		}
	}
	sort.Strings(open)
	return open
}

func stepCell(cell Cell, dx, dy int) Cell {
	return Cell{Column: cell.Column + dx, Row: cell.Row + dy}
}

func inBoard(board Board, cell Cell) bool {
	return cell.Column >= 0 && cell.Column < board.Columns && cell.Row >= 0 && cell.Row < board.Rows
}
