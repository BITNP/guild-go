<script setup lang="ts">
import { NTag } from 'naive-ui'
defineProps<{ title: string; count?: number | string }>()
</script>

<template>
  <div class="section-heading">
    <div class="section-title">
      <h2>{{ title }}</h2>
      <NTag
        v-if="count !== undefined"
        size="small"
        round
        :bordered="false"
        class="section-count"
        >{{ count }}</NTag
      >
    </div>
    <div v-if="$slots.default" class="section-actions"><slot /></div>
  </div>
</template>

<style scoped>
.section-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 8px 16px;
  min-height: 32px;
}
.section-title,
.section-actions {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
}
.section-title h2 {
  margin: 0;
  color: var(--guild-text);
  font-size: var(--guild-section-size);
  font-weight: 650;
  line-height: 28px;
}
.section-count {
  font-variant-numeric: tabular-nums;
  color: var(--guild-muted);
}
</style>
