<script setup lang="ts">
import { computed, onBeforeUnmount, ref, useId } from 'vue'
import {
  ImagePlus,
  Loader2,
  RotateCcw,
  Send,
  X,
} from '@lucide/vue'
import {
  COMMENT_IMAGE_ACCEPT,
  COMMENT_IMAGE_LIMIT,
  filesFromClipboard,
  filesFromDataTransfer,
  hasFileDrag,
  useCommentImageDrafts,
} from '@/composables/useCommentImageDrafts'
import { createComment, type Comment } from '@/lib/api'
import { errorMessage } from '@/lib/utils'

const props = defineProps<{
  nodeId: number
}>()

const emit = defineEmits<{ created: [comment: Comment] }>()
const body = ref('')
const submitting = ref(false)
const error = ref<string | null>(null)
const fileInput = ref<HTMLInputElement | null>(null)
const textarea = ref<HTMLTextAreaElement | null>(null)
const dragging = ref(false)
const bodyId = useId()
const { images, pending, addFiles, retry, remove, clear, dispose } =
  useCommentImageDrafts(() => props.nodeId)
const canSubmit = computed(() =>
  !submitting.value &&
  !pending.value &&
  (body.value.trim().length > 0 || images.value.length > 0),
)
let dragDepth = 0

function acceptFiles(files: File[]) {
  error.value = null
  error.value = addFiles(files)
}

function selectFiles(event: Event) {
  const input = event.target as HTMLInputElement
  const selected = [...(input.files ?? [])]
  input.value = ''
  acceptFiles(selected)
}

function pasteImages(event: ClipboardEvent) {
  const files = filesFromClipboard(event.clipboardData)
  if (!files.length) return
  event.preventDefault()
  acceptFiles(files)
}

function dragEnter(event: DragEvent) {
  if (submitting.value || !hasFileDrag(event.dataTransfer)) return
  event.preventDefault()
  event.stopPropagation()
  dragDepth += 1
  dragging.value = true
}

function dragOver(event: DragEvent) {
  if (submitting.value || !hasFileDrag(event.dataTransfer)) return
  event.preventDefault()
  event.stopPropagation()
  if (event.dataTransfer) event.dataTransfer.dropEffect = 'copy'
  dragging.value = true
}

function dragLeave(event: DragEvent) {
  if (!dragging.value) return
  event.preventDefault()
  event.stopPropagation()
  dragDepth = Math.max(0, dragDepth - 1)
  if (dragDepth === 0) dragging.value = false
}

function dropFiles(event: DragEvent) {
  const files = filesFromDataTransfer(event.dataTransfer)
  if (!files.length) return
  event.preventDefault()
  event.stopPropagation()
  dragDepth = 0
  dragging.value = false
  if (!submitting.value) acceptFiles(files)
}

async function submit() {
  if (!canSubmit.value) {
    if (!body.value.trim() && !images.value.length) {
      error.value = '写下一步进展或添加一张图片'
    }
    return
  }
  error.value = null
  submitting.value = true
  try {
    const attachmentIds = images.value.flatMap((image) =>
      image.status.state === 'uploaded' ? [image.status.id] : [],
    )
    const comment = await createComment(props.nodeId, body.value, attachmentIds)
    body.value = ''
    clear(false)
    emit('created', comment)
  } catch (err) {
    error.value = errorMessage(err, '进展发送失败，请重试')
  } finally {
    submitting.value = false
  }
}

function shortcut(event: KeyboardEvent) {
  if (
    event.key !== 'Enter' ||
    event.shiftKey ||
    event.isComposing ||
    event.keyCode === 229
  ) {
    return
  }
  event.preventDefault()
  void submit()
}

function focusComposer(event: PointerEvent) {
  event.stopPropagation()
  const target = event.target
  if (target instanceof HTMLTextAreaElement) return
  if (
    target instanceof Element &&
    target.closest('button, input')
  ) {
    return
  }
  event.preventDefault()
  textarea.value?.focus({ preventScroll: true })
}

onBeforeUnmount(() => dispose(!submitting.value))
</script>

<template>
  <section
    class="graph-comment-composer"
    data-graph-comment-composer
    aria-label="输入进展"
    @pointerdown.capture="focusComposer"
    @pointermove.stop
    @pointerup.stop
    @click.stop
    @dblclick.stop
    @dragenter="dragEnter"
    @dragover="dragOver"
    @dragleave="dragLeave"
    @drop="dropFiles"
  >
    <textarea
      ref="textarea"
      :id="bodyId"
      v-model="body"
      class="composer-input"
      :class="{ 'has-images': images.length > 0 }"
      rows="2"
      maxlength="5000"
      :disabled="submitting"
      :aria-invalid="!!error"
      aria-label="请输入进展"
      placeholder="输入进展，可粘贴或拖拽图片"
      @keydown="shortcut"
      @paste="pasteImages"
    />

    <div v-if="images.length" class="composer-images" aria-live="polite">
      <div v-for="image in images" :key="image.src" class="image-slot">
        <img :src="image.src" :alt="image.file.name" />
        <div
          v-if="image.status.state === 'uploading'"
          class="image-overlay"
          role="progressbar"
          :aria-label="`正在上传：${image.file.name}`"
        >
          <Loader2 :size="18" class="spin" />
        </div>
        <button
          v-else-if="image.status.state === 'failed'"
          type="button"
          class="image-overlay image-retry"
          :aria-label="`重试上传：${image.file.name}`"
          :title="image.status.message"
          @click="retry(image)"
        >
          <RotateCcw :size="16" />
        </button>
        <button
          v-if="image.status.state !== 'uploading'"
          type="button"
          class="image-remove"
          :aria-label="`移除图片：${image.file.name}`"
          :disabled="submitting"
          @click="remove(image)"
        >
          <X :size="12" />
        </button>
      </div>
    </div>

    <div class="composer-actions">
      <button
        type="button"
        class="attach-button"
        :disabled="submitting || images.length >= COMMENT_IMAGE_LIMIT"
        aria-label="添加图片，也可粘贴或拖拽图片"
        title="添加图片，也可粘贴或拖拽"
        @click="fileInput?.click()"
      >
        <ImagePlus :size="15" />
      </button>
      <button
        type="button"
        class="send-button"
        :disabled="submitting || pending || (!body.trim() && !images.length)"
        :aria-label="submitting ? '正在发送进展' : pending ? '正在上传图片' : '发送进展'"
        :title="submitting ? '正在发送进展' : pending ? '正在上传图片' : '发送进展'"
        @click="submit"
      >
        <Loader2 v-if="submitting" :size="15" class="spin" />
        <Send v-else :size="15" />
      </button>
    </div>

    <input
      ref="fileInput"
      type="file"
      :accept="COMMENT_IMAGE_ACCEPT"
      multiple
      class="file-input"
      @change="selectFiles"
    />
    <div v-if="dragging" class="drop-hint" aria-hidden="true">
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
  --composer-accent: #e8b954;
  position: absolute;
  z-index: 18;
  bottom: 82px;
  left: 30px;
  width: clamp(300px, calc(100vw - 430px), 440px);
  padding: 17px 18px 14px 22px;
  border: 1px solid rgba(226, 178, 76, 0.34);
  border-left-color: rgba(240, 188, 79, 0.82);
  border-radius: 3px;
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
  transform:
    perspective(1100px)
    rotateX(1.1deg)
    rotateY(1.4deg);
  transform-origin: 0 100%;
}

.graph-comment-composer::before,
.graph-comment-composer::after {
  position: absolute;
  width: 28px;
  height: 28px;
  border-color: rgba(239, 188, 80, 0.7);
  border-style: solid;
  content: '';
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
  min-height: 82px;
  max-height: 190px;
  resize: vertical;
  border: 1px solid rgba(108, 148, 155, 0.34);
  border-radius: 2px;
  outline: none;
  background:
    linear-gradient(90deg, rgba(80, 183, 181, 0.05), transparent 34%),
    rgba(2, 9, 12, 0.58);
  padding: 11px 12px;
  color: #eef2ec;
  font: inherit;
  font-size: 13px;
  line-height: 1.65;
  caret-color: #f0bd58;
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
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 9px;
}

.image-slot {
  position: relative;
  width: 62px;
  height: 62px;
  overflow: hidden;
  border: 1px solid rgba(108, 148, 155, 0.34);
  border-radius: 2px;
  background: #0b1418;
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
  gap: 3px;
  border: 0;
  background: rgba(3, 10, 13, 0.76);
  color: #d7e4e2;
  font-size: 12px;
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
  width: 19px;
  height: 19px;
  place-items: center;
  border: 1px solid rgba(218, 185, 119, 0.3);
  border-radius: 50%;
  background: rgba(4, 10, 12, 0.84);
  color: #cad7d5;
  cursor: pointer;
}

.composer-actions {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  margin-top: 11px;
}

.attach-button,
.send-button {
  display: inline-flex;
  min-height: 32px;
  align-items: center;
  justify-content: center;
  gap: 6px;
  border-radius: 2px;
  padding: 0 11px;
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

.attach-button span {
  color: rgba(225, 181, 90, 0.72);
  font-variant-numeric: tabular-nums;
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
  margin: 8px 0 0;
  color: #f2a18c;
  font-size: 13px;
  line-height: 1.5;
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

@media (max-width: 720px) {
  .graph-comment-composer {
    bottom: 74px;
    left: 14px;
    width: calc(100vw - 374px);
    min-width: 264px;
    padding: 14px 14px 12px 18px;
  }
}

@media (max-width: 560px) {
  .graph-comment-composer {
    right: 14px;
    bottom: 70px;
    width: auto;
    min-width: 0;
  }
}

/*
 * CSS3DRenderer owns this element's position and transform. The fixed logical
 * size keeps it aligned with the 6.2 x 2.36 comment-card plane in the scene.
 */
.graph-comment-composer {
  top: auto;
  right: auto;
  bottom: auto;
  left: auto;
  display: flex;
  width: 620px;
  height: 260px;
  min-width: 0;
  margin: 0;
  padding: 18px 22px 14px 26px;
  flex-direction: column;
  overflow: hidden;
  box-sizing: border-box;
  border-radius: 0;
  transform-origin: 50% 50%;
  visibility: hidden;
  opacity: 0;
}

.composer-input {
  min-height: 0;
  height: auto;
  max-height: none;
  flex: 1 1 auto;
  margin-top: 0;
  padding: 10px 12px;
  resize: none;
  font-size: 27px;
  line-height: 1.55;
  touch-action: auto;
  user-select: text;
  -webkit-user-select: text;
}

.composer-input.has-images {
  height: auto;
  max-height: none;
}

.composer-images {
  flex: none;
  gap: 5px;
  margin-top: 6px;
}

.image-slot {
  width: 34px;
  height: 34px;
}

.image-overlay {
  gap: 0;
  font-size: 12px;
}

.image-overlay svg {
  width: 13px;
  height: 13px;
}

.image-remove {
  width: 16px;
  height: 16px;
}

.composer-actions {
  flex: none;
  justify-content: flex-end;
  margin-top: 8px;
  padding-top: 8px;
  border-top: 1px solid rgba(139, 166, 171, 0.17);
}

.attach-button,
.send-button {
  width: 38px;
  min-height: 34px;
  padding: 0;
}

.composer-error {
  flex: none;
  margin-top: 4px;
  font-size: 16px;
  line-height: 1.3;
}

:global([data-theme='light'] .graph-comment-composer) {
  border-color: rgba(160, 105, 20, 0.28);
  border-left-color: rgba(147, 90, 8, 0.62);
  background:
    linear-gradient(135deg, rgba(193, 128, 29, 0.08), transparent 34%),
    linear-gradient(180deg, rgba(255, 255, 255, 0.97), rgba(246, 249, 250, 0.98));
  box-shadow:
    0 26px 70px rgba(48, 63, 72, 0.2),
    0 0 32px rgba(180, 118, 20, 0.05),
    inset 0 0 38px rgba(55, 137, 135, 0.02);
  color: #2a3942;
}

:global([data-theme='light'] .composer-input) {
  border-color: rgba(79, 108, 120, 0.28);
  background:
    linear-gradient(90deg, rgba(58, 143, 142, 0.04), transparent 34%),
    rgba(255, 255, 255, 0.84);
  color: #17242c;
  caret-color: #9a620e;
}

:global([data-theme='light'] .composer-input::placeholder) {
  color: rgba(73, 94, 106, 0.58);
}

:global([data-theme='light'] .composer-input:focus) {
  border-color: rgba(158, 101, 16, 0.58);
  background: rgba(255, 255, 255, 0.98);
  box-shadow:
    0 0 0 2px rgba(178, 116, 22, 0.07),
    inset 0 0 24px rgba(58, 143, 142, 0.03);
}

:global([data-theme='light'] .image-slot) {
  border-color: rgba(80, 108, 120, 0.26);
  background: #eef3f5;
}

:global([data-theme='light'] .image-overlay) {
  background: rgba(238, 244, 246, 0.86);
  color: #455b66;
}

:global([data-theme='light'] .image-remove) {
  border-color: rgba(121, 91, 47, 0.28);
  background: rgba(255, 255, 255, 0.9);
  color: #506570;
}

:global([data-theme='light'] .composer-actions) {
  border-top-color: rgba(80, 108, 120, 0.18);
}

:global([data-theme='light'] .attach-button) {
  border-color: rgba(78, 109, 121, 0.28);
  background: rgba(255, 255, 255, 0.72);
  color: #536a75;
}

:global([data-theme='light'] .send-button) {
  border-color: rgba(157, 101, 18, 0.48);
  background:
    linear-gradient(135deg, rgba(189, 124, 27, 0.14), transparent),
    rgba(255, 249, 236, 0.88);
  color: #7f4f0a;
  box-shadow: 0 0 18px rgba(176, 112, 18, 0.05);
}

:global([data-theme='light'] .composer-error) {
  color: #a33f36;
}

:global([data-theme='light'] .drop-hint) {
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
