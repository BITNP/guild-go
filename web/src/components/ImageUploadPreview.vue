<script setup lang="ts">
import { NButton, NSpin, NTooltip } from 'naive-ui'
import PreviewImage from '@/components/PreviewImage.vue'
import { RotateCcw, X } from '@lucide/vue'
defineProps<{
  src: string
  name: string
  uploading: boolean
  error: string | null
  disabled: boolean
}>()
const emit = defineEmits<{ remove: []; retry: [] }>()
</script>
<template>
  <div class="upload-image">
    <PreviewImage
      :src="src"
      :alt="name"
      :width="80"
      :height="80"
      :disabled="uploading || !!error"
      class="upload-thumbnail"
    />
    <div
      v-if="uploading"
      class="image-mask"
      role="progressbar"
      :aria-label="`正在上传：${name}`"
    >
      <NSpin :size="24" stroke="white" />
    </div>
    <div v-else-if="error" class="image-mask failed">
      <NTooltip
        ><template #trigger
          ><NButton
            text
            class="retry-image"
            :disabled="disabled"
            :aria-label="`重试上传：${name}`"
            @click="emit('retry')"
            ><span class="retry-content"
              ><span>上传失败</span
              ><span class="retry-label"
                ><RotateCcw :size="12" />重试</span
              ></span
            ></NButton
          ></template
        >{{ error }}</NTooltip
      ><span class="screen-reader" role="alert">{{ name }}：{{ error }}</span>
    </div>
    <NButton
      v-if="!uploading"
      circle
      size="tiny"
      class="remove-image"
      :disabled="disabled"
      :aria-label="`移除图片：${name}`"
      @click="emit('remove')"
      ><X :size="12"
    /></NButton>
  </div>
</template>
<style scoped>
.upload-image {
  position: relative;
  width: 80px;
  height: 80px;
  flex: none;
}
.upload-thumbnail {
  border-radius: 8px;
  overflow: hidden;
  border: 1px solid var(--guild-border);
}
.image-mask {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 8px;
  background: rgb(23 32 43 / 55%);
}
.failed {
  background: rgb(23 32 43 / 78%);
}
.retry-image {
  color: #fff;
  width: 100%;
  height: 100%;
}
.retry-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 5px;
  font-size: 12px;
}
.retry-label {
  display: flex;
  align-items: center;
  gap: 4px;
}
.remove-image {
  position: absolute;
  right: -6px;
  top: -6px;
}
.screen-reader {
  position: absolute;
  width: 1px;
  height: 1px;
  padding: 0;
  margin: -1px;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  white-space: nowrap;
  border: 0;
}
</style>
