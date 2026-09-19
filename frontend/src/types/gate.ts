// 检测快照门禁结论，取值与 backend/internal/constants/gate.go 保持一致。
// 修改结论码时必须同步后端常量，并补充审批页/方案页的展示用例。
export type GateVerdict =
  | 'ready'
  | 'plan_missing'
  | 'test_missing'
  | 'test_invalid'
  | 'version_changed'
  | 'test_deleted'
  | 'snapshot_empty'
  | 'frozen';

export interface GateSnapshotFields {
  gatePlanCode?: string;
  gateTestCode?: string;
  gateTestName?: string;
  gateTestVersion?: number;
  gateTestStatus?: string;
  gateTestUpdatedAt?: string;
  gateFrozenAt?: string;
  gateVerdict?: string;
  // 读路径实时计算，不入库；后端用 GateLiveVerdict/GateLiveReason 返回。
  gateLiveVerdict?: GateVerdict | '';
  gateLiveReason?: string;
}

interface GateMeta {
  label: string;
  tone: 'success' | 'warning' | 'danger' | 'neutral';
  hint: string;
}

// 结论的中文文案与色调集中维护，审批列表、方案页面板与详情共用同一份口径。
export const GATE_VERDICT_META: Record<GateVerdict, GateMeta> = {
  ready: { label: '门禁放行', tone: 'success', hint: '冻结检测仍为已核验且版本未变，可以批准' },
  plan_missing: { label: '方案缺失', tone: 'danger', hint: '关联编码找不到处理方案，提交待复核会被拒绝' },
  test_missing: { label: '无有效检测', tone: 'danger', hint: '处理方案下没有已核验材料检测，提交待复核会被拒绝' },
  test_invalid: { label: '检测改判', tone: 'danger', hint: '冻结检测已不再是已核验状态，审批保持待复核' },
  version_changed: { label: '检测换版', tone: 'warning', hint: '冻结检测版本已变化，需退回草稿重新冻结' },
  test_deleted: { label: '检测缺失', tone: 'danger', hint: '冻结检测已删除或不可读，审批保持待复核' },
  snapshot_empty: { label: '快照缺失', tone: 'warning', hint: '历史待复核审批缺少冻结快照，需退回后重新提交' },
  frozen: { label: '已冻结批准', tone: 'neutral', hint: '批准时通过门禁，检测快照随审批永久冻结' },
};

export function gateVerdictMeta(verdict?: string): GateMeta | null {
  if (!verdict) return null;
  return GATE_VERDICT_META[verdict as GateVerdict] ?? null;
}

// 审批当前是否应该展示门禁区块：待复核与已批准都有快照语义，草稿无需门禁。
export function hasGateSnapshot(record: GateSnapshotFields): boolean {
  return Boolean(record.gateTestCode && record.gateVerdict);
}
