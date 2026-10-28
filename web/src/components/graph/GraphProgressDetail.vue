<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted } from 'vue'
import { FileText, MessageSquare, X } from '@lucide/vue'
import type { ProgressItem } from '@/lib/api'
import {
  formatDateTime,
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
}>()

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
const body = computed(() => {
  if (comment.value) {
    return comment.value.body.trim()
      ? comment.value.body
      : comment.value.attachments.length
        ? '图片进展'
        : '无正文'
  }
  return change.value?.after.trim() ? change.value.after : '清空了节点描述'
})

function onKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape') emit('close')
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
        <span>{{ author?.userName }}</span>
        <time :datetime="createdAt" :title="formatDateTime(createdAt)">
          {{ formatDateTime(createdAt) }}
        </time>
      </div>

      <div class="progress-detail-body">
        <p>{{ body }}</p>

        <div v-if="comment?.attachments.length" class="progress-detail-images">
          <img
            v-for="(image, index) in comment.attachments"
            :key="image.id"
            :src="image.url"
            :alt="`进展图片 ${index + 1}`"
            loading="lazy"
          />
        </div>

        <details v-if="change" class="progress-before">
          <summary>查看修改前内容</summary>
          <p>{{ change.before || '此前没有节点描述' }}</p>
        </details>
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
  </div>
</template>

<style scoped>
.progress-detail-backdrop {
  position: absolute;
  inset: 0;
  z-index: 24;
  display: grid;
  place-items: center;
  padding: 28px 150px 28px 28px;
  background: rgba(2, 7, 9, 0.42);
  backdrop-filter: blur(5px);
}

.progress-detail {
  --detail-accent: #e5b451;
  display: flex;
  width: min(680px, 100%);
  max-height: min(72vh, 720px);
  flex-direction: column;
  overflow: hidden;
  border: 1px solid rgba(226, 178, 76, 0.42);
  border-left-color: rgba(240, 188, 79, 0.82);
  border-radius: 3px;
  background:
    linear-gradient(135deg, rgba(232, 181, 75, 0.09), transparent 32%),
    linear-gradient(180deg, rgba(9, 17, 20, 0.98), rgba(4, 10, 13, 0.99));
  box-shadow:
    0 30px 90px rgba(0, 0, 0, 0.6),
    inset 0 0 44px rgba(77, 177, 175, 0.025);
  color: #e7eeeb;
  transform: perspective(1400px) rotateX(0.7deg) rotateY(-1.1deg);
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
  font-size: 10px;
  font-weight: 700;
  letter-spacing: 0.14em;
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
  justify-content: space-between;
  gap: 8px 18px;
  padding: 10px 20px;
  border-bottom: 1px solid rgba(139, 166, 171, 0.12);
  color: #8fa4a7;
  font-size: 11px;
}

.progress-detail-body {
  min-height: 0;
  overflow-y: auto;
  padding: 18px 20px 22px;
  scrollbar-color: rgba(221, 176, 75, 0.45) transparent;
  scrollbar-width: thin;
}

.progress-detail-body > p,
.progress-before p {
  margin: 0;
  color: #e5ebe7;
  font-size: 15px;
  line-height: 1.85;
  overflow-wrap: anywhere;
  white-space: pre-wrap;
}

.progress-detail-images {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(112px, 1fr));
  gap: 8px;
  margin-top: 16px;
}

.progress-detail-images img {
  width: 100%;
  max-height: 190px;
  border: 1px solid rgba(139, 166, 171, 0.26);
  border-radius: 3px;
  object-fit: cover;
}

.progress-before {
  margin-top: 18px;
  border-top: 1px solid rgba(139, 166, 171, 0.16);
  padding-top: 12px;
}

.progress-before summary {
  color: #d8b96e;
  font-size: 12px;
  cursor: pointer;
}

.progress-before p {
  margin-top: 10px;
  color: #aebcbd;
  font-size: 13px;
}

.progress-detail-reactions {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px;
  padding: 12px 18px;
  border-top: 1px solid rgba(139, 166, 171, 0.18);
  color: #7f9699;
  font-size: 10px;
}

.progress-detail-reactions button {
  display: inline-flex;
  min-width: 45px;
  height: 31px;
  align-items: center;
  justify-content: center;
  gap: 4px;
  border: 1px solid rgba(107, 142, 147, 0.3);
  border-radius: 2px;
  background: rgba(16, 25, 27, 0.74);
  color: #e3e9e4;
  font-size: 16px;
  cursor: pointer;
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
  font-size: 9px;
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
  transform: perspective(1400px) translateY(16px) scale(0.97);
}

@media (max-width: 800px) {
  .progress-detail-backdrop {
    padding: 18px;
  }
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
