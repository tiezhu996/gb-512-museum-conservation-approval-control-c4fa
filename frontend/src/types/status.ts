import type { EntityConfig } from './domain';

export type ArtifactState = 'registered' | 'stable' | 'treatment' | 'closed';
export const ALL_ARTIFACT_STATE: readonly ArtifactState[] = ['registered', 'stable', 'treatment', 'closed'];
export type ApprovalState = 'draft' | 'review' | 'approved' | 'rejected';
export const ALL_APPROVAL_STATE: readonly ApprovalState[] = ['draft', 'review', 'approved', 'rejected'];

export const ENTITY_CONFIGS: readonly EntityConfig[] = [
  { key: 'artifact', path: 'artifacts', label: '文物', statuses: ['registered', 'stable', 'treatment', 'closed'] as const },
  { key: 'treatmentPlan', path: 'plans', label: '处理方案', statuses: ['draft', 'review', 'approved', 'completed'] as const },
  { key: 'materialTest', path: 'tests', label: '材料检测', statuses: ['planned', 'running', 'verified', 'invalid'] as const },
  { key: 'stageApproval', path: 'approvals', label: '阶段审批', statuses: ['draft', 'review', 'approved', 'rejected'] as const }
];
