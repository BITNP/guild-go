<script setup lang="ts">
const emit = defineEmits<{ remove: []; retry: [] }>();
defineProps<{
  src: string;
  name: string;
  uploading: boolean;
  error: string | null;
  disabled: boolean;
}>();
</script>

<template>
  <div class="upload-image">
    <PreviewImage
      :alt="name"
      class="upload-thumbnail"
      :disabled="uploading || !!error"
      :height="80"
      :src
      :width="80"
    />
    <div
      v-if="uploading"
      :aria-label="`正在上传：${name}`"
      class="image-mask"
      role="progressbar"
    >
      <NSpin :size="24" stroke="white" />
    </div>
    <div v-else-if="error" class="image-mask failed">
      <NTooltip>
        <template #trigger>
          <NButton
            :aria-label="`重试上传：${name}`"
            class="retry-image"
            :disabled
            text
            @click="emit('retry')"
          >
            <span class="retry-content">
              <span>上传失败</span>
              <span class="retry-label">
                <RotateCcw :size="12" />
                重试
              </span>
            </span>
          </NButton>
        </template>
        {{ error }}
      </NTooltip>
      <span class="screen-reader" role="alert">{{ name }}：{{ error }}</span>
    </div>
    <NButton
      v-if="!uploading"
      :aria-label="`移除图片：${name}`"
      circle
      class="remove-image"
      :disabled
      size="tiny"
      @click="emit('remove')"
    >
      <X :size="12" />
    </NButton>
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
