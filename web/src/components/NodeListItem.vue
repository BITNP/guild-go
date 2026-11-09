<script setup lang="ts">
import type { ChildNode, UserSummary } from "@/api";

defineProps<{ node: ChildNode; leaders?: UserSummary[] }>();
</script>

<template>
  <NListItem class="node-list-item">
    <article class="node-entry">
      <Check
        v-if="node.completedAt"
        aria-label="已完成"
        class="completed-icon"
        :size="16"
      />
      <Circle v-else aria-label="未完成" class="node-icon" :size="16" />
      <div class="entry-content">
        <div class="entry-heading">
          <RouterLink
            class="node-link"
            :to="{ name: 'node', params: { id: node.id } }"
          >
            {{ node.title }}
          </RouterLink>
          <NodeStatusTag
            v-if="node.completedAt || isOverdue(node.dueAt, node.completedAt)"
            :completed-at="node.completedAt"
            :due-at="node.dueAt"
          />
        </div>
        <div
          v-if="node.tags.length > 0 || node.dueAt || node.childCount"
          class="entry-metadata"
        >
          <NTag v-for="tag in node.tags" :key="tag" size="small">
            {{ tag }}
          </NTag>
          <span v-if="node.dueAt" class="metadata-item">
            <CalendarDays aria-hidden="true" :size="13" />
            预期完成 {{ formatDateTime(node.dueAt) }}
          </span>
          <span v-if="node.childCount" class="metadata-item">
            <Network aria-hidden="true" :size="13" />
            {{ node.childCount }} 个子节点
          </span>
        </div>
        <NSpace v-if="leaders?.length" class="entry-leaders" :size="12">
          <NText depth="3">负责人</NText>
          <UserIdentity
            v-for="leader in leaders"
            :key="leader.subject"
            :size="20"
            :user="leader"
          />
        </NSpace>
        <NText v-else-if="leaders" class="entry-leaders no-leader" depth="3">
          暂无负责人
        </NText>
        <CommentPreview
          v-if="node.latestComment"
          class="entry-comment"
          :comment="node.latestComment"
        />
      </div>
    </article>
  </NListItem>
</template>

<style scoped>
.node-entry {
  position: relative;
  display: grid;
  grid-template-columns: 16px minmax(0, 1fr);
  gap: 10px;
}
.entry-content {
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
.node-icon,
.completed-icon {
  margin-top: 4px;
}
.completed-icon {
  color: var(--guild-success);
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
.node-link::after {
  content: "";
  position: absolute;
  inset: 0;
}
.node-entry :deep(.user-link) {
  position: relative;
  z-index: 1;
}
.node-entry:hover .node-link {
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
.no-leader {
  display: block;
}
</style>
