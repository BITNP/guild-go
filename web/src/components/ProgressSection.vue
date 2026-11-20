<script setup lang="ts">
import { NAlert, NButton, NCard, NEmpty, NList, NSpace, NSpin } from "naive-ui";
import { ref, watch } from "vue";

import CommentComposer from "@/components/CommentComposer.vue";
import CommentItem from "@/components/CommentItem.vue";
import ConfirmAction from "@/components/ConfirmAction.vue";
import DescriptionChangeItem from "@/components/DescriptionChangeItem.vue";
import SectionHeading from "@/components/SectionHeading.vue";
import type { Comment, ProgressCursor, ProgressItem } from "@/lib/api";
import {
  ApiError,
  deleteComment,
  fetchProgress,
  toggleReaction,
} from "@/lib/api";
import { errorMessage } from "@/lib/utils";

const emit = defineEmits<{ changed: [] }>();
const props = defineProps<{
  nodeId: number;
  canComment: boolean;
  updatedAt: string;
}>();
const items = ref<ProgressItem[]>([]);
const loading = ref(true);
const error = ref<string | null>(null);
const loadError = ref<string | null>(null);
const hasMore = ref(false);
const before = ref<ProgressCursor | null>(null);
const loadingMore = ref(false);
const deleteTarget = ref<Comment | null>(null);
const deleting = ref(false);
const deleteError = ref<string | null>(null);
const reacting = ref(new Set<number>());

async function load() {
  const requestedNode = props.nodeId;
  const requestedUpdate = props.updatedAt;
  const current = () =>
    requestedNode === props.nodeId && requestedUpdate === props.updatedAt;
  loading.value = true;
  loadError.value = null;
  try {
    const page = await fetchProgress(requestedNode);
    if (!current()) {
      return;
    }
    items.value = page.items;
    hasMore.value = page.hasMore;
    before.value = page.nextBefore;
  } catch (err) {
    if (current()) {
      loadError.value =
        err instanceof ApiError
          ? err.message
          : "无法加载进展，请检查网络后重试";
    }
  } finally {
    if (current()) {
      loading.value = false;
    }
  }
}

async function loadMore() {
  if (loading.value || loadingMore.value || before.value === null) {
    return;
  }
  const requestedUpdate = props.updatedAt;
  loadingMore.value = true;
  error.value = null;
  try {
    const page = await fetchProgress(props.nodeId, before.value);
    if (requestedUpdate !== props.updatedAt) {
      return;
    }
    items.value.push(...page.items);
    hasMore.value = page.hasMore;
    before.value = page.nextBefore;
  } catch (err) {
    error.value =
      err instanceof ApiError ? err.message : "加载更多进展失败，请重试";
  } finally {
    loadingMore.value = false;
  }
}

function created(comment: Comment) {
  items.value.unshift({ kind: "comment", comment });
  emit("changed");
}

async function reaction(comment: Comment, emoji: string) {
  if (reacting.value.has(comment.id)) {
    return;
  }
  reacting.value.add(comment.id);
  error.value = null;
  try {
    const result = await toggleReaction(comment.id, emoji);
    const target = items.value.find(
      (item) => item.kind === "comment" && item.comment.id === comment.id,
    );
    if (target?.kind === "comment") {
      target.comment.reactions = result.reactions;
    }
  } catch (err) {
    error.value = errorMessage(err, "表情操作失败");
  } finally {
    reacting.value.delete(comment.id);
  }
}

function askDelete(comment: Comment) {
  deleteTarget.value = comment;
  deleteError.value = null;
}

async function confirmDelete() {
  if (!deleteTarget.value || deleting.value) {
    return;
  }
  const id = deleteTarget.value.id;
  deleting.value = true;
  deleteError.value = null;
  try {
    await deleteComment(id);
    items.value = items.value.filter(
      (item) => item.kind !== "comment" || item.comment.id !== id,
    );
    deleteTarget.value = null;
    emit("changed");
  } catch (err) {
    deleteError.value = errorMessage(err, "删除进展失败");
  } finally {
    deleting.value = false;
  }
}

watch(() => [props.nodeId, props.updatedAt], load, { immediate: true });
</script>

<template>
  <section class="comments">
    <SectionHeading
      :count="
        !loading && !loadError
          ? `${items.length}${hasMore ? '+' : ''}`
          : undefined
      "
      title="进展"
    />
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
      <NAlert v-if="error" class="comments-alert" role="alert" type="error">
        {{ error }}
      </NAlert>
      <NSpin :show="loading && items.length === 0">
        <div
          v-if="loading && items.length === 0 && !loadError"
          class="progress-placeholder"
        />
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
      <NSpace v-if="hasMore" class="load-more-row" justify="center">
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
      @close="deleteTarget = null"
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
.progress-placeholder {
  min-height: 120px;
}
.load-more-row {
  margin-top: 16px;
  padding-top: 12px;
  border-top: 1px solid var(--guild-border);
}
.delete-preview {
  white-space: pre-wrap;
  overflow-wrap: anywhere;
  max-height: 120px;
  overflow: auto;
}
</style>
