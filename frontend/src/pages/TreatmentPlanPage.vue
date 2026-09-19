<script setup lang="ts">
import EntityPage from '../components/EntityPage.vue';
import RiskTag from '../components/common/RiskTag.vue';
import ApprovalTimeline from '../components/common/ApprovalTimeline.vue';
import { ENTITY_CONFIGS } from '../types/status';
import { useTreatmentPlanStore } from '../stores/treatment-plan';

const store = useTreatmentPlanStore();
</script>

<template>
  <EntityPage :config="ENTITY_CONFIGS[1]" :store="store">
    <template #insight><RiskTag :records="store.items"/><ApprovalTimeline :records="store.items"/></template>
  </EntityPage>
</template>
