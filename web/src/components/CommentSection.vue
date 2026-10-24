<script setup lang="ts">
import { ref, watch } from 'vue'
import {
  NAlert,
  NButton,
  NCard,
  NEmpty,
  NH2,
  NList,
  NSpace,
  NSpin,
  NText,
} from 'naive-ui'
import CommentComposer from '@/components/CommentComposer.vue'
import CommentItem from '@/components/CommentItem.vue'
import ConfirmAction from '@/components/ConfirmAction.vue'
import {
  deleteComment,
  fetchComments,
  toggleReaction,
  type Comment,
} from '@/lib/api'
import { errorMessage } from '@/lib/utils'
const props = defineProps<{ nodeId: number; canComment: boolean }>(),
  emit = defineEmits<{ changed: [] }>()
const items = ref<Comment[]>([]),
  loading = ref(true),
  error = ref<string | null>(null),
  loadError = ref<string | null>(null),
  hasMore = ref(false),
  before = ref<number | null>(null),
  loadingMore = ref(false),
  deleteTarget = ref<Comment | null>(null),
  deleting = ref(false),
  deleteError = ref<string | null>(null),
  reacting = ref(new Set<number>())
async function load() {
  loading.value = true
  loadError.value = null
  try {
    const page = await fetchComments(props.nodeId)
    items.value = page.items
    hasMore.value = page.hasMore
    before.value = page.nextBefore
  } catch (err) {
    loadError.value = errorMessage(err, '无法加载评论')
  } finally {
    loading.value = false
  }
}
async function loadMore() {
  if (loadingMore.value || before.value === null) return
  loadingMore.value = true
  error.value = null
  try {
    const page = await fetchComments(props.nodeId, before.value)
    items.value.push(...page.items)
    hasMore.value = page.hasMore
    before.value = page.nextBefore
  } catch (err) {
    error.value = errorMessage(err, '加载更多评论失败')
  } finally {
    loadingMore.value = false
  }
}
function created(comment: Comment) {
  items.value.unshift(comment)
  emit('changed')
}
async function reaction(comment: Comment, emoji: string) {
  if (reacting.value.has(comment.id)) return
  reacting.value.add(comment.id)
  error.value = null
  try {
    const result = await toggleReaction(comment.id, emoji)
    const target = items.value.find((item) => item.id === comment.id)
    if (target) target.reactions = result.reactions
  } catch (err) {
    error.value = errorMessage(err, '表情操作失败')
  } finally {
    reacting.value.delete(comment.id)
  }
}
function askDelete(comment: Comment) {
  deleteTarget.value = comment
  deleteError.value = null
}
async function confirmDelete() {
  if (!deleteTarget.value || deleting.value) return
  const id = deleteTarget.value.id
  deleting.value = true
  deleteError.value = null
  try {
    await deleteComment(id)
    items.value = items.value.filter((item) => item.id !== id)
    deleteTarget.value = null
    emit('changed')
  } catch (err) {
    deleteError.value = errorMessage(err, '删除评论失败')
  } finally {
    deleting.value = false
  }
}
watch(() => props.nodeId, load, { immediate: true })
</script>
<template>
  <section class="comments">
    <div class="comments-heading">
      <NH2
        >评论
        <NText v-if="!loading && !loadError" depth="3" class="comment-count"
          >{{ items.length }}{{ hasMore ? '+' : '' }}</NText
        ></NH2
      ><NText depth="3">最新在前</NText>
    </div>
    <CommentComposer
      v-if="canComment"
      :key="nodeId"
      :node-id="nodeId"
      @created="created"
    /><NAlert v-else type="info"
      >如需发表评论，请联系负责人将你添加为成员。</NAlert
    ><NAlert v-if="error" type="error" role="alert">{{ error }}</NAlert
    ><NCard size="small"
      ><NSpin :show="loading"
        ><NAlert v-if="loadError" type="error"
          >{{ loadError }}
          <NButton text @click="load">重新加载</NButton></NAlert
        ><NEmpty
          v-else-if="!loading && !items.length"
          description="暂无评论" /><NList v-else
          ><CommentItem
            v-for="comment in items"
            :key="comment.id"
            :comment="comment"
            :busy="reacting.has(comment.id)"
            @delete="askDelete"
            @reaction="reaction" /></NList></NSpin
      ><NSpace v-if="hasMore" justify="center"
        ><NButton :loading="loadingMore" @click="loadMore"
          >加载更多评论</NButton
        ></NSpace
      ></NCard
    ><ConfirmAction
      :open="deleteTarget !== null"
      title="删除评论"
      confirm-text="删除评论"
      :busy="deleting"
      :error="deleteError"
      @confirm="confirmDelete"
      @close="deleteTarget = null"
      ><p>图片附件也会一并删除，操作不可撤销。</p>
      <p v-if="deleteTarget?.body" class="delete-preview">
        {{ deleteTarget.body }}
      </p></ConfirmAction
    >
  </section>
</template>
<style scoped>
.comments {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.comments-heading {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
}
.comments-heading h2 {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
}
.comment-count {
  margin-left: 6px;
  font-size: 13px;
  font-weight: 400;
}
.delete-preview {
  white-space: pre-wrap;
  overflow-wrap: anywhere;
  max-height: 120px;
  overflow: auto;
}
</style>
