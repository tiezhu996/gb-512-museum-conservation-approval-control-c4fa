package constants

import "testing"

func TestArtifactTransitionGraph(t *testing.T) {
	if !CanTransition(ArtifactTransitions, "registered", "stable") {
		t.Fatalf("expected registered -> stable transition to be allowed")
	}
	if CanTransition(ArtifactTransitions, "registered", "unknown") {
		t.Fatal("unknown status must never be accepted")
	}
}

func TestStageApprovalCannotSkipReview(t *testing.T) {
	if CanTransition(StageApprovalTransitions, "draft", "approved") {
		t.Fatal("draft approval must not skip independent review")
	}
	if !CanTransition(StageApprovalTransitions, "draft", "review") {
		t.Fatal("draft approval must enter review")
	}
}
