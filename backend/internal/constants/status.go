package constants

// Shared status values are mirrored in frontend/src/types/status.ts. Keeping
// the lists explicit makes state-machine drift visible during code review.

type ArtifactState string

const (
	ArtifactStateRegistered ArtifactState = "registered"
	ArtifactStateStable     ArtifactState = "stable"
	ArtifactStateTreatment  ArtifactState = "treatment"
	ArtifactStateClosed     ArtifactState = "closed"
)

var AllArtifactState = []string{"registered", "stable", "treatment", "closed"}

type ApprovalState string

const (
	ApprovalStateDraft    ApprovalState = "draft"
	ApprovalStateReview   ApprovalState = "review"
	ApprovalStateApproved ApprovalState = "approved"
	ApprovalStateRejected ApprovalState = "rejected"
)

var AllApprovalState = []string{"draft", "review", "approved", "rejected"}

var ArtifactTransitions = map[string]map[string]bool{
	"registered": {"stable": true, "treatment": true},
	"stable":     {"treatment": true, "closed": true, "registered": true},
	"treatment":  {"closed": true, "stable": true},
	"closed":     {"treatment": true},
}

var TreatmentPlanTransitions = map[string]map[string]bool{
	"draft":     {"review": true, "approved": true},
	"review":    {"approved": true, "completed": true, "draft": true},
	"approved":  {"completed": true, "review": true},
	"completed": {"approved": true},
}

var MaterialTestTransitions = map[string]map[string]bool{
	"planned":  {"running": true, "verified": true},
	"running":  {"verified": true, "invalid": true, "planned": true},
	"verified": {"invalid": true, "running": true},
	"invalid":  {"verified": true},
}

var StageApprovalTransitions = map[string]map[string]bool{
	"draft":    {"review": true},
	"review":   {"approved": true, "rejected": true, "draft": true},
	"approved": {},
	"rejected": {"draft": true, "review": true},
}

func CanTransition(graph map[string]map[string]bool, from, to string) bool {
	targets, exists := graph[from]
	return exists && targets[to]
}
