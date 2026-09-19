<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import type { DomainRecord } from '../../types/domain';
import { listAllStageApprovalsForGate } from '../../api/stage-approval';
import { gateVerdictMeta } from '../../types/gate';
import GateSnapshotCard from './GateSnapshotCard.vue';
import EmptyState from './EmptyState.vue';

// 方案页门禁面板：拉取审批列表，按 RelatedCode（处理方案编码）归组展示。
// 这样在处理方案工作台即可看到每个方案下阶段审批冻结了哪条检测、版本是多少、
// 门禁当前是否仍然放行，无需切换到审批页。
const props = defineProps<{ plans: DomainRecord[] }>();
const approvals = ref<DomainRecord[]>([]);
const loading = ref(false);
const error = ref('');

onMounted(load);

async function load() {
  loading.value = true;
  error.value = '';
  try {
    const result = await listAllStageApprovalsForGate();
    approvals.value = result.data.filter((item) => item.status === 'review' || item.status === 'approved');
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  } finally {
    loading.value = false;
  }
}

const rows = computed(() => {
  const related = new Set(props.plans.map((plan) => plan.code));
  return approvals.value
    .filter((approval) => related.has(approval.relatedCode))
    .sort((a, b) => (a.relatedCode < b.relatedCode ? -1 : a.relatedCode > b.relatedCode ? 1 : b.version - a.version));
});

function badgeOf(record: DomainRecord) {
  const verdict = record.status === 'review' && record.gateLiveVerdict ? record.gateLiveVerdict : record.gateVerdict;
  return gateVerdictMeta(verdict);
}

function planName(code: string): string {
  return props.plans.find((plan) => plan.code === code)?.name ?? code;
}
</script>

<template>
  <section class="gate-panel approval-timeline" aria-label="检测快照门禁面板" v-loading="loading">
    <header>
      <strong>检测快照门禁</strong>
      <span>按处理方案展示阶段审批冻结的检测编码、版本与实时门禁结论</span>
    </header>
    <el-alert v-if="error" :title="error" type="error" show-icon :closable="false"/>
    <EmptyState v-else-if="!loading && rows.length === 0" title="暂无门禁记录"
      description="当审批进入待复核时，会在此冻结对应方案下最新的已核验材料检测。"/>
    <div v-else class="gate-panel__grid">
      <article v-for="approval in rows" :key="approval.id" class="gate-panel__item">
        <div class="gate-panel__head">
          <div>
            <strong>{{ approval.code }}</strong>
            <small>{{ planName(approval.relatedCode) }} · 关联方案 {{ approval.relatedCode }}</small>
          </div>
          <span class="status" :class="`status--${badgeOf(approval)?.tone ?? 'neutral'}`">
            {{ badgeOf(approval)?.label ?? '无结论' }}
          </span>
        </div>
        <GateSnapshotCard :record="approval"/>
      </article>
    </div>
  </section>
</template>

<style scoped>
.gate-panel__grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(280px, 1fr)); gap: 10px; }
.gate-panel__item { background: #f7fafb; border-left: 3px solid #2a9d78; padding: 11px 13px; display: grid; gap: 8px; }
.gate-panel__head { display: flex; align-items: flex-start; justify-content: space-between; gap: 12px; }
.gate-panel__head small { display: block; color: #8295a1; font-size: 12px; margin-top: 2px; }
</style>
