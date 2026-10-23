<script setup lang="ts">
import { computed, defineAsyncComponent, onBeforeUnmount, reactive, ref, useId } from 'vue'
import { ImagePlus, Loader2, Send } from '@lucide/vue'
import ImageUploadPreview from '@/components/ImageUploadPreview.vue'
import { createComment, deleteCommentUpload, uploadCommentImage, type Comment } from '@/lib/api'
import { errorMessage } from '@/lib/utils'

const props = defineProps<{
  nodeId: number
}>()

const emit = defineEmits<{
  created: [comment: Comment]
}>()

const inputId = useId()
const ImageViewer = defineAsyncComponent(() => import('@/components/ImageViewer.vue'))
type UploadImage = {
  file: File
  src: string
  status: { state: 'uploading' } | { state: 'uploaded'; id: number } | { state: 'failed'; error: string }
}
const body = ref('')
const images = ref<UploadImage[]>([])
const imageIndex = ref<number | null>(null)
const viewerImages = computed(() => images.value.map(image => ({ src: image.src, alt: image.file.name })))
const uploading = computed(() => images.value.some(image => image.status.state === 'uploading'))
const failed = computed(() => images.value.some(image => image.status.state === 'failed'))
const fileInput = ref<HTMLInputElement | null>(null)
const submitting = ref(false)
const error = ref<string | null>(null)
let active = true

const MAX_FILES = 4
const MAX_SIZE = 5 * 1024 * 1024
const ALLOWED = new Set(['image/jpeg', 'image/png', 'image/webp', 'image/gif'])

function openPicker() {
  fileInput.value?.click()
}

function onFiles(event: Event) {
  const input = event.target as HTMLInputElement
  const selected = [...(input.files ?? [])]
  input.value = ''
  error.value = null
  for (const file of selected) {
    if (images.value.length >= MAX_FILES) {
      error.value = '每条评论最多 4 张图片'
      break
    }
    if (!ALLOWED.has(file.type)) {
      error.value = '只支持 JPEG、PNG、WebP、GIF 图片'
      continue
    }
    if (file.size > MAX_SIZE) {
      error.value = '单张图片不能超过 5MB'
      continue
    }
    const image = reactive<UploadImage>({ file, src: URL.createObjectURL(file), status: { state: 'uploading' } })
    images.value.push(image)
    void upload(image)
  }
}

function discardUpload(id: number) {
  // Expiry cleanup handles uploads left behind when this request cannot finish.
  void deleteCommentUpload(id).catch(() => {})
}

async function upload(image: UploadImage) {
  image.status = { state: 'uploading' }
  try {
    const { id } = await uploadCommentImage(props.nodeId, image.file)
    if (!images.value.includes(image)) {
      discardUpload(id)
      return
    }
    image.status = { state: 'uploaded', id }
  } catch (err) {
    image.status = { state: 'failed', error: errorMessage(err) }
  }
}

function retry(image: UploadImage) {
  if (!submitting.value && image.status.state === 'failed') void upload(image)
}

function removeImage(image: UploadImage) {
  if (submitting.value || image.status.state === 'uploading') return
  imageIndex.value = null
  images.value = images.value.filter(item => item !== image)
  URL.revokeObjectURL(image.src)
  if (image.status.state === 'uploaded') discardUpload(image.status.id)
}

function clearImages(discard: boolean) {
  imageIndex.value = null
  for (const image of images.value) {
    URL.revokeObjectURL(image.src)
    if (discard && image.status.state === 'uploaded') discardUpload(image.status.id)
  }
  images.value = []
}

async function submit() {
  if (submitting.value || uploading.value || failed.value) return
  error.value = null
  if (!body.value.trim() && images.value.length === 0) {
    error.value = '请输入评论或添加图片'
    return
  }
  submitting.value = true
  try {
    const attachmentIds = images.value.flatMap(image => image.status.state === 'uploaded' ? [image.status.id] : [])
    const comment = await createComment(props.nodeId, body.value, attachmentIds)
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

onBeforeUnmount(() => {
  active = false
  clearImages(!submitting.value)
})

function onKeydown(event: KeyboardEvent) {
  if ((event.metaKey || event.ctrlKey) && event.key === 'Enter') {
    event.preventDefault()
    void submit()
  }
}
</script>

<template>
  <div class="rounded-lg border border-line-strong bg-surface focus-within:border-accent">
    <label :for="inputId" class="block px-3 pt-3 text-[13px] font-medium">发表评论</label>
    <textarea
      :id="inputId"
      :disabled="submitting"
      :aria-describedby="`${inputId}-hint`"
      v-model="body"
      class="w-full resize-y rounded-t-lg border-0 bg-transparent px-3 py-2.5 text-[13px] leading-6 text-ink outline-none placeholder:text-faint"
      rows="3"
      maxlength="5000"
      placeholder="写下进展或问题…"
      @keydown="onKeydown"
    />

    <div v-if="images.length" class="flex flex-wrap gap-2 px-3 pb-2">
      <ImageUploadPreview
        v-for="(image, index) in images"
        :key="image.src"
        :src="image.src"
        :name="image.file.name"
        :uploading="image.status.state === 'uploading'"
        :error="image.status.state === 'failed' ? image.status.error : null"
        :disabled="submitting"
        @preview="imageIndex = index"
        @remove="removeImage(image)"
        @retry="retry(image)"
      />
    </div>
    <ImageViewer v-if="imageIndex !== null" :images="viewerImages" :index="imageIndex" @close="imageIndex = null" />

    <div class="flex flex-wrap items-center gap-2 border-t border-line px-3 py-2">
      <button type="button" class="btn btn-ghost btn-xs" :disabled="submitting || images.length >= MAX_FILES" @click="openPicker">
        <ImagePlus class="h-3.5 w-3.5" />
        添加图片
        <span class="text-faint">{{ images.length }}/{{ MAX_FILES }}</span>
      </button>
      <input
        ref="fileInput"
        type="file"
        class="hidden"
        accept="image/jpeg,image/png,image/webp,image/gif"
        multiple
        @change="onFiles"
      >
      <button type="button" class="btn btn-primary btn-xs ml-auto" :disabled="submitting || uploading || failed" @click="submit">
        <Loader2 v-if="submitting" class="h-3.5 w-3.5 animate-spin" />
        <Send v-else class="h-3.5 w-3.5" />
        {{ submitting ? '发表中…' : '发表评论' }}
      </button>
    </div>

    <p :id="`${inputId}-hint`" class="border-t border-line px-3 py-2 text-xs leading-5 text-muted">最多 4 张图片，每张不超过 5 MB；支持 JPEG、PNG、WebP、GIF。<span class="ml-1 hidden sm:inline">Ctrl / ⌘ + Enter 发表</span></p>
    <p v-if="error" role="alert" class="border-t border-danger/20 bg-danger-soft px-3 py-1.5 text-xs text-danger">{{ error }}</p>
  </div>
</template>
