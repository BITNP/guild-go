<script setup lang="ts">
import { ref, useId } from 'vue'
import {
  NAlert,
  NButton,
  NCard,
  NInput,
  NText,
} from 'naive-ui'
import { ImagePlus } from '@lucide/vue'
import ImageUploadPreview from '@/components/ImageUploadPreview.vue'
import ImageGallery from '@/components/ImageGallery.vue'
import { useCommentComposer } from '@/composables/useCommentComposer'
import {
  COMMENT_IMAGE_ACCEPT,
  COMMENT_IMAGE_LIMIT,
} from '@/composables/useCommentImageDrafts'
import type { Comment } from '@/lib/api'

const props = defineProps<{ nodeId: number }>()
const emit = defineEmits<{ created: [comment: Comment] }>()
const fileInput = ref<HTMLInputElement | null>(null)
const hintId = useId()
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
  emptyMessage: '请输入进展或添加图片',
  onCreated: (comment) => emit('created', comment),
})

function shortcut(event: KeyboardEvent) {
  if ((event.ctrlKey || event.metaKey) && event.key === 'Enter') {
    event.preventDefault()
    void submit()
  }
}
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
      <form @submit.prevent="submit">
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
          <NText :id="hintId" depth="3" class="composer-hint">
            可粘贴或拖拽图片，最多 {{ COMMENT_IMAGE_LIMIT }} 张 · Ctrl / ⌘ + Enter 发布
          </NText>
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
        <NAlert
          v-if="error"
          type="error"
          role="alert"
          class="composer-error"
        >
          {{ error }}
        </NAlert>
      </form>
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
  gap: 8px 12px;
  margin-top: 12px;
}
.file-input {
  display: none;
}
.composer-hint {
  margin-right: auto;
  font-size: 12px;
  line-height: 1.5;
}
.composer-error {
  margin-top: 12px;
}
</style>
