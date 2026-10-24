<script setup lang="ts">
import { computed, onBeforeUnmount, reactive, ref, useId } from 'vue'
import { NAlert, NButton, NForm, NFormItem, NInput, NText } from 'naive-ui'
import { ImagePlus } from '@lucide/vue'
import ImageUploadPreview from '@/components/ImageUploadPreview.vue'
import ImageGallery from '@/components/ImageGallery.vue'
import {
  createComment,
  deleteCommentUpload,
  uploadCommentImage,
  type Comment,
} from '@/lib/api'
import { errorMessage } from '@/lib/utils'
const props = defineProps<{ nodeId: number }>(),
  emit = defineEmits<{ created: [comment: Comment] }>()
type DraftImage = {
  file: File
  src: string
  status:
    | { state: 'uploading' }
    | { state: 'uploaded'; id: number }
    | { state: 'failed'; message: string }
}
const body = ref(''),
  images = ref<DraftImage[]>([]),
  submitting = ref(false),
  error = ref<string | null>(null),
  fileInput = ref<HTMLInputElement | null>(null)
const hintId = useId()
const pending = computed(() =>
  images.value.some((image) => image.status.state !== 'uploaded'),
)
const types = new Set(['image/jpeg', 'image/png', 'image/webp', 'image/gif'])
let active = true
function discard(id: number) {
  void deleteCommentUpload(id).catch(() => {
    /* Expiry cleanup removes uploads left by interrupted requests. */
  })
}
async function upload(image: DraftImage) {
  image.status = { state: 'uploading' }
  try {
    const { id } = await uploadCommentImage(props.nodeId, image.file)
    if (!images.value.includes(image)) {
      discard(id)
      return
    }
    image.status = { state: 'uploaded', id }
  } catch (err) {
    image.status = { state: 'failed', message: errorMessage(err) }
  }
}
function selectFiles(event: Event) {
  const input = event.target as HTMLInputElement
  const selected = [...(input.files ?? [])]
  input.value = ''
  error.value = null
  for (const file of selected) {
    if (images.value.length >= 4) {
      error.value = '每条评论最多 4 张图片'
      break
    }
    if (!types.has(file.type)) {
      error.value = '只支持 JPEG、PNG、WebP、GIF 图片'
      continue
    }
    if (file.size > 5 * 1024 * 1024) {
      error.value = '单张图片不能超过 5 MB'
      continue
    }
    const image = reactive<DraftImage>({
      file,
      src: URL.createObjectURL(file),
      status: { state: 'uploading' },
    })
    images.value.push(image)
    void upload(image)
  }
}
function retry(image: DraftImage) {
  if (!submitting.value && image.status.state === 'failed') void upload(image)
}
function remove(image: DraftImage) {
  if (submitting.value || image.status.state === 'uploading') return
  images.value = images.value.filter((item) => item !== image)
  URL.revokeObjectURL(image.src)
  if (image.status.state === 'uploaded') discard(image.status.id)
}
function clearImages(abandon: boolean) {
  for (const image of images.value) {
    URL.revokeObjectURL(image.src)
    if (abandon && image.status.state === 'uploaded') discard(image.status.id)
  }
  images.value = []
}
async function submit() {
  if (submitting.value || pending.value) return
  error.value = null
  if (!body.value.trim() && !images.value.length) {
    error.value = '请输入评论或添加图片'
    return
  }
  submitting.value = true
  try {
    const ids = images.value.flatMap((image) =>
      image.status.state === 'uploaded' ? [image.status.id] : [],
    )
    const comment = await createComment(props.nodeId, body.value, ids)
    if (!active) return
    body.value = ''
    clearImages(false)
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
onBeforeUnmount(() => {
  active = false
  clearImages(!submitting.value)
})
</script>
<template>
  <NForm class="comment-composer" @submit.prevent="submit"
    ><NFormItem label="发表评论" :show-feedback="false"
      ><NInput
        v-model:value="body"
        type="textarea"
        :disabled="submitting"
        :maxlength="5000"
        :autosize="{ minRows: 2, maxRows: 10 }"
        :input-props="{
          'aria-label': '发表评论',
          'aria-describedby': hintId,
        }"
        placeholder="写下进展或问题…"
        @keydown="shortcut" /></NFormItem
    ><ImageGallery
      ><div v-if="images.length" class="image-previews">
        <ImageUploadPreview
          v-for="image in images"
          :key="image.src"
          :src="image.src"
          :name="image.file.name"
          :uploading="image.status.state === 'uploading'"
          :error="image.status.state === 'failed' ? image.status.message : null"
          :disabled="submitting"
          @retry="retry(image)"
          @remove="remove(image)"
        /></div
    ></ImageGallery>
    <div class="composer-actions">
      <NButton
        size="small"
        :disabled="submitting || images.length >= 4"
        @click="fileInput?.click()"
        ><template #icon><ImagePlus :size="16" /></template>添加图片
        {{ images.length }}/4</NButton
      ><NButton
        type="primary"
        size="small"
        attr-type="submit"
        :loading="submitting"
        :disabled="pending"
        >发表评论</NButton
      >
    </div>
    <input
      ref="fileInput"
      type="file"
      accept="image/jpeg,image/png,image/webp,image/gif"
      multiple
      class="file-input"
      @change="selectFiles"
    /><NText :id="hintId" depth="3" class="composer-hint"
      >最多 4 张图片，每张不超过 5 MB；支持 JPEG、PNG、WebP、GIF。Ctrl / ⌘ +
      Enter 发表</NText
    ><NAlert v-if="error" type="error" role="alert" class="composer-error">{{
      error
    }}</NAlert></NForm
  >
</template>
<style scoped>
.comment-composer {
  padding: 12px;
  margin-bottom: 4px;
  border: 1px solid var(--guild-border);
  border-radius: 8px;
  background: var(--guild-surface-subtle);
}
.comment-composer :deep(.n-form-item-label) {
  font-weight: 600;
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
