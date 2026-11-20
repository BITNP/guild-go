<script setup lang="ts">
import { FileText, MessageSquare, Trash2, X } from "@lucide/vue";
import { computed, onBeforeUnmount, onMounted, ref } from "vue";

import ConfirmAction from "@/components/ConfirmAction.vue";
import DescriptionDiff from "@/components/DescriptionDiff.vue";
import ImageGallery from "@/components/ImageGallery.vue";
import PreviewImage from "@/components/PreviewImage.vue";
import UserIdentity from "@/components/UserIdentity.vue";
import type { ProgressItem } from "@/lib/api";
import { deleteComment } from "@/lib/api";
import {
  errorMessage,
  formatDateTime,
  linkify,
  reactionEmoji,
  reactionKeys,
} from "@/lib/utils";

const emit = defineEmits<{
  close: [];
  reaction: [commentId: number, emoji: string];
  deleted: [commentId: number];
}>();

const props = defineProps<{
  item: ProgressItem;
  nodeTitle: string;
  busy: boolean;
}>();

const deleteOpen = ref(false);
const deleting = ref(false);
const deleteError = ref<string | null>(null);

const comment = computed(() =>
  props.item.kind === "comment" ? props.item.comment : null,
);
const change = computed(() =>
  props.item.kind === "description_change"
    ? props.item.descriptionChange
    : null,
);
const author = computed(() => comment.value?.author ?? change.value?.author);
const createdAt = computed(
  () => comment.value?.createdAt ?? change.value?.createdAt ?? "",
);
const commentSegments = computed(() => linkify(comment.value?.body ?? ""));
const changeAction = computed(() => {
  if (!change.value) {
    return "";
  }
  if (!change.value.before) {
    return "添加了节点描述";
  }
  if (!change.value.after) {
    return "清空了节点描述";
  }

  return "修改了节点描述";
});

function askDelete() {
  deleteError.value = null;
  deleteOpen.value = true;
}

async function confirmDelete() {
  if (!comment.value || deleting.value) {
    return;
  }
  deleting.value = true;
  deleteError.value = null;
  try {
    await deleteComment(comment.value.id);
    deleteOpen.value = false;
    emit("deleted", comment.value.id);
  } catch (err) {
    deleteError.value = errorMessage(err, "删除进展失败");
  } finally {
    deleting.value = false;
  }
}

function onKeydown(event: KeyboardEvent) {
  if (
    event.key !== "Escape" ||
    event.defaultPrevented ||
    deleteOpen.value ||
    document.querySelector(".n-modal-container, .n-image-preview-container")
  ) {
    return;
  }
  emit("close");
}

onMounted(() => window.addEventListener("keydown", onKeydown));
onBeforeUnmount(() => window.removeEventListener("keydown", onKeydown));
</script>

<template>
  <div class="progress-detail-backdrop" @click.self="emit('close')">
    <article
      aria-labelledby="progress-detail-title"
      aria-modal="true"
      class="progress-detail"
      data-graph-progress-detail
      role="dialog"
    >
      <header class="progress-detail-header">
        <span aria-hidden="true" class="progress-detail-mark">
          <MessageSquare v-if="comment" :size="17" />
          <FileText v-else :size="17" />
        </span>
        <div>
          <span class="progress-detail-kicker">
            {{ comment ? "进展" : "描述变更" }}
          </span>
          <strong id="progress-detail-title">{{ nodeTitle }}</strong>
        </div>
        <button
          aria-label="关闭进展详情"
          class="progress-detail-close"
          type="button"
          @click="emit('close')"
        >
          <X :size="17" />
        </button>
      </header>

      <div class="progress-detail-meta">
        <UserIdentity v-if="author" show-position :size="24" :user="author" />
        <div class="progress-detail-meta-right">
          <time :datetime="createdAt" :title="formatDateTime(createdAt)">
            {{ formatDateTime(createdAt) }}
          </time>
          <button
            v-if="comment?.canDelete"
            class="progress-detail-delete"
            :disabled="busy || deleting"
            type="button"
            @click="askDelete"
          >
            <Trash2 aria-hidden="true" :size="14" />
            <span>删除</span>
          </button>
        </div>
      </div>

      <div class="progress-detail-body">
        <template v-if="comment">
          <p v-if="comment.body.trim()">
            <template v-for="(segment, index) in commentSegments" :key="index">
              <a
                v-if="segment.type === 'link'"
                :href="segment.href"
                rel="noopener noreferrer"
                target="_blank"
              >
                {{ segment.value }}
              </a>
              <template v-else>{{ segment.value }}</template>
            </template>
          </p>
          <p
            v-else-if="comment.attachments.length === 0"
            class="progress-detail-empty"
          >
            无正文
          </p>

          <ImageGallery
            v-if="comment.attachments.length > 0"
            class="progress-detail-gallery"
          >
            <div class="progress-detail-images">
              <PreviewImage
                v-for="(image, index) in comment.attachments"
                :key="image.id"
                :alt="`进展图片 ${index + 1}`"
                class="progress-detail-image"
                :height="100"
                lazy
                :src="image.url"
                :width="132"
              />
            </div>
          </ImageGallery>
        </template>

        <template v-else-if="change">
          <p class="progress-change-action">{{ changeAction }}</p>
          <DescriptionDiff
            :after="change.after"
            :before="change.before"
            class="progress-change-diff"
          />
        </template>
      </div>

      <footer v-if="comment" class="progress-detail-reactions">
        <span>回应</span>
        <button
          v-for="emoji in reactionKeys"
          :key="emoji"
          :aria-label="reactionEmoji[emoji]?.label"
          :class="{
            active: comment.reactions.some(
              (reaction) => reaction.emoji === emoji && reaction.reactedByMe,
            ),
          }"
          :disabled="busy"
          type="button"
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
      :busy="deleting"
      confirm-text="删除进展"
      :error="deleteError"
      :open="deleteOpen"
      title="删除进展"
      @close="deleteOpen = false"
      @confirm="confirmDelete"
    >
      <p>
        {{
          comment?.attachments.length
            ? `附带的 ${comment.attachments.length} 张图片也会一并删除，操作不可撤销。`
            : "删除后不可撤销。"
        }}
      </p>
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
  background: var(--graph-backdrop-bg);
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
  border: 1px solid var(--graph-detail-border);
  border-radius: 8px;
  background: var(--graph-detail-bg);
  box-shadow: var(--graph-detail-shadow);
  color: var(--graph-detail-text);
}

.progress-detail-header {
  display: flex;
  align-items: center;
  gap: 11px;
  padding: 16px 18px;
  border-bottom: 1px solid var(--graph-panel-divider);
}

.progress-detail-mark {
  display: grid;
  width: 34px;
  height: 34px;
  flex: none;
  place-items: center;
  border: 1px solid var(--graph-detail-mark-border);
  border-radius: 6px;
  color: var(--graph-detail-mark-color);
}

.progress-detail-header > div {
  display: grid;
  min-width: 0;
  flex: 1 1 auto;
  gap: 2px;
}

.progress-detail-kicker {
  color: var(--graph-detail-kicker);
  font-size: 12px;
  font-weight: 600;
}

.progress-detail-header strong {
  overflow: hidden;
  color: var(--graph-detail-title);
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
  border: 1px solid var(--graph-close-border);
  border-radius: 4px;
  background: var(--graph-close-bg);
  color: var(--graph-close-color);
  cursor: pointer;
}

.progress-detail-close:hover {
  border-color: var(--graph-close-hover-border);
  color: var(--graph-close-hover-color);
}

.progress-detail-meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 8px 18px;
  padding: 10px 20px;
  border-bottom: 1px solid var(--graph-panel-divider);
  color: var(--graph-detail-meta);
  font-size: 13px;
}

.progress-detail-meta :deep(.user-name) {
  color: var(--graph-detail-user-name);
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
  border: 1px solid var(--graph-danger-border);
  border-radius: 4px;
  background: transparent;
  color: var(--graph-danger-color);
  font-size: 12px;
  cursor: pointer;
}

.progress-detail-delete:hover:not(:disabled) {
  border-color: var(--graph-danger-hover-border);
  background: var(--graph-danger-hover-bg);
  color: var(--graph-danger-hover-color);
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
  color: var(--graph-detail-body);
  font-size: 15px;
  line-height: 1.85;
  overflow-wrap: anywhere;
  white-space: pre-wrap;
}

.progress-detail-body a {
  color: var(--graph-content-link-color);
  text-decoration: underline;
  text-underline-offset: 3px;
}

.progress-detail-empty {
  color: var(--graph-detail-empty);
}

.progress-change-action {
  margin-bottom: 12px;
  color: var(--graph-detail-change-action);
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
  border-top: 1px solid var(--graph-panel-divider);
  color: var(--graph-detail-meta);
  font-size: 13px;
}

.progress-detail-reactions button {
  display: inline-flex;
  min-width: 45px;
  height: 31px;
  align-items: center;
  justify-content: center;
  gap: 4px;
  border: 1px solid var(--graph-reaction-btn-border);
  border-radius: 4px;
  background: var(--graph-reaction-btn-bg);
  color: var(--graph-reaction-btn-color);
  font-size: 16px;
  cursor: pointer;
}

.progress-detail-reactions button:hover:not(:disabled) {
  border-color: var(--graph-reaction-btn-hover-border);
  background: var(--graph-reaction-btn-hover-bg);
}

.progress-detail-reactions button.active {
  border-color: var(--graph-reaction-active-border);
  background: var(--graph-reaction-active-bg);
}

.progress-detail-reactions button:disabled {
  cursor: wait;
  opacity: 0.55;
}

.progress-detail-reactions small {
  color: var(--graph-reaction-count-color);
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

@media (prefers-reduced-motion: reduce) {
  .progress-detail-enter-active,
  .progress-detail-leave-active,
  .progress-detail-enter-active .progress-detail,
  .progress-detail-leave-active .progress-detail {
    transition: none;
  }
}
</style>
