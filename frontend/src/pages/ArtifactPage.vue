<script setup lang="ts">
import EntityPage from '../components/EntityPage.vue';
import RiskTag from '../components/common/RiskTag.vue';
import { ENTITY_CONFIGS } from '../types/status';
import { useArtifactStore } from '../stores/artifact';

const store = useArtifactStore();
</script>

<template><EntityPage :config="ENTITY_CONFIGS[0]" :store="store"><template #insight><RiskTag :records="store.items"/></template></EntityPage></template>
