<script setup lang="ts">
const {
  detail,
  loading,
  error,
  actionError,
  toggling,
  editOpen,
  createOpen,
  membersOpen,
  moveOpen,
  orderOpen,
  deleteOpen,
  openEdit,
  openCreate,
  openDelete,
  openMove,
  openOrder,
  openMembers,
  closeAll,
  load,
  toggleCompletion,
  onSaved,
  onCreated,
  onOrdered,
  onDeleted,
  onCommentChanged,
  goHome,
} = useNodePage();
</script>

<template>
  <AppShell>
    <section
      :aria-busy="loading"
      class="node-page"
      :inert="loading || undefined"
    >
      <NSpace :size="24" vertical>
        <NAlert v-if="error" type="error">
          {{ error }}
          <NButton text @click="load()">重试</NButton>
          <NButton text @click="goHome">返回总览</NButton>
        </NAlert>
        <template v-else-if="detail">
          <header class="node-header">
            <NBreadcrumb aria-label="节点路径" class="node-breadcrumb">
              <NBreadcrumbItem class="breadcrumb-item">
                <RouterLink :to="{ name: 'home' }">总览</RouterLink>
              </NBreadcrumbItem>
              <NBreadcrumbItem
                v-for="item in detail.breadcrumb"
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
                <span class="breadcrumb-label">{{ detail.title }}</span>
              </NBreadcrumbItem>
            </NBreadcrumb>
            <NCard class="node-summary" size="small">
              <div class="node-heading">
                <div class="node-title">
                  <NButton
                    :aria-label="
                      detail.completedAt ? '标记为未完成' : '标记为已完成'
                    "
                    :aria-pressed="!!detail.completedAt"
                    circle
                    class="node-completion"
                    :disabled="!detail.permissions.canEdit"
                    :loading="toggling"
                    quaternary
                    :type="detail.completedAt ? 'success' : 'default'"
                    @click="toggleCompletion"
                  >
                    <Check v-if="detail.completedAt" :size="24" />
                    <Circle v-else :size="26" />
                  </NButton>
                  <NH1 :class="{ completed: !!detail.completedAt }">
                    {{ detail.title }}
                  </NH1>
                </div>
                <NSpace align="center" class="node-metadata" :size="8">
                  <NodeStatusTag
                    :completed-at="detail.completedAt"
                    :due-at="detail.dueAt"
                  />
                  <NTag v-for="tag in detail.tags" :key="tag" size="small">
                    {{ tag }}
                  </NTag>
                  <span v-if="detail.dueAt" class="due-date">
                    <CalendarDays aria-hidden="true" :size="13" />
                    预期完成 {{ formatDateTime(detail.dueAt) }}
                  </span>
                </NSpace>
                <NSpace
                  v-if="
                    detail.permissions.canEdit ||
                    detail.permissions.canManageMembers ||
                    detail.permissions.canDelete
                  "
                  class="node-actions"
                >
                  <NButton
                    v-if="detail.permissions.canEdit"
                    size="small"
                    @click="openEdit"
                  >
                    编辑节点
                  </NButton>
                  <NButton
                    v-if="detail.permissions.canManageMembers"
                    size="small"
                    @click="openMembers"
                  >
                    管理成员
                  </NButton>
                  <NButton
                    v-if="detail.permissions.canEdit"
                    size="small"
                    @click="openMove"
                  >
                    移动节点
                  </NButton>
                  <NButton
                    v-if="detail.permissions.canDelete"
                    quaternary
                    size="small"
                    type="error"
                    @click="openDelete(detail)"
                  >
                    删除节点
                  </NButton>
                </NSpace>
              </div>
              <p v-if="detail.description" class="node-description">
                {{ detail.description }}
              </p>
            </NCard>
          </header>
          <NAlert v-if="actionError" type="error">{{ actionError }}</NAlert>
          <section aria-label="子节点" class="node-children">
            <SectionHeading :count="detail.children.length" title="子节点">
              <NButton
                v-if="
                  detail.permissions.canReorderChildren &&
                  detail.children.length > 1
                "
                size="small"
                @click="openOrder"
              >
                调整顺序
              </NButton>
              <NButton
                v-if="detail.permissions.canAddChild"
                size="small"
                type="primary"
                @click="openCreate(detail)"
              >
                新建子节点
              </NButton>
            </SectionHeading>
            <NCard size="small">
              <NList v-if="detail.children.length > 0">
                <NodeListItem
                  v-for="child in detail.children"
                  :key="child.id"
                  :node="child"
                />
              </NList>
              <NEmpty v-else description="暂无子节点" size="small" />
            </NCard>
          </section>
          <ProgressSection
            :key="detail.id"
            :can-comment="detail.permissions.canComment"
            :node-id="detail.id"
            :updated-at="detail.updatedAt"
            @changed="onCommentChanged"
          />
        </template>
      </NSpace>
    </section>
    <template #rail>
      <div :aria-busy="loading" :inert="loading || undefined">
        <NodeDetails v-if="detail && !error" :node="detail" />
      </div>
    </template>
  </AppShell>
  <template v-if="detail">
    <NodeFormDialog
      v-model:open="editOpen"
      mode="edit"
      :node="detail"
      @saved="onSaved"
    />
    <NodeFormDialog
      v-model:open="createOpen"
      mode="create"
      :parent-id="detail.id"
      :parent-title="detail.title"
      @saved="onCreated"
    />
    <MembersDialog
      :node="detail"
      :open="membersOpen"
      @close="closeAll"
      @saved="onSaved"
    />
    <MoveNodeDialog
      :node="detail"
      :open="moveOpen"
      @close="closeAll"
      @saved="onSaved"
    />
    <OrderDialog
      :children="detail.children"
      :open="orderOpen"
      :parent-id="detail.id"
      :parent-title="detail.title"
      @close="closeAll"
      @saved="onOrdered"
    />
    <DeleteNodeDialog
      :node="detail"
      :open="deleteOpen"
      @close="closeAll"
      @deleted="onDeleted"
    />
  </template>
</template>

<style scoped>
.node-page {
  max-width: 1080px;
  margin: auto;
  padding: 20px;
}
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
.node-children {
  display: flex;
  flex-direction: column;
  gap: 12px;
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
  .node-page {
    padding: 16px;
  }
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
