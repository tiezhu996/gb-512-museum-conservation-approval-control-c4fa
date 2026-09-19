
<script setup lang="ts">import EntityPage from '../components/EntityPage.vue'; import { ENTITY_CONFIGS } from '../types/status'; import { useMaterialTestStore } from '../stores/material-test'; const store = useMaterialTestStore();</script>
<template><EntityPage :config="ENTITY_CONFIGS[2]" :store="store"/></template>
