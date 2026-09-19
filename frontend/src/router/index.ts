import { createRouter, createWebHistory } from 'vue-router';
import ArtifactPage from '../pages/ArtifactPage.vue';
import TreatmentPlanPage from '../pages/TreatmentPlanPage.vue';
import MaterialTestPage from '../pages/MaterialTestPage.vue';
import StageApprovalPage from '../pages/StageApprovalPage.vue';
import AuditPage from '../pages/AuditPage.vue';
import { ensureSession } from '../hooks/useAuth';

const roleRank: Record<string, number> = { viewer: 1, operator: 2, reviewer: 3, admin: 4 };
export const router = createRouter({ history: createWebHistory(), routes: [
  { path: '/', redirect: '/artifacts' },
  { path: '/artifacts', component: ArtifactPage, meta: { minimumRole: 'viewer' } },
  { path: '/plans', component: TreatmentPlanPage, meta: { minimumRole: 'viewer' } },
  { path: '/tests', component: MaterialTestPage, meta: { minimumRole: 'viewer' } },
  { path: '/approvals', component: StageApprovalPage, meta: { minimumRole: 'operator' } },
  { path: '/audit', component: AuditPage, meta: { minimumRole: 'reviewer' } },
] });

router.beforeEach(async (to) => {
  const current = await ensureSession();
  const minimumRole = String(to.meta.minimumRole || 'viewer');
  if ((roleRank[current.role] || 0) < (roleRank[minimumRole] || 1)) return '/artifacts';
  return true;
});
