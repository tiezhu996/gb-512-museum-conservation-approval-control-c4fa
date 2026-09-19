<script setup lang="ts">
import type { DomainRecord } from '../../types/domain';
import StatusBadge from './StatusBadge.vue';

defineProps<{ records: DomainRecord[] }>();
</script>

<template>
  <section class="risk-board" aria-label="文物风险等级">
    <header><strong>文物风险等级</strong><span>{{ records.filter((item) => ['high', 'critical'].includes(item.riskLevel)).length }} 项需优先处理</span></header>
    <div class="risk-grid">
      <article v-for="item in records.slice(0, 4)" :key="item.id">
        <div><strong>{{ item.code }}</strong><span>{{ item.name }}</span></div>
        <em :class="`risk risk--${item.riskLevel}`">{{ item.riskLevel }}</em>
        <StatusBadge :status="item.status"/>
      </article>
    </div>
  </section>
</template>
