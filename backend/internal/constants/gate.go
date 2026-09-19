package constants

// 检测快照门禁结论。阶段审批进入待复核（review）时冻结当时可批准的材料检测
// 快照；批准时逐条比对快照与检测现状，任何一项漂移都会让审批停留在 review。
// 常量同时用于后端持久化/审计与前端展示，取值必须与 frontend/src/types/gate.ts 保持一致。
const (
	// GateVerdictReady：已按关联编码找到处理方案，且存在已核验检测快照，允许进入下一步。
	GateVerdictReady = "ready"
	// GateVerdictPlanMissing：关联编码找不到处理方案，提交待复核被拒绝。
	GateVerdictPlanMissing = "plan_missing"
	// GateVerdictTestMissing：方案存在，但没有任何已核验材料检测，提交待复核被拒绝。
	GateVerdictTestMissing = "test_missing"
	// GateVerdictTestInvalid：批准复核时，冻结的检测已被改判为非“已核验”状态。
	GateVerdictTestInvalid = "test_invalid"
	// GateVerdictVersionChanged：批准复核时，冻结的检测仍在，但版本已经变化（换版）。
	GateVerdictVersionChanged = "version_changed"
	// GateVerdictTestDeleted：批准复核时，冻结的检测已被删除或无法读取。
	GateVerdictTestDeleted = "test_deleted"
	// GateVerdictSnapshotEmpty：待复核审批是门禁上线前产生的历史数据，没有冻结快照。
	GateVerdictSnapshotEmpty = "snapshot_empty"
	// GateVerdictFrozen：已批准审批保留的历史快照，仅用于列表展示。
	GateVerdictFrozen = "frozen"
)

// AllGateVerdict 用于校验与文档化，顺序即前端展示优先级。
var AllGateVerdict = []string{
	GateVerdictReady,
	GateVerdictPlanMissing,
	GateVerdictTestMissing,
	GateVerdictTestInvalid,
	GateVerdictVersionChanged,
	GateVerdictTestDeleted,
	GateVerdictSnapshotEmpty,
	GateVerdictFrozen,
}

// MaterialTestVerified 是材料检测“已核验”状态。状态机定义在 status.go，
// 门禁只承认这一个状态为可冻结的有效检测。
const MaterialTestVerified = "verified"
