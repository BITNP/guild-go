<script setup lang="ts">
import { ref, useId } from 'vue'
import { ImagePlus, Loader2, Send } from '@lucide/vue'
import ImageUploadPreviews from '@/components/ImageUploadPreviews.vue'
import { createComment, type Comment } from '@/lib/api'
import { errorMessage } from '@/lib/utils'

const props = defineProps<{
  nodeId: number
}>()

const emit = defineEmits<{
  created: [comment: Comment]
}>()

const inputId = useId()
const body = ref('')
const files = ref<File[]>([])
const fileInput = ref<HTMLInputElement | null>(null)
const submitting = ref(false)
const error = ref<string | null>(null)
const uploadError = ref<string | null>(null)

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
  uploadError.value = null
  for (const file of selected) {
    if (files.value.length >= MAX_FILES) {
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
    files.value = [...files.value, file]
  }
}

function removeFile(index: number) {
  files.value = files.value.filter((_, i) => i !== index)
  uploadError.value = null
}

async function submit() {
  if (submitting.value) return
  error.value = null
  uploadError.value = null
  if (!body.value.trim() && files.value.length === 0) {
    error.value = '请输入评论或添加图片'
    return
  }
  submitting.value = true
  try {
    const comment = await createComment(props.nodeId, body.value, files.value)
    body.value = ''
    files.value = []
    emit('created', comment)
  } catch (err) {
    if (files.value.length) uploadError.value = errorMessage(err)
    else error.value = errorMessage(err)
  } finally {
    submitting.value = false
  }
}

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

    <ImageUploadPreviews :files="files" :uploading="submitting" :error="uploadError" @remove="removeFile" @retry="submit" />

    <div class="flex flex-wrap items-center gap-2 border-t border-line px-3 py-2">
      <button type="button" class="btn btn-ghost btn-xs" :disabled="submitting || files.length >= MAX_FILES" @click="openPicker">
        <ImagePlus class="h-3.5 w-3.5" />
        添加图片
        <span class="text-faint">{{ files.length }}/{{ MAX_FILES }}</span>
      </button>
      <input
        ref="fileInput"
        type="file"
        class="hidden"
        accept="image/jpeg,image/png,image/webp,image/gif"
        multiple
        @change="onFiles"
      >
      <button type="button" class="btn btn-primary btn-xs ml-auto" :disabled="submitting" @click="submit">
        <Loader2 v-if="submitting" class="h-3.5 w-3.5 animate-spin" />
        <Send v-else class="h-3.5 w-3.5" />
        {{ submitting ? '发表中…' : '发表评论' }}
      </button>
    </div>

    <p :id="`${inputId}-hint`" class="border-t border-line px-3 py-2 text-xs leading-5 text-muted">最多 4 张图片，每张不超过 5 MB；支持 JPEG、PNG、WebP、GIF。<span class="ml-1 hidden sm:inline">Ctrl / ⌘ + Enter 发表</span></p>
    <p v-if="error" role="alert" class="border-t border-danger/20 bg-danger-soft px-3 py-1.5 text-xs text-danger">{{ error }}</p>
  </div>
</template>
