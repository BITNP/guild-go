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
          <NodeHeader
            :node="detail"
            :toggling
            @edit="openEdit"
            @manage-members="openMembers"
            @move="openMove"
            @remove="openDelete(detail)"
            @toggle-completion="toggleCompletion"
          />
          <NAlert v-if="actionError" type="error">{{ actionError }}</NAlert>
          <NodeChildren
            :node="detail"
            @create="openCreate(detail)"
            @reorder="openOrder"
          />
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
@media (max-width: 639px) {
  .node-page {
    padding: 16px;
  }
}
</style>
