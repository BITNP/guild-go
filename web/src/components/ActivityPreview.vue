<script setup lang="ts">
import { NTag } from 'naive-ui'
import UserIdentity from '@/components/UserIdentity.vue'
import type { RecentActivity } from '@/lib/api'
import { formatDateTime, reactionEmoji } from '@/lib/utils'

defineProps<{ activity: RecentActivity; occurredAt: Date }>()

const clock = new Intl.DateTimeFormat('zh-CN', {
  hour: '2-digit',
  minute: '2-digit',
  hourCycle: 'h23',
})
</script>

<template>
  <div class="activity-preview">
    <p v-if="activity.kind === 'comment'" class="activity-body">
      {{ activity.bodyPreview || '发布了图片' }}
    </p>
    <div class="activity-byline">
      <UserIdentity v-if="activity.author" :user="activity.author" :size="18" />
      <template v-if="activity.kind === 'comment'">
        <span v-if="activity.attachmentCount">{{ activity.attachmentCount }} 张图片</span>
        <span
          v-for="reaction in activity.reactionSummary"
          :key="reaction.emoji"
          :aria-label="`${reactionEmoji[reaction.emoji]?.label ?? reaction.emoji} ${reaction.count}`"
        >
          {{ reactionEmoji[reaction.emoji]?.symbol }} {{ reaction.count }}
        </span>
      </template>
      <span v-else-if="activity.kind === 'node_created'">新建节点</span>
      <NTag v-else type="success" size="small" :bordered="false">已完成</NTag>
      <time
        :datetime="occurredAt.toISOString()"
        :title="formatDateTime(occurredAt.toISOString())"
      >
        {{ clock.format(occurredAt) }}
      </time>
    </div>
  </div>
</template>

<style scoped>
.activity-body {
  display: -webkit-box;
  margin: 0 0 4px;
  overflow: hidden;
  overflow-wrap: anywhere;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}
.activity-byline {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 3px 8px;
  color: var(--guild-muted);
  font-size: var(--guild-meta-size);
}
.activity-byline time {
  white-space: nowrap;
  font-variant-numeric: tabular-nums;
}
</style>
