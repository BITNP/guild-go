<script setup lang="ts">
import { NText } from 'naive-ui'
import CommentPreview from '@/components/CommentPreview.vue'
import NodeStatusTag from '@/components/NodeStatusTag.vue'
import UserIdentity from '@/components/UserIdentity.vue'
import type { RecentActivity } from '@/lib/api'
import { formatDateTime, timeAgo } from '@/lib/utils'

defineProps<{ activity: RecentActivity }>()
</script>

<template>
  <CommentPreview v-if="activity.kind === 'comment'" :comment="activity" />
  <div v-else class="activity-preview">
    <div class="activity-byline">
      <UserIdentity v-if="activity.author" :user="activity.author" :size="22" />
      <span>{{ activity.kind === 'node_created' ? '创建了节点' : '节点有更新' }}</span>
      <NText depth="3">
        <time :datetime="activity.createdAt" :title="formatDateTime(activity.createdAt)">
          {{ timeAgo(activity.createdAt) }}
        </time>
      </NText>
    </div>
    <div class="activity-status">
      <NText depth="3">当前状态</NText>
      <NodeStatusTag :due-at="activity.dueAt" :completed-at="activity.completedAt" />
    </div>
  </div>
</template>

<style scoped>
.activity-preview {
  border-left: 2px solid var(--guild-primary);
  padding: 8px 10px;
  background: var(--guild-surface-subtle);
  font-size: var(--guild-meta-size);
}
.activity-byline {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px 8px;
}
.activity-status {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 8px;
}
</style>
