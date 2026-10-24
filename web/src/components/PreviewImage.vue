<script setup lang="ts">
import { ref } from 'vue'
import { NButton, NImage, type ImageInst } from 'naive-ui'
const props = withDefaults(
  defineProps<{
    src: string
    alt: string
    width?: number
    height?: number
    disabled?: boolean
    lazy?: boolean
  }>(),
  { width: 80, height: 80, disabled: false, lazy: false },
)
const image = ref<ImageInst | null>(null),
  attempt = ref(0)
function previewKey(event: KeyboardEvent) {
  if (!props.disabled && (event.key === 'Enter' || event.key === ' ')) {
    event.preventDefault()
    image.value?.showPreview()
  }
}
</script>
<template>
  <NImage
    :key="attempt"
    ref="image"
    :src="src"
    :alt="disabled ? alt : `预览${alt}`"
    :width="width"
    :height="height"
    object-fit="cover"
    :lazy="lazy"
    :preview-disabled="disabled"
    :img-props="{
      tabindex: disabled ? undefined : 0,
      role: disabled ? undefined : 'button',
      onKeydown: previewKey,
    }"
    class="preview-image"
    :style="{ width: `${width}px`, height: `${height}px` }"
    ><template #error
      ><NButton
        text
        class="image-error"
        :aria-label="`重新加载${alt}`"
        @click="attempt++"
        >图片加载失败<br />重试</NButton
      ></template
    ></NImage
  >
</template>
<style scoped>
.preview-image {
  border: 1px solid var(--guild-border);
  border-radius: 8px;
  overflow: hidden;
  flex: none;
}
.image-error {
  width: 100%;
  height: 100%;
  font-size: 12px;
  text-align: center;
}
.preview-image :deep(img:focus-visible) {
  outline: 2px solid var(--guild-primary);
  outline-offset: -3px;
}
</style>
