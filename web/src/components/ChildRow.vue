<script setup lang="ts">
import { Check, Clock, MessageSquare } from '@lucide/vue'
import UserAvatar from '@/components/UserAvatar.vue'
import type { ChildNode } from '@/lib/api'
import {
  formatDateTime,
  isOverdue,
  reactionEmoji,
  timeAgo,
} from '@/lib/utils'

defineProps<{
  child: ChildNode
}>()
</script>

<template>
  <RouterLink
    :to="{ name: 'node', params: { id: child.id } }"
    class="block border-b border-line px-3 py-2.5 last:border-b-0 row-hover"
  >
    <div class="flex items-start gap-2">
      <span
        v-if="child.completedAt"
        class="mt-0.5 flex h-4 w-4 shrink-0 items-center justify-center rounded-full bg-ok-soft text-ok"
        title="已完成"
      >
        <Check class="h-3 w-3" />
      </span>
      <span
        v-else
        class="status-dot mt-1.5 shrink-0"
        :class="isOverdue(child.dueAt, child.completedAt) ? 'bg-warn' : 'bg-line-strong'"
      />

      <div class="min-w-0 flex-1">
        <div class="flex flex-wrap items-center gap-x-2 gap-y-1">
          <span class="truncate text-[13px] font-medium text-ink" :class="child.completedAt ? 'text-muted line-through' : ''">
            {{ child.title }}
          </span>
          <span
            v-for="tag in child.tags.slice(0, 3)"
            :key="tag"
            class="tag"
          ># {{ tag }}</span>
          <span v-if="isOverdue(child.dueAt, child.completedAt)" class="inline-flex items-center gap-0.5 rounded bg-warn-soft px-1 py-0.5 text-[10px] text-warn">
            <Clock class="h-2.5 w-2.5" /> 已逾期
          </span>
          <span v-else-if="child.dueAt && !child.completedAt" class="inline-flex items-center gap-0.5 text-[10px] text-muted">
            <Clock class="h-2.5 w-2.5" /> {{ formatDateTime(child.dueAt) }}
          </span>
        </div>

        <div class="mt-1 flex flex-wrap items-center gap-x-3 gap-y-0.5 text-[10px] text-faint">
          <span v-if="child.childCount > 0">{{ child.childCount }} 个子节点</span>
          <span class="inline-flex items-center gap-0.5"><MessageSquare class="h-2.5 w-2.5" /> {{ child.commentCount }}</span>
          <span v-if="child.latestCommentAt">最近评论 {{ timeAgo(child.latestCommentAt) }}</span>
        </div>

        <div
          v-if="child.latestComment"
          class="mt-1.5 flex items-start gap-1.5 rounded border border-line/70 bg-surface-2/70 px-2 py-1.5"
        >
          <UserAvatar
            :name="child.latestComment.author.userName"
            :avatar="child.latestComment.author.avatar"
            size-class="h-4 w-4 text-[8px]"
          />
          <div class="min-w-0 flex-1">
            <div class="flex items-center gap-1.5 text-[10px]">
              <span class="truncate font-medium text-ink-soft">{{ child.latestComment.author.userName }}</span>
              <span class="text-faint">{{ timeAgo(child.latestComment.createdAt) }}</span>
              <span v-if="child.latestComment.attachmentCount" class="text-faint">· {{ child.latestComment.attachmentCount }} 图</span>
            </div>
            <p class="line-clamp-1 text-[11px] text-muted">{{ child.latestComment.bodyPreview || '（图片）' }}</p>
          </div>
          <div v-if="child.latestComment.reactionSummary.length" class="flex shrink-0 items-center gap-1 text-[10px] text-muted">
            <span
              v-for="reaction in child.latestComment.reactionSummary.slice(0, 3)"
              :key="reaction.emoji"
              class="inline-flex items-center gap-0.5"
            >
              {{ reactionEmoji[reaction.emoji]?.symbol }}{{ reaction.count }}
            </span>
          </div>
        </div>
        <p v-else class="mt-1.5 text-[11px] text-faint">暂无评论</p>
      </div>
    </div>
  </RouterLink>
</template>