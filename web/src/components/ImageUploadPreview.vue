<script setup lang="ts">
import { Loader2, RotateCcw, X } from '@lucide/vue'

defineProps<{
  src: string
  name: string
  uploading: boolean
  error: string | null
  disabled: boolean
}>()

const emit = defineEmits<{
  preview: []
  remove: []
  retry: []
}>()
</script>

<template>
  <div class="relative">
    <button
      type="button"
      class="block cursor-zoom-in overflow-hidden rounded-md border border-line hover:border-accent"
      :aria-label="`预览图片：${name}`"
      aria-haspopup="dialog"
      :disabled="uploading || !!error"
      @click="emit('preview')"
    >
      <img :src="src" alt="" class="h-16 w-16 object-cover">
    </button>
    <div
      v-if="uploading"
      class="absolute inset-0 flex items-center justify-center rounded-md bg-ink/50"
      role="progressbar"
      :aria-label="`正在上传：${name}`"
    >
      <Loader2 class="h-6 w-6 animate-spin text-white motion-reduce:animate-none" aria-hidden="true" />
    </div>
    <button
      v-else-if="error"
      type="button"
      class="absolute inset-0 flex flex-col items-center justify-center gap-1 rounded-md bg-ink/75 text-xs text-white hover:bg-ink/85"
      :aria-label="`重试上传：${name}`"
      :title="error"
      :disabled="disabled"
      @click="emit('retry')"
    >
      <span>上传失败</span>
      <span class="flex items-center gap-1"><RotateCcw class="h-3 w-3" aria-hidden="true" />重试</span>
    </button>
    <span v-if="error" role="alert" class="sr-only">{{ name }}：{{ error }}。可重试上传或移除图片。</span>
    <button
      v-if="!uploading"
      type="button"
      class="absolute -right-1.5 -top-1.5 flex h-6 w-6 items-center justify-center rounded-full bg-ink text-white"
      :aria-label="`移除图片：${name}`"
      :disabled="disabled"
      @click="emit('remove')"
    >
      <X class="h-2.5 w-2.5" aria-hidden="true" />
    </button>
  </div>
</template>
