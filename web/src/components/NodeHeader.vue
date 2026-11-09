<script setup lang="ts">
const emit = defineEmits<{
  toggleCompletion: [];
  edit: [];
  manageMembers: [];
  move: [];
  remove: [];
}>();
defineProps<{ node: NodeDetail; toggling: boolean }>();
</script>

<template>
  <header class="node-header">
    <NBreadcrumb aria-label="节点路径" class="node-breadcrumb">
      <NBreadcrumbItem class="breadcrumb-item">
        <RouterLink :to="{ name: 'home' }">总览</RouterLink>
      </NBreadcrumbItem>
      <NBreadcrumbItem
        v-for="item in node.breadcrumb"
        :key="item.id"
        class="breadcrumb-item"
      >
        <RouterLink
          class="breadcrumb-label"
          :to="{ name: 'node', params: { id: item.id } }"
        >
          {{ item.title }}
        </RouterLink>
      </NBreadcrumbItem>
      <NBreadcrumbItem
        aria-current="page"
        class="breadcrumb-item"
        :clickable="false"
      >
        <span class="breadcrumb-label">{{ node.title }}</span>
      </NBreadcrumbItem>
    </NBreadcrumb>
    <NCard class="node-summary" size="small">
      <div class="node-heading">
        <div class="node-title">
          <NButton
            :aria-label="node.completedAt ? '标记为未完成' : '标记为已完成'"
            :aria-pressed="!!node.completedAt"
            circle
            class="node-completion"
            :disabled="!node.permissions.canEdit"
            :loading="toggling"
            quaternary
            :type="node.completedAt ? 'success' : 'default'"
            @click="emit('toggleCompletion')"
          >
            <Check v-if="node.completedAt" :size="24" />
            <Circle v-else :size="26" />
          </NButton>
          <NH1 :class="{ completed: !!node.completedAt }">
            {{ node.title }}
          </NH1>
        </div>
        <NSpace align="center" class="node-metadata" :size="8">
          <NodeStatusTag
            :completed-at="node.completedAt"
            :due-at="node.dueAt"
          />
          <NTag v-for="tag in node.tags" :key="tag" size="small">
            {{ tag }}
          </NTag>
          <span v-if="node.dueAt" class="due-date">
            <CalendarDays aria-hidden="true" :size="13" />
            预期完成 {{ formatDateTime(node.dueAt) }}
          </span>
        </NSpace>
        <NSpace
          v-if="
            node.permissions.canEdit ||
            node.permissions.canManageMembers ||
            node.permissions.canDelete
          "
          class="node-actions"
        >
          <NButton
            v-if="node.permissions.canEdit"
            size="small"
            @click="emit('edit')"
          >
            编辑节点
          </NButton>
          <NButton
            v-if="node.permissions.canManageMembers"
            size="small"
            @click="emit('manageMembers')"
          >
            管理成员
          </NButton>
          <NButton
            v-if="node.permissions.canEdit"
            size="small"
            @click="emit('move')"
          >
            移动节点
          </NButton>
          <NButton
            v-if="node.permissions.canDelete"
            quaternary
            size="small"
            type="error"
            @click="emit('remove')"
          >
            删除节点
          </NButton>
        </NSpace>
      </div>
      <p v-if="node.description" class="node-description">
        {{ node.description }}
      </p>
    </NCard>
  </header>
</template>

<style scoped>
.node-header {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.node-breadcrumb {
  font-size: 12px;
  white-space: normal;
}
.breadcrumb-item {
  max-width: 100%;
}
.breadcrumb-label {
  overflow-wrap: anywhere;
}
.node-metadata {
  padding-left: 46px;
  grid-area: metadata;
}
.due-date {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  color: var(--guild-muted);
  font-size: 12px;
}
.node-heading {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  grid-template-areas: "title actions" "metadata metadata";
  gap: 8px 16px;
}
.node-actions {
  grid-area: actions;
  align-self: start;
}
.node-title {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  grid-area: title;
  min-width: 0;
}
.node-title h1 {
  font-size: var(--guild-page-title-size);
  line-height: 36px;
  font-weight: 600;
  margin: 0;
  overflow-wrap: anywhere;
}
.node-completion {
  flex: none;
}
.completed {
  text-decoration: line-through;
  color: var(--guild-muted);
}
.node-description {
  margin: 16px 0 0;
  padding-top: 12px;
  border-top: 1px solid var(--guild-border);
  white-space: pre-wrap;
  overflow-wrap: anywhere;
  line-height: 1.8;
}
@media (max-width: 639px) {
  .node-title h1 {
    font-size: 22px;
  }
  .node-heading {
    grid-template-columns: minmax(0, 1fr);
    grid-template-areas: "title" "metadata" "actions";
  }
  .node-actions {
    margin-top: 4px;
  }
  .node-metadata {
    padding-left: 0;
  }
}
</style>
