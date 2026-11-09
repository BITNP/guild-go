<script setup lang="ts">
import type { ImageInst } from "naive-ui";

const props = withDefaults(
  defineProps<{
    src: string;
    alt: string;
    width?: number;
    height?: number;
    disabled?: boolean;
    lazy?: boolean;
  }>(),
  { width: 80, height: 80, disabled: false, lazy: false },
);
const image = ref<ImageInst | null>(null);
const attempt = ref(0);

function previewKey(event: KeyboardEvent) {
  if (!props.disabled && (event.key === "Enter" || event.key === " ")) {
    event.preventDefault();
    image.value?.showPreview();
  }
}
</script>

<template>
  <NImage
    :key="attempt"
    ref="image"
    :alt="disabled ? alt : `预览${alt}`"
    class="preview-image"
    :height
    :img-props="{
      tabindex: disabled ? undefined : 0,
      role: disabled ? undefined : 'button',
      onKeydown: previewKey,
    }"
    :lazy
    object-fit="cover"
    :preview-disabled="disabled"
    :src
    :style="{ width: `${width}px`, height: `${height}px` }"
    :width
  >
    <template #error>
      <NButton
        :aria-label="`重新加载${alt}`"
        class="image-error"
        text
        @click="attempt++"
      >
        图片加载失败
        <br />
        重试
      </NButton>
    </template>
  </NImage>
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
