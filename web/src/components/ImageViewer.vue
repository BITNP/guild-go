<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import { ChevronLeft, ChevronRight, Maximize, X, ZoomIn, ZoomOut } from '@lucide/vue'
import VueEasyLightbox from 'vue-easy-lightbox/external-css'
import 'vue-easy-lightbox/external-css/vue-easy-lightbox.css'

const props = defineProps<{
  images: { src: string, alt: string }[]
  index: number
}>()

const emit = defineEmits<{ close: [] }>()
const dialog = ref<HTMLDialogElement | null>(null)
const closeButton = ref<HTMLButtonElement | null>(null)
const visible = ref(false)
const currentIndex = ref(props.index)
const failed = ref(false)
const currentImage = computed(() => props.images[currentIndex.value]!)

function onIndexChange(_previous: number, index: number) {
  currentIndex.value = index
  failed.value = false
}

onMounted(async () => {
  dialog.value?.showModal()
  visible.value = true
  await nextTick()
  closeButton.value?.focus()
})

onBeforeUnmount(() => dialog.value?.close())
</script>

<template>
  <Teleport to="body">
    <dialog
      ref="dialog"
      aria-label="图片查看器"
      class="image-viewer fixed inset-0 m-0 h-dvh max-h-none w-screen max-w-none overflow-hidden border-0 bg-transparent p-0 text-white"
      @cancel.prevent="emit('close')"
    >
      <VueEasyLightbox
        :visible="visible"
        :imgs="images"
        :index="index"
        :zoom-scale="0.25"
        esc-disabled
        title-disabled
        rotate-disabled
        @hide="emit('close')"
        @on-index-change="onIndexChange"
        @on-error="failed = true"
      >
        <template #close-btn="{ close }">
          <span class="absolute left-4 top-4 rounded-md bg-ink/80 px-3 py-2 text-sm tabular-nums" aria-live="polite">
            {{ currentIndex + 1 }} / {{ images.length }}
          </span>
          <button ref="closeButton" type="button" class="viewer-control absolute right-4 top-4" aria-label="关闭大图" title="关闭（Esc）" @click="close">
            <X class="h-5 w-5" />
          </button>
        </template>

        <template #prev-btn="{ prev }">
          <button v-if="images.length > 1" type="button" class="viewer-control absolute left-3 top-1/2 -translate-y-1/2" :disabled="currentIndex === 0" aria-label="上一张" title="上一张（←）" @click="prev">
            <ChevronLeft class="h-5 w-5" />
          </button>
        </template>

        <template #next-btn="{ next }">
          <button v-if="images.length > 1" type="button" class="viewer-control absolute right-3 top-1/2 -translate-y-1/2" :disabled="currentIndex === images.length - 1" aria-label="下一张" title="下一张（→）" @click="next">
            <ChevronRight class="h-5 w-5" />
          </button>
        </template>

        <template #toolbar="{ toolbarMethods }">
          <div class="absolute bottom-4 left-1/2 flex -translate-x-1/2 items-center gap-1 rounded-lg border border-white/20 bg-ink p-1">
            <button type="button" class="viewer-control" :disabled="failed" aria-label="缩小" title="缩小" @click="toolbarMethods.zoomOut">
              <ZoomOut class="h-5 w-5" />
            </button>
            <button type="button" class="viewer-control" :disabled="failed" aria-label="放大" title="放大" @click="toolbarMethods.zoomIn">
              <ZoomIn class="h-5 w-5" />
            </button>
            <button type="button" class="viewer-control" :disabled="failed" aria-label="适应窗口" title="适应窗口" @click="toolbarMethods.resize">
              <Maximize class="h-5 w-5" />
            </button>
            <a :href="currentImage.src" target="_blank" rel="noopener noreferrer" class="viewer-control whitespace-nowrap text-[13px]" aria-label="在新标签页打开原图">打开原图</a>
          </div>
        </template>

        <template #onerror>
          <div class="absolute inset-0 flex flex-col items-center justify-center gap-2 px-16 text-center" role="alert">
            <p>图片加载失败</p>
            <p class="text-sm text-white/75">可打开原图，或关闭后重试。</p>
          </div>
        </template>
      </VueEasyLightbox>
    </dialog>
  </Teleport>
</template>

<style scoped>
.image-viewer :deep(.vel-modal) {
  background: rgb(15 23 34 / 90%);
}

.image-viewer :deep(.vel-img) {
  max-height: calc(100dvh - 10rem);
  box-shadow: none;
}

.viewer-control {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 44px;
  min-height: 44px;
  padding: 10px;
  border-radius: 6px;
  background: rgb(23 32 43 / 85%);
  color: white;
}

.viewer-control:hover {
  background: #3c495a;
}

.viewer-control:focus-visible {
  outline-color: white;
}

.viewer-control:disabled {
  opacity: 0.35;
  cursor: default;
}
</style>
