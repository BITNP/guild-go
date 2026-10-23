<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { Loader2 } from '@lucide/vue'
import BaseModal from '@/components/BaseModal.vue'
import CommentComposer from '@/components/CommentComposer.vue'
import CommentItem from '@/components/CommentItem.vue'
import {
  deleteComment,
  fetchComments,
  toggleReaction,
  type Comment,
} from '@/lib/api'
import { errorMessage } from '@/lib/utils'

const props = defineProps<{
  nodeId: number
  canComment: boolean
}>()

const emit = defineEmits<{
  changed: []
}>()

const items = ref<Comment[]>([])
const loading = ref(true)
const error = ref<string | null>(null)
const loadError = ref<string | null>(null)
const hasMore = ref(false)
const nextBefore = ref<number | null>(null)
const loadingMore = ref(false)
const deleteTarget = ref<Comment | null>(null)
const deleting = ref(false)
const deleteError = ref<string | null>(null)

async function load() {
  loading.value = true
  loadError.value = null
  try {
    const page = await fetchComments(props.nodeId)
    items.value = page.items
    hasMore.value = page.hasMore
    nextBefore.value = page.nextBefore
  } catch (err) {
    loadError.value = errorMessage(err, '无法加载评论')
  } finally {
    loading.value = false
  }
}

async function loadMore() {
  if (loadingMore.value || nextBefore.value == null) return
  loadingMore.value = true
  error.value = null
  try {
    const page = await fetchComments(props.nodeId, nextBefore.value)
    items.value = [...items.value, ...page.items]
    hasMore.value = page.hasMore
    nextBefore.value = page.nextBefore
  } catch (err) {
    error.value = errorMessage(err, '加载更多评论失败')
  } finally {
    loadingMore.value = false
  }
}

function onCreated(comment: Comment) {
  items.value = [comment, ...items.value]
  emit('changed')
}

async function onReaction(comment: Comment, emoji: string) {
  try {
    const result = await toggleReaction(comment.id, emoji)
    const target = items.value.find((item) => item.id === comment.id)
    if (target) target.reactions = result.reactions
  } catch (err) {
    error.value = errorMessage(err, '表情操作失败')
  }
}

function askDelete(comment: Comment) {
  deleteError.value = null
  deleteTarget.value = comment
}

async function confirmDelete() {
  if (!deleteTarget.value || deleting.value) return
  deleting.value = true
  deleteError.value = null
  try {
    await deleteComment(deleteTarget.value.id)
    items.value = items.value.filter((item) => item.id !== deleteTarget.value?.id)
    deleteTarget.value = null
    emit('changed')
  } catch (err) {
    deleteError.value = errorMessage(err, '删除评论失败')
  } finally {
    deleting.value = false
  }
}

onMounted(load)
watch(() => props.nodeId, load)

function closeDelete() {
  if (deleting.value) return
  deleteTarget.value = null
}
</script>

<template>
  <section class="flex flex-col gap-2">
    <div class="flex items-center justify-between">
      <h2 class="section-label">评论 <span v-if="!loading && !loadError" class="ml-1 font-normal text-muted">{{ items.length }}{{ hasMore ? '+' : '' }}</span></h2>
      <span class="text-xs text-faint">最新在前</span>
    </div>

    <CommentComposer v-if="canComment" :key="nodeId" :node-id="nodeId" @created="onCreated" />
    <p v-else class="rounded-md border border-line bg-surface-2 px-3 py-2 text-xs text-muted">
      如需发表评论，请联系负责人将你添加为成员。
    </p>

    <p v-if="error" role="alert" class="rounded-md border border-danger/20 bg-danger-soft px-3 py-1.5 text-xs text-danger">{{ error }}</p>

    <div class="rounded-lg border border-line bg-surface">
      <p v-if="loading" class="empty-state"><Loader2 class="h-4 w-4 animate-spin" /> 正在加载评论…</p>
      <p v-else-if="loadError" class="empty-state text-danger" role="alert">{{ loadError }} <button type="button" class="btn mt-2" @click="load">重新加载</button></p>
      <p v-else-if="items.length === 0" class="empty-state">暂无评论</p>
      <template v-else>
        <CommentItem
          v-for="comment in items"
          :key="comment.id"
          :comment="comment"
          @delete="askDelete"
          @reaction="onReaction"
        />
        <div v-if="hasMore" class="border-t border-line px-3 py-2 text-center">
          <button type="button" class="btn btn-xs" :disabled="loadingMore" @click="loadMore">
            <Loader2 v-if="loadingMore" class="h-3 w-3 animate-spin" />
            {{ loadingMore ? '加载中…' : '加载更多评论' }}
          </button>
        </div>
      </template>
    </div>

    <BaseModal
      :open="deleteTarget !== null"
      title="删除评论"
      :busy="deleting"
      @close="closeDelete"
    >
      <p class="text-[13px] text-ink-soft">确定删除这条评论吗？图片附件也会一并删除，操作不可撤销。</p>
      <p v-if="deleteTarget?.body" class="mt-2 rounded-md border border-line bg-surface-2 px-2.5 py-2 text-[12px] text-muted line-clamp-3">
        {{ deleteTarget.body }}
      </p>
      <p v-if="deleteError" role="alert" class="mt-2 text-xs text-danger">{{ deleteError }}</p>
      <template #footer>
        <button type="button" class="btn" :disabled="deleting" @click="closeDelete">取消</button>
        <button type="button" class="btn btn-danger" :disabled="deleting" @click="confirmDelete">
          <Loader2 v-if="deleting" class="h-3.5 w-3.5 animate-spin" />
          {{ deleting ? '删除中…' : '删除' }}
        </button>
      </template>
    </BaseModal>
  </section>
</template>
