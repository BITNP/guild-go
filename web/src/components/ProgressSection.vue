<script setup lang="ts">
const emit = defineEmits<{ changed: [] }>();
const props = defineProps<{
  nodeId: number;
  canComment: boolean;
  updatedAt: string;
}>();
const {
  items,
  loading,
  displayError,
  loadError,
  hasMore,
  loadingMore,
  deleteTarget,
  deleting,
  deleteError,
  reacting,
  count,
  load,
  loadMore,
  created,
  reaction,
  askDelete,
  confirmDelete,
  closeDelete,
} = useProgressSection({
  nodeId: () => props.nodeId,
  updatedAt: () => props.updatedAt,
  onChanged: () => emit("changed"),
});
</script>

<template>
  <section class="comments">
    <SectionHeading :count title="进展" />
    <CommentComposer
      v-if="canComment"
      :key="nodeId"
      :node-id
      @created="created"
    />
    <NAlert v-else type="info">
      如需发布进展，请联系负责人将你添加为成员。
    </NAlert>
    <NCard size="small">
      <NAlert
        v-if="displayError"
        class="comments-alert"
        role="alert"
        type="error"
      >
        {{ displayError }}
      </NAlert>
      <NSpin :show="loading && items.length === 0">
        <NAlert v-if="loadError" type="error">
          {{ loadError }}
          <NButton text @click="load">重新加载</NButton>
        </NAlert>
        <NEmpty
          v-else-if="!loading && items.length === 0"
          description="暂无进展"
        />
        <NList v-else>
          <template
            v-for="item in items"
            :key="`${item.kind}-${item.kind === 'comment' ? item.comment.id : item.descriptionChange.id}`"
          >
            <CommentItem
              v-if="item.kind === 'comment'"
              :busy="reacting.has(item.comment.id)"
              :comment="item.comment"
              @delete="askDelete"
              @reaction="reaction"
            />
            <DescriptionChangeItem v-else :change="item.descriptionChange" />
          </template>
        </NList>
      </NSpin>
      <NSpace v-if="hasMore" justify="center">
        <NButton :disabled="loading" :loading="loadingMore" @click="loadMore">
          加载更多进展
        </NButton>
      </NSpace>
    </NCard>
    <ConfirmAction
      :busy="deleting"
      confirm-text="删除进展"
      :error="deleteError"
      :open="deleteTarget !== null"
      title="删除进展"
      @close="closeDelete"
      @confirm="confirmDelete"
    >
      <p>图片附件也会一并删除，操作不可撤销。</p>
      <p v-if="deleteTarget?.body" class="delete-preview">
        {{ deleteTarget.body }}
      </p>
    </ConfirmAction>
  </section>
</template>

<style scoped>
.comments {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.comments-alert {
  margin-bottom: 16px;
}
.delete-preview {
  white-space: pre-wrap;
  overflow-wrap: anywhere;
  max-height: 120px;
  overflow: auto;
}
</style>
