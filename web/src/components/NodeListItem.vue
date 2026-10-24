<script setup lang="ts">
import { NListItem, NSpace, NTag, NText } from 'naive-ui'
import {
  Check,
  Circle,
  CalendarDays,
  MessageSquare,
  Network,
} from '@lucide/vue'
import NodeStatusTag from '@/components/NodeStatusTag.vue'
import CommentPreview from '@/components/CommentPreview.vue'
import UserIdentity from '@/components/UserIdentity.vue'
import type { ChildNode, UserSummary } from '@/lib/api'
import { formatDateTime, isOverdue } from '@/lib/utils'
defineProps<{ node: ChildNode; leaders?: UserSummary[] }>()
</script>
<template>
  <NListItem class="node-list-item">
    <template #prefix>
      <Check
        v-if="node.completedAt"
        :size="16"
        class="completed-icon"
        aria-label="已完成"
      />
      <Circle v-else :size="16" class="node-icon" aria-label="未完成" />
    </template>
    <article class="node-entry">
      <div class="entry-heading">
        <RouterLink
          :to="{ name: 'node', params: { id: node.id } }"
          class="node-link"
          >{{ node.title }}</RouterLink
        >
        <NodeStatusTag
          v-if="node.completedAt || isOverdue(node.dueAt, node.completedAt)"
          :due-at="node.dueAt"
          :completed-at="node.completedAt"
        />
      </div>
      <div class="entry-metadata">
        <NTag v-for="tag in node.tags" :key="tag" size="small">{{ tag }}</NTag>
        <span v-if="node.dueAt" class="metadata-item"
          ><CalendarDays :size="13" aria-hidden="true" />预期完成
          {{ formatDateTime(node.dueAt) }}</span
        >
        <span v-if="node.childCount" class="metadata-item"
          ><Network :size="13" aria-hidden="true" />{{
            node.childCount
          }}
          个子节点</span
        >
        <span v-if="node.commentCount" class="metadata-item"
          ><MessageSquare :size="13" aria-hidden="true" />{{
            node.commentCount
          }}
          条评论</span
        >
      </div>
      <NSpace v-if="leaders" :size="12" class="entry-leaders">
        <NText depth="3">负责人</NText>
        <UserIdentity
          v-for="leader in leaders"
          :key="leader.subject"
          :user="leader"
          :size="20"
        />
        <NText v-if="!leaders.length" depth="3">暂无负责人</NText>
      </NSpace>
      <CommentPreview
        v-if="node.latestComment"
        :comment="node.latestComment"
        class="entry-comment"
      />
    </article>
  </NListItem>
</template>
<style scoped>
.node-list-item :deep(.n-list-item__prefix) {
  align-self: flex-start;
  margin-right: 10px;
  padding-top: 4px;
}
.node-list-item :deep(.n-list-item__main) {
  min-width: 0;
}
.node-list-item:first-child {
  padding-top: 0;
}
.node-list-item:last-child {
  padding-bottom: 0;
}
.node-icon {
  color: var(--guild-muted);
}
.completed-icon {
  color: #157347;
}
.entry-heading {
  display: flex;
  align-items: baseline;
  flex-wrap: wrap;
  gap: 6px 10px;
}
.node-link {
  color: inherit;
  text-decoration: none;
  overflow-wrap: anywhere;
  font-size: var(--guild-item-title-size);
  line-height: 24px;
  font-weight: 600;
}
.node-link:hover {
  color: var(--guild-primary);
}
.entry-metadata {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 4px 12px;
  margin-top: 4px;
  color: var(--guild-muted);
  font-size: var(--guild-meta-size);
  line-height: 20px;
}
.metadata-item {
  display: inline-flex;
  align-items: center;
  gap: 4px;
}
.entry-leaders,
.entry-comment {
  margin-top: 8px;
}
.entry-leaders {
  font-size: 12px;
}
</style>
