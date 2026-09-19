<script setup lang="ts">
import { computed } from 'vue';
import type { DomainRecord } from '../../types/domain';
import { gateVerdictMeta, hasGateSnapshot } from '../../types/gate';
import { formatDate } from '../../utils/format';

const props = defineProps<{ record: DomainRecord; compact?: boolean }>();

// 展示优先级：待复核时以后端实时重算的结论为准（检测可能已改判/换版），
// 已批准时只展示冻结结论；没有快照（草稿/历史脏数据）给出中性提示。
const verdict = computed(() => {
  if (props.record.status === 'review' && props.record.gateLiveVerdict) {
    return gateVerdictMeta(props.record.gateLiveVerdict);
  }
  return gateVerdictMeta(props.record.gateVerdict);
});
const verdictReason = computed(() => {
  if (props.record.status === 'review' && props.record.gateLiveReason) {
    return props.record.gateLiveReason;
  }
  return verdict.value?.hint ?? '';
});
const snapshot = computed(() => hasGateSnapshot(props.record));
const frozenLabel = computed(() => (snapshot.value ? `${props.record.gateTestCode} · v${props.record.gateTestVersion}` : '未冻结'));
</script>

<template>
  <div class="gate-card" :class="{ 'gate-card--compact': compact }">
    <template v-if="snapshot">
      <div class="gate-card__line">
        <span class="gate-card__code">{{ record.gateTestCode }}</span>
        <span class="status" :class="`status--${verdict?.tone ?? 'neutral'}`">{{ verdict?.label ?? '未知结论' }}</span>
      </div>
      <dl class="gate-card__meta">
        <dt>检测版本</dt><dd>v{{ record.gateTestVersion }}（{{ record.gateTestStatus }}）</dd>
        <dt>处理方案</dt><dd>{{ record.gatePlanCode || record.relatedCode }}</dd>
        <template v-if="!compact">
          <dt>检测名称</dt><dd>{{ record.gateTestName }}</dd>
          <dt>检测更新</dt><dd>{{ formatDate(record.gateTestUpdatedAt || '') }}</dd>
          <dt>冻结时间</dt><dd>{{ formatDate(record.gateFrozenAt || '') }}</dd>
        </template>
      </dl>
      <p v-if="verdictReason" class="gate-card__reason">{{ verdictReason }}</p>
    </template>
    <template v-else>
      <div class="gate-card__line"><span class="gate-card__code muted">{{ frozenLabel }}</span></div>
      <p class="gate-card__reason muted">
        {{ record.status === 'draft' ? '草稿尚未提交复核，门禁将在进入待复核时冻结最新已核验检测。' : '该审批缺少冻结检测快照。' }}
      </p>
    </template>
  </div>
</template>

<style scoped>
.gate-card { min-width: 200px; display: grid; gap: 6px; }
.gate-card--compact { min-width: 168px; gap: 4px; }
.gate-card__line { display: flex; align-items: center; justify-content: space-between; gap: 10px; }
.gate-card__code { font-weight: 700; color: #173a4d; }
.gate-card__meta { display: grid; grid-template-columns: auto 1fr; gap: 2px 10px; margin: 0; font-size: 12px; color: #506572; }
.gate-card__meta dt { color: #8295a1; }
.gate-card__meta dd { margin: 0; }
.gate-card__reason { margin: 0; font-size: 12px; color: #677985; line-height: 1.5; }
.muted { color: #93a4af; }
</style>
