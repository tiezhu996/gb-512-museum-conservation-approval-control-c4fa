<script setup lang="ts">
import EntityPage from '../components/EntityPage.vue';
import ApprovalTimeline from '../components/common/ApprovalTimeline.vue';
import { ENTITY_CONFIGS } from '../types/status';
import { useStageApprovalStore } from '../stores/stage-approval';

const store = useStageApprovalStore();
</script>

<template><EntityPage :config="ENTITY_CONFIGS[3]" :store="store"><template #insight><ApprovalTimeline :records="store.items"/></template></EntityPage></template>
