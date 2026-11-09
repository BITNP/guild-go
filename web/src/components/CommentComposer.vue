<script setup lang="ts">
import { onBeforeUnmount, ref, useId } from 'vue'
import {
  NAlert,
  NButton,
  NCard,
  NForm,
  NFormItem,
  NInput,
  NText,
} from 'naive-ui'
import { ImagePlus } from '@lucide/vue'
import ImageUploadPreview from '@/components/ImageUploadPreview.vue'
import ImageGallery from '@/components/ImageGallery.vue'
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

const props = defineProps<{ nodeId: number }>()
const emit = defineEmits<{ created: [comment: Comment] }>()
const body = ref('')
const submitting = ref(false)
const error = ref<string | null>(null)
const fileInput = ref<HTMLInputElement | null>(null)
const dragging = ref(false)
const hintId = useId()
const { images, pending, addFiles, retry, remove, clear, dispose } =
  useCommentImageDrafts(() => props.nodeId)
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
  dragDepth += 1
  dragging.value = true
}

function dragOver(event: DragEvent) {
  if (submitting.value || !hasFileDrag(event.dataTransfer)) return
  event.preventDefault()
  if (event.dataTransfer) event.dataTransfer.dropEffect = 'copy'
  dragging.value = true
}

function dragLeave(event: DragEvent) {
  if (!dragging.value) return
  event.preventDefault()
  dragDepth = Math.max(0, dragDepth - 1)
  if (dragDepth === 0) dragging.value = false
}

function dropFiles(event: DragEvent) {
  const files = filesFromDataTransfer(event.dataTransfer)
  if (!files.length) return
  event.preventDefault()
  dragDepth = 0
  dragging.value = false
  if (!submitting.value) acceptFiles(files)
}

async function submit() {
  if (submitting.value || pending.value) return
  error.value = null
  if (!body.value.trim() && !images.value.length) {
    error.value = '请输入进展或添加图片'
    return
  }
  submitting.value = true
  try {
    const ids = images.value.flatMap((image) =>
      image.status.state === 'uploaded' ? [image.status.id] : [],
    )
    const comment = await createComment(props.nodeId, body.value, ids)
    body.value = ''
    clear(false)
    emit('created', comment)
  } catch (err) {
    error.value = errorMessage(err)
  } finally {
    submitting.value = false
  }
}

function shortcut(event: KeyboardEvent) {
  if ((event.ctrlKey || event.metaKey) && event.key === 'Enter') {
    event.preventDefault()
    void submit()
  }
}

onBeforeUnmount(() => dispose(!submitting.value))
</script>
<template>
  <div
    class="composer-shell"
    @dragenter="dragEnter"
    @dragover="dragOver"
    @dragleave="dragLeave"
    @drop="dropFiles"
  >
    <NCard size="small">
      <NForm @submit.prevent="submit">
        <NFormItem label="发布进展" :show-feedback="false">
          <NInput
            v-model:value="body"
            type="textarea"
            :disabled="submitting"
            :maxlength="5000"
            :autosize="{ minRows: 2, maxRows: 10 }"
            :input-props="{
              'aria-label': '发布进展',
              'aria-describedby': hintId,
              onPaste: pasteImages,
            }"
            placeholder="写下进展或问题…"
            @keydown="shortcut"
          />
        </NFormItem>
        <ImageGallery>
          <div v-if="images.length" class="image-previews">
            <ImageUploadPreview
              v-for="image in images"
              :key="image.src"
              :src="image.src"
              :name="image.file.name"
              :uploading="image.status.state === 'uploading'"
              :error="
                image.status.state === 'failed' ? image.status.message : null
              "
              :disabled="submitting"
              @retry="retry(image)"
              @remove="remove(image)"
            />
          </div>
        </ImageGallery>
        <div class="composer-actions">
          <NButton
            size="small"
            :disabled="submitting || images.length >= COMMENT_IMAGE_LIMIT"
            @click="fileInput?.click()"
          >
            <template #icon><ImagePlus :size="16" /></template>
            添加图片 {{ images.length }}/{{ COMMENT_IMAGE_LIMIT }}
          </NButton>
          <NButton
            type="primary"
            size="small"
            attr-type="submit"
            :loading="submitting"
            :disabled="pending"
          >
            发布进展
          </NButton>
        </div>
        <input
          ref="fileInput"
          type="file"
          :accept="COMMENT_IMAGE_ACCEPT"
          multiple
          class="file-input"
          @change="selectFiles"
        />
        <NText :id="hintId" depth="3" class="composer-hint">
          可粘贴、拖拽或选择图片 · 最多 {{ COMMENT_IMAGE_LIMIT }} 张，每张不超过
          5 MB · Ctrl / ⌘ + Enter 发布
        </NText>
        <NAlert
          v-if="error"
          type="error"
          role="alert"
          class="composer-error"
        >
          {{ error }}
        </NAlert>
      </NForm>
    </NCard>
    <div v-if="dragging" class="drop-hint" aria-hidden="true">
      <ImagePlus :size="22" />
      <span>松开以添加图片</span>
    </div>
  </div>
</template>
<style scoped>
.composer-shell {
  position: relative;
}
.drop-hint {
  position: absolute;
  z-index: 2;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  border: 1px dashed var(--guild-primary);
  border-radius: 8px;
  background: var(--guild-surface);
  color: var(--guild-primary);
  font-size: 14px;
  font-weight: 600;
  pointer-events: none;
}
.image-previews {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  padding: 16px 6px 4px;
}
.composer-actions {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 16px;
}
.file-input {
  display: none;
}
.composer-hint {
  display: block;
  margin-top: 12px;
  font-size: 12px;
  line-height: 1.7;
}
.composer-error {
  margin-top: 12px;
}
</style>
