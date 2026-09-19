<script setup lang="ts">
import type { DomainRecord } from '../../types/domain';
import StatusBadge from './StatusBadge.vue';
import EmptyState from './EmptyState.vue';

defineProps<{ records: DomainRecord[] }>();
</script>

<template>
  <section class="approval-timeline" aria-label="阶段审批时间线">
    <header><strong>阶段审批时间线</strong><span>意见按版本追加，历史不可覆盖</span></header>
    <div v-if="records.length" class="timeline-grid">
      <article v-for="item in records.slice(0, 4)" :key="item.id">
        <div class="timeline-marker">v{{ item.version }}</div>
        <div>
          <strong>{{ item.code }} · {{ item.name }}</strong>
          <p v-if="item.opinions?.length">{{ item.opinions[item.opinions.length - 1].actor }}：{{ item.opinions[item.opinions.length - 1].opinion }}</p>
          <p v-else>等待首条阶段意见</p>
        </div>
        <StatusBadge :status="item.status"/>
      </article>
    </div>
    <EmptyState v-else title="暂无审批轨迹" description="方案提交评审后，版本意见会在这里显示。"/>
  </section>
</template>
