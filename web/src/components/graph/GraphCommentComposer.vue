<script setup lang="ts">
import { ImagePlus, Loader2, RotateCcw, Send, X } from "@lucide/vue";
import { ref, useId } from "vue";

import { useCommentComposer } from "@/composables/useCommentComposer";
import {
  COMMENT_IMAGE_ACCEPT,
  COMMENT_IMAGE_LIMIT,
} from "@/composables/useCommentImageDrafts";
import type { Comment } from "@/lib/api";

const emit = defineEmits<{ created: [comment: Comment] }>();

const props = defineProps<{
  nodeId: number;
}>();

const fileInput = ref<HTMLInputElement | null>(null);
const textarea = ref<HTMLTextAreaElement | null>(null);
const bodyId = useId();
const {
  body,
  submitting,
  error,
  dragging,
  images,
  pending,
  retry,
  remove,
  selectFiles,
  pasteImages,
  dragEnter,
  dragOver,
  dragLeave,
  dropFiles,
  submit,
} = useCommentComposer(() => props.nodeId, {
  emptyMessage: "写下一步进展或添加一张图片",
  submitErrorFallback: "进展发送失败，请重试",
  onCreated: (comment) => emit("created", comment),
});

function shortcut(event: KeyboardEvent) {
  if (
    event.key !== "Enter" ||
    (!event.ctrlKey && !event.metaKey) ||
    event.isComposing ||
    event.keyCode === 229
  ) {
    return;
  }
  event.preventDefault();
  void submit();
}

function focusComposer(event: PointerEvent) {
  event.stopPropagation();
  const target = event.target;
  if (target instanceof HTMLTextAreaElement) {
    return;
  }
  if (target instanceof Element && target.closest("button, input")) {
    return;
  }
  event.preventDefault();
  textarea.value?.focus({ preventScroll: true });
}
</script>

<template>
  <section
    aria-label="输入进展"
    class="graph-comment-composer"
    data-graph-comment-composer
    @click.stop
    @dblclick.stop
    @dragenter="dragEnter"
    @dragleave="dragLeave"
    @dragover="dragOver"
    @drop="dropFiles"
    @pointerdown.capture="focusComposer"
    @pointermove.stop
    @pointerup.stop
  >
    <textarea
      v-model="body"
      ref="textarea"
      :id="bodyId"
      :aria-invalid="!!error"
      aria-label="请输入进展"
      class="composer-input"
      :class="{ 'has-images': images.length > 0 }"
      :disabled="submitting"
      maxlength="5000"
      placeholder="输入进展，可粘贴或拖拽图片（Ctrl / ⌘ + Enter 发布）"
      rows="2"
      @keydown="shortcut"
      @paste="pasteImages"
    />

    <div v-if="images.length > 0" aria-live="polite" class="composer-images">
      <div v-for="image in images" :key="image.src" class="image-slot">
        <img :alt="image.file.name" :src="image.src" />
        <div
          v-if="image.status.state === 'uploading'"
          :aria-label="`正在上传：${image.file.name}`"
          class="image-overlay"
          role="progressbar"
        >
          <Loader2 class="spin" :size="18" />
        </div>
        <button
          v-else-if="image.status.state === 'failed'"
          :aria-label="`重试上传：${image.file.name}`"
          class="image-overlay image-retry"
          :title="image.status.message"
          type="button"
          @click="retry(image)"
        >
          <RotateCcw :size="16" />
        </button>
        <button
          v-if="image.status.state !== 'uploading'"
          :aria-label="`移除图片：${image.file.name}`"
          class="image-remove"
          :disabled="submitting"
          type="button"
          @click="remove(image)"
        >
          <X :size="14" />
        </button>
      </div>
    </div>

    <div class="composer-actions">
      <button
        aria-label="添加图片，也可粘贴或拖拽图片"
        class="attach-button"
        :disabled="submitting || images.length >= COMMENT_IMAGE_LIMIT"
        title="添加图片，也可粘贴或拖拽"
        type="button"
        @click="fileInput?.click()"
      >
        <ImagePlus aria-hidden="true" :size="20" />
        <span>添加图片 {{ images.length }}/{{ COMMENT_IMAGE_LIMIT }}</span>
      </button>
      <button
        :aria-label="
          submitting ? '正在发送进展' : pending ? '正在上传图片' : '发布进展'
        "
        class="send-button"
        :disabled="
          submitting || pending || (!body.trim() && images.length === 0)
        "
        :title="
          submitting ? '正在发送进展' : pending ? '正在上传图片' : '发布进展'
        "
        type="button"
        @click="submit"
      >
        <Loader2 v-if="submitting" aria-hidden="true" class="spin" :size="20" />
        <Send v-else aria-hidden="true" :size="20" />
        <span>{{ submitting ? "发送中…" : "发布" }}</span>
      </button>
    </div>

    <input
      ref="fileInput"
      :accept="COMMENT_IMAGE_ACCEPT"
      class="file-input"
      multiple
      type="file"
      @change="selectFiles"
    />
    <div v-if="dragging" aria-hidden="true" class="drop-hint">
      <ImagePlus :size="20" />
      <span>松开以添加图片</span>
    </div>
    <p v-if="error" class="composer-error" role="alert">
      {{ error }}
    </p>
  </section>
</template>

<style scoped>
/*
 * CSS3DRenderer owns this element's position and transform. The fixed logical
 * size keeps it aligned with the 6.2 x 2.36 comment-card plane in the scene.
 */
.graph-comment-composer {
  position: relative;
  display: flex;
  width: 620px;
  height: 260px;
  margin: 0;
  padding: 18px 22px 14px 22px;
  flex-direction: column;
  overflow: hidden;
  box-sizing: border-box;
  border: 1px solid var(--graph-panel-border);
  border-left: 3px solid var(--graph-detail-edge);
  border-radius: 6px;
  background: var(--graph-composer-bg);
  box-shadow: var(--graph-composer-shadow);
  color: var(--graph-composer-text);
  pointer-events: auto;
  backdrop-filter: blur(22px) saturate(120%);
  transform-origin: 50% 50%;
  visibility: hidden;
  opacity: 0;
}

.composer-input {
  display: block;
  width: 100%;
  min-height: 0;
  flex: 1 1 auto;
  resize: none;
  border: 1px solid var(--graph-composer-input-border);
  border-radius: 4px;
  outline: none;
  background: var(--graph-composer-input-bg);
  padding: 10px 12px;
  color: var(--graph-composer-input-color);
  font: inherit;
  font-size: 27px;
  line-height: 1.55;
  caret-color: var(--graph-composer-caret);
  touch-action: auto;
  user-select: text;
  -webkit-user-select: text;
  transition:
    border-color 160ms ease,
    box-shadow 160ms ease,
    background 160ms ease;
}

.composer-input::placeholder {
  color: var(--graph-composer-placeholder);
}

.composer-input:focus {
  border-color: var(--graph-composer-focus-border);
  background: var(--graph-composer-focus-bg);
  box-shadow: var(--graph-composer-focus-shadow);
}

.composer-input:disabled {
  cursor: wait;
  opacity: 0.7;
}

.composer-images {
  display: flex;
  flex: none;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 8px;
}

.image-slot {
  position: relative;
  width: 56px;
  height: 56px;
  overflow: hidden;
  border: 1px solid var(--graph-slot-border);
  border-radius: 4px;
  background: var(--graph-slot-bg);
}

.image-slot img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.image-overlay {
  position: absolute;
  inset: 0;
  display: grid;
  place-content: center;
  border: 0;
  background: var(--graph-slot-overlay-bg);
  color: var(--graph-slot-overlay-color);
  font-size: 14px;
  text-align: center;
}

.image-retry {
  cursor: pointer;
}

.image-retry:hover {
  color: #f0c46c;
}

.image-remove {
  position: absolute;
  top: 2px;
  right: 2px;
  display: grid;
  width: 22px;
  height: 22px;
  place-items: center;
  border: 1px solid var(--graph-slot-remove-border);
  border-radius: 50%;
  background: var(--graph-slot-remove-bg);
  color: var(--graph-slot-remove-color);
  cursor: pointer;
}

.composer-actions {
  display: flex;
  flex: none;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  margin-top: 10px;
  padding-top: 10px;
  border-top: 1px solid var(--graph-panel-divider);
}

.attach-button,
.send-button {
  display: inline-flex;
  min-height: 42px;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 0 16px;
  border-radius: 4px;
  font-size: 18px;
  cursor: pointer;
  transition:
    border-color 150ms ease,
    background 150ms ease,
    color 150ms ease,
    opacity 150ms ease;
}

.attach-button {
  border: 1px solid var(--graph-attach-border);
  background: var(--graph-attach-bg);
  color: var(--graph-attach-color);
}

.send-button {
  border: 1px solid var(--graph-send-border);
  background: var(--graph-send-bg);
  color: var(--graph-send-color);
  font-weight: 650;
}

.attach-button:hover:not(:disabled),
.send-button:hover:not(:disabled) {
  border-color: rgba(247, 201, 103, 0.92);
  color: var(--graph-link-hover-color);
}

.attach-button:disabled,
.send-button:disabled {
  cursor: not-allowed;
  opacity: 0.48;
}

.composer-error {
  flex: none;
  margin: 6px 0 0;
  color: var(--graph-composer-error);
  font-size: 16px;
  line-height: 1.3;
}

.drop-hint {
  position: absolute;
  z-index: 4;
  inset: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
  border: 1px dashed var(--graph-drop-border);
  border-radius: 4px;
  background: var(--graph-drop-bg);
  color: var(--graph-drop-color);
  font-size: 20px;
  font-weight: 600;
  pointer-events: none;
}

.file-input {
  display: none;
}

.spin {
  animation: composer-spin 0.9s linear infinite;
}

@keyframes composer-spin {
  to {
    transform: rotate(360deg);
  }
}

@media (prefers-reduced-motion: reduce) {
  .graph-comment-composer {
    backdrop-filter: none;
  }

  .spin {
    animation: none;
  }
}
</style>
