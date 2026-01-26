package job

import "testing"

func TestJobPlanner(t *testing.T) {
	planner := JobPlanner{}
	jobs, skipped := planner.Plan([]string{"a.mp4", "b.gif", "c.txt"})
	if len(jobs) != 2 {
		t.Fatalf("expected 2 jobs, got %d", len(jobs))
	}
	if len(skipped) != 1 {
		t.Fatalf("expected 1 skipped, got %d", len(skipped))
	}
}
