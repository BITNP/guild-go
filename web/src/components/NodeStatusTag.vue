<script setup lang="ts">
const props = defineProps<{
  dueAt: string | null;
  completedAt: string | null;
}>();
const overdue = computed(() => isOverdue(props.dueAt, props.completedAt));
</script>

<template>
  <NTag
    :bordered="false"
    size="small"
    :type="completedAt ? 'success' : overdue ? 'warning' : 'default'"
  >
    {{ completedAt ? "已完成" : overdue ? "已逾期" : "未完成" }}
  </NTag>
</template>
