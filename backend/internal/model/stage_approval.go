package model

import "time"

// StageApproval models 阶段审批 as an independently versioned aggregate. The fields
// cover ownership, operational context, evidence and measured risk so later
// changes naturally span persistence, service and UI layers.
type StageApproval struct {
	BaseModel
	Facility    string            `json:"facility" gorm:"size:120;index"`
	Owner       string            `json:"owner" gorm:"size:120;index"`
	Category    string            `json:"category" gorm:"size:80;index"`
	RiskLevel   string            `json:"riskLevel" gorm:"size:32;index"`
	MetricValue float64           `json:"metricValue"`
	MetricUnit  string            `json:"metricUnit" gorm:"size:24"`
	EffectiveAt time.Time         `json:"effectiveAt"`
	Evidence    string            `json:"evidence" gorm:"size:2000"`
	RelatedCode string            `json:"relatedCode" gorm:"size:64;index"`
	Opinions    []ApprovalOpinion `json:"opinions" gorm:"foreignKey:StageApprovalID;constraint:OnDelete:CASCADE"`

	// 检测快照门禁字段：审批由 draft 进入 review 的瞬间写入，此后冻结。
	// GatePlanCode 冗余关联的处理方案编码，GateTestCode/GateTestVersion/GateTestStatus
	// 记录当时“按更新时间最新的已核验材料检测”，GateVerdict 保存冻结结论。
	GatePlanCode    string    `json:"gatePlanCode" gorm:"size:64;index"`
	GateTestCode    string    `json:"gateTestCode" gorm:"size:64;index"`
	GateTestName    string    `json:"gateTestName" gorm:"size:160"`
	GateTestVersion uint      `json:"gateTestVersion"`
	GateTestStatus  string    `json:"gateTestStatus" gorm:"size:40"`
	GateTestUpdated time.Time `json:"gateTestUpdatedAt"`
	GateFrozenAt    time.Time `json:"gateFrozenAt"`
	GateVerdict     string    `json:"gateVerdict" gorm:"size:32;index"`

	// 以下字段不入库，由 service 在读路径上按快照与检测现状实时计算，
	// 供审批列表与方案页展示“门禁现在是否仍然放行”。
	GateLiveVerdict string `json:"gateLiveVerdict,omitempty" gorm:"-"`
	GateLiveReason  string `json:"gateLiveReason,omitempty" gorm:"-"`
}

// HasGateSnapshot 判断审批是否携带可用于批准校验的冻结快照。
func (item *StageApproval) HasGateSnapshot() bool {
	return item.GateTestCode != "" && item.GateVerdict != ""
}

// GateSnapshot 是冻结时刻的不可变检测坐标，service 用它与现状做比对。
type GateSnapshot struct {
	PlanCode    string
	TestCode    string
	TestName    string
	TestVersion uint
	TestStatus  string
	TestUpdated time.Time
	FrozenAt    time.Time
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
