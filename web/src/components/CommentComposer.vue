<script setup lang="ts">
import type { Comment } from "@/api";

const emit = defineEmits<{ created: [comment: Comment] }>();
const props = defineProps<{ nodeId: number }>();
const fileInput = ref<HTMLInputElement | null>(null);
const hintId = useId();
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
  dispose,
} = useCommentComposer(() => props.nodeId, {
  emptyMessage: "请输入进展或添加图片",
  submitError: "发布进展失败，请重试",
});

async function publish() {
  const comment = await submit();
  if (comment) {
    emit("created", comment);
  }
}

function shortcut(event: KeyboardEvent) {
  if ((event.ctrlKey || event.metaKey) && event.key === "Enter") {
    event.preventDefault();
    void publish();
  }
}

onBeforeUnmount(dispose);
</script>

<template>
  <div
    class="composer-shell"
    @dragenter="dragEnter"
    @dragleave="dragLeave"
    @dragover="dragOver"
    @drop="dropFiles"
  >
    <NCard size="small">
      <NForm @submit.prevent="publish">
        <NFormItem label="发布进展" :show-feedback="false">
          <NInput
            v-model:value="body"
            :autosize="{ minRows: 2, maxRows: 10 }"
            :disabled="submitting"
            :input-props="{
              'aria-label': '发布进展',
              'aria-describedby': hintId,
              'onPaste': pasteImages,
            }"
            :maxlength="5000"
            placeholder="写下进展或问题…"
            type="textarea"
            @keydown="shortcut"
          />
        </NFormItem>
        <ImageGallery>
          <div v-if="images.length > 0" class="image-previews">
            <ImageUploadPreview
              v-for="image in images"
              :key="image.src"
              :disabled="submitting"
              :image
              @remove="remove(image)"
              @retry="retry(image)"
            />
          </div>
        </ImageGallery>
        <div class="composer-actions">
          <NButton
            :disabled="submitting || images.length >= COMMENT_IMAGE_LIMIT"
            size="small"
            @click="fileInput?.click()"
          >
            <template #icon><ImagePlus :size="16" /></template>
            添加图片 {{ images.length }}/{{ COMMENT_IMAGE_LIMIT }}
          </NButton>
          <NButton
            attr-type="submit"
            :disabled="pending"
            :loading="submitting"
            size="small"
            type="primary"
          >
            发布进展
          </NButton>
        </div>
        <input
          ref="fileInput"
          :accept="COMMENT_IMAGE_ACCEPT"
          class="file-input"
          multiple
          type="file"
          @change="selectFiles"
        />
        <NText :id="hintId" class="composer-hint" depth="3">
          可粘贴、拖拽或选择图片 · 最多 {{ COMMENT_IMAGE_LIMIT }} 张，每张不超过
          5 MB · Ctrl / ⌘ + Enter 发布
        </NText>
        <NAlert v-if="error" class="composer-error" role="alert" type="error">
          {{ error }}
        </NAlert>
      </NForm>
    </NCard>
    <div v-if="dragging" aria-hidden="true" class="drop-hint">
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
