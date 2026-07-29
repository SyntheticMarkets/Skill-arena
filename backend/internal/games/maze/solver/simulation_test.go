package solver

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

func TestSimulationAppliesAcceptedAndBlockedActionsAuthoritatively(t *testing.T) {
	instance := testSolver(t)
	actions := []SimulationAction{
		{ArrowID: "a0001", Accepted: false},
		{ArrowID: "a0000", Accepted: true},
		{ArrowID: "a0001", Accepted: true},
	}
	first, err := instance.Simulate(context.Background(), uniqueBoard(), actions)
	if err != nil {
		t.Fatal(err)
	}
	second, err := instance.Simulate(context.Background(), uniqueBoard(), actions)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatal("identical replay actions produced different simulation results")
	}
	if len(first.Steps) != 3 || first.Steps[0].BlockerID != "a0000" ||
		first.Steps[0].CollisionDistance != 2 || first.Steps[0].Accepted ||
		!first.Steps[2].Complete || !first.Complete {
		t.Fatalf("simulation result = %+v", first)
	}
}

func TestSimulationRejectsActionCollisionMismatchAndReuse(t *testing.T) {
	instance := testSolver(t)
	cases := [][]SimulationAction{
		{{ArrowID: "a0001", Accepted: true}},
		{{ArrowID: "a0000", Accepted: false}},
		{{ArrowID: "missing", Accepted: true}},
		{{ArrowID: "a0000", Accepted: true}, {ArrowID: "a0000", Accepted: true}},
	}
	for _, actions := range cases {
		if _, err := instance.Simulate(context.Background(), uniqueBoard(), actions); !errors.Is(err, ErrActionMismatch) {
			t.Fatalf("actions=%+v error=%v", actions, err)
		}
	}
}

func TestSimulationHonorsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := testSolver(t).Simulate(ctx, uniqueBoard(), nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled simulation error = %v", err)
	}
}
