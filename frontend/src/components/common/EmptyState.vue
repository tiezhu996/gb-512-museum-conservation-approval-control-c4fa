<script setup lang="ts">
defineProps<{ title: string; description?: string }>();
</script>

<template>
  <section class="empty-state" role="status">
    <strong>{{ title }}</strong>
    <p>{{ description || '当前没有可展示的业务记录。' }}</p>
  </section>
</template>
