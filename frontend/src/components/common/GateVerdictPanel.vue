<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import type { DomainRecord } from '../../types/domain';
import { listStageApproval } from '../../api/stage-approval';
import { gateVerdictTone } from '../../utils/format';
import StatusBadge from './StatusBadge.vue';
import EmptyState from './EmptyState.vue';

const props = defineProps<{ plans: DomainRecord[] }>();
const approvals = ref<DomainRecord[]>([]);
const loading = ref(false);

async function loadApprovals() {
  loading.value = true;
  try {
    const result = await listStageApproval(1, 100);
    approvals.value = result.data;
  } catch {
    approvals.value = [];
  } finally {
    loading.value = false;
  }
}

onMounted(loadApprovals);
watch(() => props.plans.map((plan) => plan.updatedAt).join(','), loadApprovals);

// 审批按关联编码挂在方案上：关联编码可命中方案编码或方案的关联编码。
const rows = computed(() => {
  const keys = new Set<string>();
  for (const plan of props.plans) {
    if (plan.code) keys.add(plan.code);
    if (plan.relatedCode) keys.add(plan.relatedCode);
  }
  return approvals.value
    .filter((item) => item.relatedCode && keys.has(item.relatedCode))
    .slice(0, 6);
});
</script>

<template>
  <section class="approval-timeline gate-panel" aria-label="检测门禁快照">
    <header>
      <strong>检测门禁快照</strong>
      <span>进入复核冻结最新已核验检测，批准时复核版本与状态</span>
    </header>
    <div v-if="rows.length" class="gate-grid" v-loading="loading">
      <article v-for="item in rows" :key="item.id">
        <div class="timeline-marker">{{ item.gateTestCode || '未冻结' }}</div>
        <div>
          <strong>{{ item.code }} · {{ item.name }}</strong>
          <p v-if="item.gateTestCode">冻结检测 {{ item.gateTestCode }} v{{ item.gateTestVersion }}</p>
          <p v-else>待复核时按关联编码 {{ item.relatedCode }} 冻结检测快照</p>
          <p v-if="item.gateVerdict" :class="`gate-verdict gate-verdict--${gateVerdictTone(item.gateVerdict)}`">{{ item.gateVerdict }}</p>
        </div>
        <StatusBadge :status="item.status"/>
      </article>
    </div>
    <EmptyState v-else title="暂无关联审批" description="当前方案的关联编码尚未挂接阶段审批，门禁快照会在审批进入复核时生成。"/>
  </section>
</template>
