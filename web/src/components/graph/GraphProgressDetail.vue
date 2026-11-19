<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { FileText, MessageSquare, Trash2, X } from '@lucide/vue'
import ConfirmAction from '@/components/ConfirmAction.vue'
import DescriptionDiff from '@/components/DescriptionDiff.vue'
import ImageGallery from '@/components/ImageGallery.vue'
import PreviewImage from '@/components/PreviewImage.vue'
import UserIdentity from '@/components/UserIdentity.vue'
import { deleteComment, type ProgressItem } from '@/lib/api'
import {
  errorMessage,
  formatDateTime,
  linkify,
  reactionEmoji,
  reactionKeys,
} from '@/lib/utils'

const props = defineProps<{
  item: ProgressItem
  nodeTitle: string
  busy: boolean
}>()

const emit = defineEmits<{
  close: []
  reaction: [commentId: number, emoji: string]
  deleted: [commentId: number]
}>()

const deleteOpen = ref(false)
const deleting = ref(false)
const deleteError = ref<string | null>(null)

const comment = computed(() =>
  props.item.kind === 'comment' ? props.item.comment : null,
)
const change = computed(() =>
  props.item.kind === 'description_change'
    ? props.item.descriptionChange
    : null,
)
const author = computed(() => comment.value?.author ?? change.value?.author)
const createdAt = computed(() =>
  comment.value?.createdAt ?? change.value?.createdAt ?? '',
)
const commentSegments = computed(() => linkify(comment.value?.body ?? ''))
const changeAction = computed(() => {
  if (!change.value) return ''
  if (!change.value.before) return '添加了节点描述'
  if (!change.value.after) return '清空了节点描述'
  return '修改了节点描述'
})

function askDelete() {
  deleteError.value = null
  deleteOpen.value = true
}

async function confirmDelete() {
  if (!comment.value || deleting.value) return
  deleting.value = true
  deleteError.value = null
  try {
    await deleteComment(comment.value.id)
    deleteOpen.value = false
    emit('deleted', comment.value.id)
  } catch (err) {
    deleteError.value = errorMessage(err, '删除进展失败')
  } finally {
    deleting.value = false
  }
}

function onKeydown(event: KeyboardEvent) {
  if (
    event.key !== 'Escape' ||
    event.defaultPrevented ||
    deleteOpen.value ||
    document.querySelector('.n-modal-container, .n-image-preview-container')
  ) {
    return
  }
  emit('close')
}

onMounted(() => window.addEventListener('keydown', onKeydown))
onBeforeUnmount(() => window.removeEventListener('keydown', onKeydown))
</script>

<template>
  <div class="progress-detail-backdrop" @click.self="emit('close')">
    <article
      class="progress-detail"
      data-graph-progress-detail
      role="dialog"
      aria-modal="true"
      aria-labelledby="progress-detail-title"
    >
      <header class="progress-detail-header">
        <span class="progress-detail-mark" aria-hidden="true">
          <MessageSquare v-if="comment" :size="17" />
          <FileText v-else :size="17" />
        </span>
        <div>
          <span class="progress-detail-kicker">
            {{ comment ? '进展' : '描述变更' }}
          </span>
          <strong id="progress-detail-title">{{ nodeTitle }}</strong>
        </div>
        <button
          type="button"
          class="progress-detail-close"
          aria-label="关闭进展详情"
          @click="emit('close')"
        >
          <X :size="17" />
        </button>
      </header>

      <div class="progress-detail-meta">
        <UserIdentity v-if="author" :user="author" :size="24" show-position />
        <div class="progress-detail-meta-right">
          <time :datetime="createdAt" :title="formatDateTime(createdAt)">
            {{ formatDateTime(createdAt) }}
          </time>
          <button
            v-if="comment?.canDelete"
            type="button"
            class="progress-detail-delete"
            :disabled="busy || deleting"
            @click="askDelete"
          >
            <Trash2 :size="14" aria-hidden="true" />
            <span>删除</span>
          </button>
        </div>
      </div>

      <div class="progress-detail-body">
        <template v-if="comment">
          <p v-if="comment.body.trim()">
            <template
              v-for="(segment, index) in commentSegments"
              :key="index"
              ><a
                v-if="segment.type === 'link'"
                :href="segment.href"
                target="_blank"
                rel="noopener noreferrer"
                >{{ segment.value }}</a
              ><template v-else>{{ segment.value }}</template></template
            >
          </p>
          <p v-else-if="!comment.attachments.length" class="progress-detail-empty">
            无正文
          </p>

          <ImageGallery
            v-if="comment.attachments.length"
            class="progress-detail-gallery"
          >
            <div class="progress-detail-images">
              <PreviewImage
                v-for="(image, index) in comment.attachments"
                :key="image.id"
                :src="image.url"
                :alt="`进展图片 ${index + 1}`"
                :width="132"
                :height="100"
                lazy
                class="progress-detail-image"
              />
            </div>
          </ImageGallery>
        </template>

        <template v-else-if="change">
          <p class="progress-change-action">{{ changeAction }}</p>
          <DescriptionDiff
            :before="change.before"
            :after="change.after"
            class="progress-change-diff"
          />
        </template>
      </div>

      <footer v-if="comment" class="progress-detail-reactions">
        <span>回应</span>
        <button
          v-for="emoji in reactionKeys"
          :key="emoji"
          type="button"
          :class="{
            active: comment.reactions.some(
              (reaction) => reaction.emoji === emoji && reaction.reactedByMe,
            ),
          }"
          :disabled="busy"
          :aria-label="reactionEmoji[emoji]?.label"
          @click="emit('reaction', comment.id, emoji)"
        >
          {{ reactionEmoji[emoji]?.symbol }}
          <small>
            {{
              comment.reactions.find((reaction) => reaction.emoji === emoji)
                ?.count ?? 0
            }}
          </small>
        </button>
      </footer>
    </article>

    <ConfirmAction
      :open="deleteOpen"
      title="删除进展"
      confirm-text="删除进展"
      :busy="deleting"
      :error="deleteError"
      @confirm="confirmDelete"
      @close="deleteOpen = false"
    >
      <p>图片附件也会一并删除，操作不可撤销。</p>
      <p v-if="comment?.body" class="delete-preview">
        {{ comment.body }}
      </p>
    </ConfirmAction>
  </div>
</template>

<style scoped>
.progress-detail-backdrop {
  position: absolute;
  inset: 0;
  z-index: 24;
  display: grid;
  place-items: center;
  padding: 18px;
  background: rgba(2, 7, 9, 0.42);
  backdrop-filter: blur(5px);
}

@media (min-width: 981px) {
  .progress-detail-backdrop {
    padding: 24px 430px 24px 24px;
  }
}

.progress-detail {
  --detail-accent: #e5b451;
  display: flex;
  width: min(680px, 100%);
  max-height: min(72vh, 720px);
  flex-direction: column;
  overflow: hidden;
  border: 1px solid rgba(226, 178, 76, 0.42);
  border-radius: 8px;
  background:
    linear-gradient(135deg, rgba(232, 181, 75, 0.09), transparent 32%),
    linear-gradient(180deg, rgba(9, 17, 20, 0.98), rgba(4, 10, 13, 0.99));
  box-shadow:
    0 30px 90px rgba(0, 0, 0, 0.6),
    inset 0 0 44px rgba(77, 177, 175, 0.025);
  color: #e7eeeb;
}

.progress-detail-header {
  display: flex;
  align-items: center;
  gap: 11px;
  padding: 16px 18px;
  border-bottom: 1px solid rgba(139, 166, 171, 0.18);
}

.progress-detail-mark {
  display: grid;
  width: 34px;
  height: 34px;
  flex: none;
  place-items: center;
  border: 1px solid rgba(231, 181, 76, 0.44);
  border-radius: 6px;
  color: #edc469;
}

.progress-detail-header > div {
  display: grid;
  min-width: 0;
  flex: 1 1 auto;
  gap: 2px;
}

.progress-detail-kicker {
  color: rgba(120, 222, 198, 0.8);
  font-size: 12px;
  font-weight: 600;
}

.progress-detail-header strong {
  overflow: hidden;
  color: #f2eee3;
  font-size: 16px;
  font-weight: 650;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.progress-detail-close {
  display: grid;
  width: 30px;
  height: 30px;
  flex: none;
  place-items: center;
  border: 1px solid rgba(139, 166, 171, 0.24);
  border-radius: 4px;
  background: rgba(8, 15, 18, 0.7);
  color: #b7c5c5;
  cursor: pointer;
}

.progress-detail-close:hover {
  border-color: rgba(238, 187, 78, 0.7);
  color: #f2ca78;
}

.progress-detail-meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 8px 18px;
  padding: 10px 20px;
  border-bottom: 1px solid rgba(139, 166, 171, 0.12);
  color: #8fa4a7;
  font-size: 13px;
}

.progress-detail-meta :deep(.user-name) {
  color: #e7eeeb;
  font-weight: 600;
}

.progress-detail-meta-right {
  display: inline-flex;
  align-items: center;
  gap: 12px;
  margin-left: auto;
}

.progress-detail-delete {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 3px 8px;
  border: 1px solid rgba(226, 125, 104, 0.34);
  border-radius: 4px;
  background: transparent;
  color: #e3a091;
  font-size: 12px;
  cursor: pointer;
}

.progress-detail-delete:hover:not(:disabled) {
  border-color: rgba(241, 143, 119, 0.7);
  background: rgba(177, 69, 50, 0.12);
  color: #ffb9a8;
}

.progress-detail-delete:disabled {
  cursor: wait;
  opacity: 0.55;
}

.progress-detail-body {
  min-height: 0;
  overflow-y: auto;
  padding: 18px 20px 22px;
  scrollbar-color: rgba(221, 176, 75, 0.45) transparent;
  scrollbar-width: thin;
}

.progress-detail-body > p {
  margin: 0;
  color: #e5ebe7;
  font-size: 15px;
  line-height: 1.85;
  overflow-wrap: anywhere;
  white-space: pre-wrap;
}

.progress-detail-body a {
  color: #f0c975;
  text-decoration: underline;
  text-underline-offset: 3px;
}

.progress-detail-empty {
  color: #8fa4a7;
}

.progress-change-action {
  margin-bottom: 12px;
  color: #b7c8c9;
  font-size: 14px;
}

.progress-change-diff {
  margin-top: 12px;
}

.progress-detail-gallery {
  display: block;
  margin-top: 16px;
}

.progress-detail-images {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(112px, 1fr));
  gap: 8px;
}

.progress-detail-image {
  width: 132px;
  height: 100px;
  border: 1px solid rgba(139, 166, 171, 0.26);
  border-radius: 3px;
  overflow: hidden;
}

.delete-preview {
  white-space: pre-wrap;
  overflow-wrap: anywhere;
  max-height: 120px;
  overflow: auto;
}

.progress-detail-reactions {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px;
  padding: 12px 18px;
  border-top: 1px solid rgba(139, 166, 171, 0.18);
  color: #7f9699;
  font-size: 13px;
}

.progress-detail-reactions button {
  display: inline-flex;
  min-width: 45px;
  height: 31px;
  align-items: center;
  justify-content: center;
  gap: 4px;
  border: 1px solid rgba(107, 142, 147, 0.3);
  border-radius: 4px;
  background: rgba(16, 25, 27, 0.74);
  color: #e3e9e4;
  font-size: 16px;
  cursor: pointer;
}

.progress-detail-reactions button:hover:not(:disabled) {
  border-color: rgba(239, 188, 80, 0.52);
}

.progress-detail-reactions button.active {
  border-color: rgba(239, 188, 80, 0.7);
  background: rgba(221, 170, 61, 0.16);
}

.progress-detail-reactions button:disabled {
  cursor: wait;
  opacity: 0.55;
}

.progress-detail-reactions small {
  color: #9bb1b2;
  font-size: 12px;
  font-variant-numeric: tabular-nums;
}

.progress-detail-enter-active,
.progress-detail-leave-active {
  transition: opacity 180ms ease;
}

.progress-detail-enter-active .progress-detail,
.progress-detail-leave-active .progress-detail {
  transition: transform 260ms cubic-bezier(0.2, 0.8, 0.24, 1);
}

.progress-detail-enter-from,
.progress-detail-leave-to {
  opacity: 0;
}

.progress-detail-enter-from .progress-detail,
.progress-detail-leave-to .progress-detail {
  transform: translateY(16px) scale(0.97);
}

:global([data-theme='light'] .progress-detail-backdrop) {
  background: rgba(220, 228, 232, 0.42);
}

:global([data-theme='light'] .progress-detail) {
  border-color: rgba(158, 103, 18, 0.3);
  background:
    linear-gradient(135deg, rgba(193, 128, 29, 0.07), transparent 32%),
    linear-gradient(180deg, rgba(255, 255, 255, 0.99), rgba(245, 248, 249, 0.99));
  box-shadow:
    0 30px 90px rgba(45, 60, 69, 0.24),
    inset 0 0 44px rgba(55, 137, 135, 0.02);
  color: #273740;
}

:global([data-theme='light'] .progress-detail-header),
:global([data-theme='light'] .progress-detail-meta) {
  border-color: rgba(81, 105, 116, 0.16);
}

:global([data-theme='light'] .progress-detail-header strong) {
  color: #17242c;
}

:global([data-theme='light'] .progress-detail-mark) {
  border-color: rgba(153, 99, 16, 0.36);
  color: #8b5a10;
}

:global([data-theme='light'] .progress-detail-kicker) {
  color: #207a63;
}

:global([data-theme='light'] .progress-detail-meta) {
  color: #667b85;
}

:global([data-theme='light'] .progress-detail-meta .user-name) {
  color: #17242c;
}

:global([data-theme='light'] .progress-detail-delete) {
  border-color: rgba(194, 59, 59, 0.32);
  color: #a33333;
}

:global([data-theme='light'] .progress-detail-delete:hover:not(:disabled)) {
  border-color: rgba(194, 59, 59, 0.65);
  background: rgba(194, 59, 59, 0.08);
  color: #8a1f1f;
}

:global([data-theme='light'] .progress-detail-close) {
  border-color: rgba(78, 105, 117, 0.24);
  background: rgba(255, 255, 255, 0.78);
  color: #546b75;
}

:global([data-theme='light'] .progress-detail-close:hover) {
  border-color: rgba(155, 101, 17, 0.55);
  color: #7c4d09;
}

:global([data-theme='light'] .progress-detail-body > p) {
  color: #24343d;
}

:global([data-theme='light'] .progress-detail-empty) {
  color: #667b85;
}

:global([data-theme='light'] .progress-detail-body a) {
  color: var(--guild-primary);
}

:global([data-theme='light'] .progress-change-action) {
  color: #4f6470;
}

:global([data-theme='light'] .progress-detail-reactions) {
  border-top-color: rgba(80, 105, 116, 0.16);
  color: #687d87;
}

:global([data-theme='light'] .progress-detail-reactions button) {
  border-color: rgba(82, 108, 120, 0.24);
  background: rgba(255, 255, 255, 0.76);
  color: #283942;
}

:global([data-theme='light'] .progress-detail-reactions button:hover:not(:disabled)) {
  border-color: rgba(156, 100, 16, 0.52);
  background: rgba(255, 250, 239, 0.96);
}

:global([data-theme='light'] .progress-detail-reactions small) {
  color: #556a75;
}

:global([data-theme='light'] .progress-detail-reactions button.active) {
  border-color: rgba(156, 100, 16, 0.52);
  background: rgba(190, 126, 29, 0.1);
}

@media (prefers-reduced-motion: reduce) {
  .progress-detail-enter-active,
  .progress-detail-leave-active,
  .progress-detail-enter-active .progress-detail,
  .progress-detail-leave-active .progress-detail {
    transition: none;
  }
}
</style>
