<script setup lang="ts">
import { defineAsyncComponent, onBeforeUnmount, ref, watch } from 'vue'
import { Loader2, RotateCcw, X } from '@lucide/vue'

const ImageViewer = defineAsyncComponent(() => import('@/components/ImageViewer.vue'))

const props = defineProps<{
  files: File[]
  uploading: boolean
  error: string | null
}>()

const emit = defineEmits<{
  remove: [index: number]
  retry: []
}>()

const previews = ref<{ src: string; alt: string }[]>([])
const imageIndex = ref<number | null>(null)

function clearPreviews() {
  imageIndex.value = null
  for (const image of previews.value) URL.revokeObjectURL(image.src)
  previews.value = []
}

watch(() => props.files, (files) => {
  clearPreviews()
  previews.value = files.map(file => ({ src: URL.createObjectURL(file), alt: file.name }))
}, { immediate: true })

onBeforeUnmount(clearPreviews)
</script>

<template>
  <div v-if="previews.length" class="flex flex-wrap gap-2 px-3 pb-2">
    <p v-if="error" role="alert" class="sr-only">{{ error }}。可重试发表或移除图片。</p>
    <div v-for="(preview, index) in previews" :key="preview.src" class="relative">
      <button
        type="button"
        class="block cursor-zoom-in overflow-hidden rounded-md border border-line hover:border-accent"
        :aria-label="`预览图片 ${index + 1}`"
        aria-haspopup="dialog"
        :disabled="uploading || !!error"
        @click="imageIndex = index"
      >
        <img :src="preview.src" alt="" class="h-16 w-16 object-cover">
      </button>
      <div
        v-if="uploading"
        class="absolute inset-0 flex items-center justify-center rounded-md bg-ink/50"
        role="progressbar"
        :aria-label="`正在上传第 ${index + 1} 张图片`"
      >
        <Loader2 class="h-6 w-6 animate-spin text-white motion-reduce:animate-none" aria-hidden="true" />
      </div>
      <button
        v-else-if="error"
        type="button"
        class="absolute inset-0 flex flex-col items-center justify-center gap-1 rounded-md bg-ink/75 text-xs text-white hover:bg-ink/85"
        :aria-label="`第 ${index + 1} 张图片上传失败，重试发表评论`"
        :title="error"
        @click="emit('retry')"
      >
        <span>上传失败</span>
        <span class="flex items-center gap-1"><RotateCcw class="h-3 w-3" aria-hidden="true" />重试</span>
      </button>
      <button
        v-if="!uploading"
        type="button"
        class="absolute -right-1.5 -top-1.5 flex h-6 w-6 items-center justify-center rounded-full bg-ink text-white"
        :aria-label="`移除第 ${index + 1} 张图片`"
        @click="emit('remove', index)"
      >
        <X class="h-2.5 w-2.5" aria-hidden="true" />
      </button>
    </div>
    <ImageViewer v-if="imageIndex !== null" :images="previews" :index="imageIndex" @close="imageIndex = null" />
  </div>
</template>
