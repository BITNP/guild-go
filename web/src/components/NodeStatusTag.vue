<script setup lang="ts">
import { computed } from 'vue'
import { NTag } from 'naive-ui'
import { isOverdue } from '@/lib/utils'
const props = defineProps<{
  dueAt: string | null
  completedAt: string | null
}>()
const overdue = computed(() => isOverdue(props.dueAt, props.completedAt))
</script>
<template>
  <NTag
    :type="completedAt ? 'success' : overdue ? 'warning' : 'default'"
    size="small"
    :bordered="false"
    >{{ completedAt ? '已完成' : overdue ? '已逾期' : '未完成' }}</NTag
  >
</template>
