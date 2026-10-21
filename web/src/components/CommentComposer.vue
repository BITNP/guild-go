<script setup lang="ts">
import { onBeforeUnmount, ref } from 'vue'
import { ImagePlus, Loader2, Send, X } from '@lucide/vue'
import { createComment, type Comment } from '@/lib/api'
import { errorMessage } from '@/lib/utils'

const props = defineProps<{
  nodeId: number
}>()

const emit = defineEmits<{
  created: [comment: Comment]
}>()

const body = ref('')
const files = ref<File[]>([])
const previews = ref<string[]>([])
const fileInput = ref<HTMLInputElement | null>(null)
const submitting = ref(false)
const error = ref<string | null>(null)

const MAX_FILES = 4
const MAX_SIZE = 5 * 1024 * 1024
const ALLOWED = new Set(['image/jpeg', 'image/png', 'image/webp', 'image/gif'])

function openPicker() {
  fileInput.value?.click()
}

function clearFiles() {
  for (const url of previews.value) URL.revokeObjectURL(url)
  files.value = []
  previews.value = []
}

function onFiles(event: Event) {
  const input = event.target as HTMLInputElement
  const selected = [...(input.files ?? [])]
  input.value = ''
  error.value = null
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
    files.value.push(file)
    previews.value.push(URL.createObjectURL(file))
  }
}

function removeFile(index: number) {
  URL.revokeObjectURL(previews.value[index]!)
  previews.value.splice(index, 1)
  files.value.splice(index, 1)
}

async function submit() {
  if (submitting.value) return
  error.value = null
  if (!body.value.trim() && files.value.length === 0) {
    error.value = '写点什么，或添加一张图片'
    return
  }
  submitting.value = true
  try {
    const comment = await createComment(props.nodeId, body.value, files.value)
    body.value = ''
    clearFiles()
    emit('created', comment)
  } catch (err) {
    error.value = errorMessage(err)
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

onBeforeUnmount(clearFiles)
</script>

<template>
  <div class="rounded-lg border border-line bg-surface">
    <textarea
      v-model="body"
      class="w-full resize-y rounded-t-lg border-0 bg-transparent px-3 py-2.5 text-[13px] leading-6 text-ink outline-none placeholder:text-faint"
      rows="3"
      maxlength="5000"
      placeholder="发表评论… 支持换行与 http(s) 链接，Ctrl/⌘ + Enter 发送"
      @keydown="onKeydown"
    />

    <div v-if="previews.length" class="flex flex-wrap gap-2 px-3 pb-2">
      <div v-for="(preview, index) in previews" :key="preview" class="relative">
        <img :src="preview" alt="待上传图片" class="h-16 w-16 rounded-md border border-line object-cover">
        <button
          type="button"
          class="absolute -right-1.5 -top-1.5 flex h-4 w-4 items-center justify-center rounded-full bg-ink text-white"
          aria-label="移除图片"
          @click="removeFile(index)"
        >
          <X class="h-2.5 w-2.5" />
        </button>
      </div>
    </div>

    <div class="flex items-center gap-2 border-t border-line px-3 py-2">
      <button type="button" class="btn btn-ghost btn-xs" :disabled="files.length >= MAX_FILES" @click="openPicker">
        <ImagePlus class="h-3.5 w-3.5" />
        图片
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
      <span class="text-[10px] text-faint">JPEG / PNG / WebP / GIF，单张 ≤ 5MB</span>
      <button type="button" class="btn btn-primary btn-xs ml-auto" :disabled="submitting" @click="submit">
        <Loader2 v-if="submitting" class="h-3.5 w-3.5 animate-spin" />
        <Send v-else class="h-3.5 w-3.5" />
        {{ submitting ? '发送中…' : '发表' }}
      </button>
    </div>

    <p v-if="error" class="border-t border-danger/20 bg-danger-soft px-3 py-1.5 text-[11px] text-danger">{{ error }}</p>
  </div>
</template>