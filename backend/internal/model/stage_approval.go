package model

import "time"

// StageApproval models 阶段审批 as an independently versioned aggregate. The fields
// cover ownership, operational context, evidence and measured risk so later
// changes naturally span persistence, service and UI layers.
type StageApproval struct {
	BaseModel
	Facility    string    `json:"facility" gorm:"size:120;index"`
	Owner       string    `json:"owner" gorm:"size:120;index"`
	Category    string    `json:"category" gorm:"size:80;index"`
	RiskLevel   string    `json:"riskLevel" gorm:"size:32;index"`
	MetricValue float64   `json:"metricValue"`
	MetricUnit  string    `json:"metricUnit" gorm:"size:24"`
	EffectiveAt time.Time `json:"effectiveAt"`
	Evidence    string    `json:"evidence" gorm:"size:2000"`
	RelatedCode string    `json:"relatedCode" gorm:"size:64;index"`
	// Gate snapshot: frozen when the approval enters review. The gate re-reads
	// the material test at approval time and blocks if it drifted.
	GateTestCode    string            `json:"gateTestCode" gorm:"size:64;index"`
	GateTestVersion uint              `json:"gateTestVersion"`
	GateVerdict     string            `json:"gateVerdict" gorm:"size:300"`
	Opinions        []ApprovalOpinion `json:"opinions" gorm:"foreignKey:StageApprovalID;constraint:OnDelete:CASCADE"`
}

func (item *StageApproval) GetBase() *BaseModel { return &item.BaseModel }

func (item StageApproval) TableName() string { return "stage_approvals" }

var StageApprovalInitialStatus = "draft"

// ApprovalOpinion is append-only: one immutable opinion is stored for every
// aggregate version created by an approval state transition.
type ApprovalOpinion struct {
	ID              uint      `json:"id" gorm:"primaryKey"`
	StageApprovalID uint      `json:"stageApprovalId" gorm:"uniqueIndex:idx_approval_opinion_version;not null"`
	Version         uint      `json:"version" gorm:"uniqueIndex:idx_approval_opinion_version;not null"`
	Status          string    `json:"status" gorm:"size:40;not null"`
	Opinion         string    `json:"opinion" gorm:"size:500;not null"`
	Actor           string    `json:"actor" gorm:"size:80;not null;index"`
	RequestID       string    `json:"requestId" gorm:"size:64;not null;index"`
	CreatedAt       time.Time `json:"createdAt" gorm:"index"`
}
