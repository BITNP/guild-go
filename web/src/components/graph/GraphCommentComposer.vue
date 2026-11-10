<script setup lang="ts">
import type { Comment } from "@/api";

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
  canSubmit,
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
  dispose,
} = useCommentComposer(() => props.nodeId, {
  emptyMessage: "写下一步进展或添加一张图片",
  submitError: "进展发送失败，请重试",
});

async function publish() {
  const comment = await submit();
  if (comment) {
    emit("created", comment);
  }
}

function shortcut(event: KeyboardEvent) {
  if (
    event.key !== "Enter" ||
    event.shiftKey ||
    event.isComposing ||
    event.keyCode === 229
  ) {
    return;
  }
  event.preventDefault();
  void publish();
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

onBeforeUnmount(dispose);
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
      :disabled="submitting"
      maxlength="5000"
      placeholder="输入进展，可粘贴或拖拽图片"
      rows="2"
      @keydown="shortcut"
      @paste="pasteImages"
    />

    <div v-if="images.length > 0" aria-live="polite" class="composer-images">
      <ImageUploadPreview
        v-for="image in images"
        :key="image.src"
        compact
        :disabled="submitting"
        :image
        @remove="remove(image)"
        @retry="retry(image)"
      />
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
        <ImagePlus :size="15" />
      </button>
      <button
        :aria-label="
          submitting ? '正在发送进展' : pending ? '正在上传图片' : '发送进展'
        "
        class="send-button"
        :disabled="!canSubmit"
        :title="
          submitting ? '正在发送进展' : pending ? '正在上传图片' : '发送进展'
        "
        type="button"
        @click="publish"
      >
        <Loader2 v-if="submitting" class="spin" :size="15" />
        <Send v-else :size="15" />
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
.graph-comment-composer {
  position: absolute;
  z-index: 18;
  display: flex;
  width: 620px;
  height: 260px;
  margin: 0;
  padding: 18px 22px 14px 26px;
  flex-direction: column;
  overflow: hidden;
  box-sizing: border-box;
  border: 1px solid rgba(226, 178, 76, 0.34);
  border-left-color: rgba(240, 188, 79, 0.82);
  background:
    linear-gradient(135deg, rgba(232, 181, 75, 0.1), transparent 34%),
    linear-gradient(180deg, rgba(9, 17, 20, 0.94), rgba(5, 11, 14, 0.97));
  box-shadow:
    0 26px 70px rgba(0, 0, 0, 0.48),
    0 0 32px rgba(227, 173, 60, 0.06),
    inset 0 0 38px rgba(81, 178, 176, 0.025);
  color: #dce5e4;
  pointer-events: auto;
  backdrop-filter: blur(22px) saturate(120%);
  transform-origin: 50% 50%;
  visibility: hidden;
  opacity: 0;
}

.graph-comment-composer::before,
.graph-comment-composer::after {
  position: absolute;
  width: 28px;
  height: 28px;
  border-color: rgba(239, 188, 80, 0.7);
  border-style: solid;
  content: "";
  pointer-events: none;
}

.graph-comment-composer::before {
  top: -1px;
  left: -1px;
  border-width: 1px 0 0 1px;
}

.graph-comment-composer::after {
  right: -1px;
  bottom: -1px;
  border-width: 0 1px 1px 0;
}

.composer-input {
  display: block;
  width: 100%;
  min-height: 0;
  flex: 1 1 auto;
  resize: none;
  border: 1px solid rgba(108, 148, 155, 0.34);
  border-radius: 2px;
  outline: none;
  background:
    linear-gradient(90deg, rgba(80, 183, 181, 0.05), transparent 34%),
    rgba(2, 9, 12, 0.58);
  padding: 10px 12px;
  color: #eef2ec;
  font: inherit;
  font-size: 27px;
  line-height: 1.55;
  caret-color: #f0bd58;
  touch-action: auto;
  user-select: text;
  -webkit-user-select: text;
  transition:
    border-color 160ms ease,
    box-shadow 160ms ease,
    background 160ms ease;
}

.composer-input::placeholder {
  color: rgba(150, 174, 178, 0.58);
}

.composer-input:focus {
  border-color: rgba(239, 188, 80, 0.72);
  background: rgba(6, 14, 17, 0.78);
  box-shadow:
    0 0 0 2px rgba(232, 181, 75, 0.08),
    inset 0 0 24px rgba(78, 178, 175, 0.035);
}

.composer-input:disabled {
  cursor: wait;
  opacity: 0.7;
}

.composer-images {
  display: flex;
  flex: none;
  flex-wrap: wrap;
  gap: 5px;
  margin-top: 6px;
}

.composer-actions {
  display: flex;
  flex: none;
  align-items: center;
  justify-content: flex-end;
  gap: 10px;
  margin-top: 8px;
  padding-top: 8px;
  border-top: 1px solid rgba(139, 166, 171, 0.17);
}

.attach-button,
.send-button {
  display: inline-flex;
  width: 38px;
  min-height: 34px;
  align-items: center;
  justify-content: center;
  gap: 6px;
  border-radius: 2px;
  padding: 0;
  font-size: 13px;
  cursor: pointer;
  transition:
    border-color 150ms ease,
    background 150ms ease,
    color 150ms ease,
    opacity 150ms ease;
}

.attach-button {
  border: 1px solid rgba(113, 148, 154, 0.34);
  background: rgba(10, 19, 22, 0.58);
  color: #adbec0;
}

.send-button {
  border: 1px solid rgba(239, 188, 80, 0.66);
  background:
    linear-gradient(135deg, rgba(239, 188, 80, 0.2), transparent),
    rgba(37, 31, 15, 0.76);
  color: #f6d58f;
  font-weight: 650;
  letter-spacing: 0.04em;
  box-shadow: 0 0 18px rgba(234, 179, 65, 0.08);
}

.attach-button:hover:not(:disabled),
.send-button:hover:not(:disabled) {
  border-color: rgba(247, 201, 103, 0.92);
  color: #fff0c8;
}

.send-button:hover:not(:disabled) {
  background:
    linear-gradient(135deg, rgba(239, 188, 80, 0.28), transparent),
    rgba(47, 38, 17, 0.88);
}

.attach-button:disabled,
.send-button:disabled {
  cursor: not-allowed;
  opacity: 0.48;
}

.composer-error {
  flex: none;
  margin: 4px 0 0;
  color: #f2a18c;
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
  border: 1px dashed rgba(240, 188, 79, 0.82);
  border-radius: 2px;
  background: rgba(4, 11, 14, 0.9);
  color: #f5d28b;
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
  .spin {
    animation: none;
  }
}

[data-theme="light"] .graph-comment-composer {
  border-color: rgba(160, 105, 20, 0.28);
  border-left-color: rgba(147, 90, 8, 0.62);
  background:
    linear-gradient(135deg, rgba(193, 128, 29, 0.08), transparent 34%),
    linear-gradient(
      180deg,
      rgba(255, 255, 255, 0.97),
      rgba(246, 249, 250, 0.98)
    );
  box-shadow:
    0 26px 70px rgba(48, 63, 72, 0.2),
    0 0 32px rgba(180, 118, 20, 0.05),
    inset 0 0 38px rgba(55, 137, 135, 0.02);
  color: #2a3942;
}

[data-theme="light"] .composer-input {
  border-color: rgba(79, 108, 120, 0.28);
  background:
    linear-gradient(90deg, rgba(58, 143, 142, 0.04), transparent 34%),
    rgba(255, 255, 255, 0.84);
  color: #17242c;
  caret-color: #9a620e;
}

[data-theme="light"] .composer-input::placeholder {
  color: rgba(73, 94, 106, 0.58);
}

[data-theme="light"] .composer-input:focus {
  border-color: rgba(158, 101, 16, 0.58);
  background: rgba(255, 255, 255, 0.98);
  box-shadow:
    0 0 0 2px rgba(178, 116, 22, 0.07),
    inset 0 0 24px rgba(58, 143, 142, 0.03);
}

[data-theme="light"] .composer-actions {
  border-top-color: rgba(80, 108, 120, 0.18);
}

[data-theme="light"] .attach-button {
  border-color: rgba(78, 109, 121, 0.28);
  background: rgba(255, 255, 255, 0.72);
  color: #536a75;
}

[data-theme="light"] .send-button {
  border-color: rgba(157, 101, 18, 0.48);
  background:
    linear-gradient(135deg, rgba(189, 124, 27, 0.14), transparent),
    rgba(255, 249, 236, 0.88);
  color: #7f4f0a;
  box-shadow: 0 0 18px rgba(176, 112, 18, 0.05);
}

[data-theme="light"] .composer-error {
  color: #a33f36;
}

[data-theme="light"] .drop-hint {
  border-color: rgba(147, 90, 8, 0.62);
  background: rgba(250, 246, 237, 0.94);
  color: #7f4f0a;
}

@media (prefers-reduced-motion: reduce) {
  .graph-comment-composer {
    backdrop-filter: none;
  }
}
</style>
