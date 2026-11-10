<script setup lang="ts">
const emit = defineEmits<{ remove: []; retry: [] }>();
const props = withDefaults(
  defineProps<{
    image: CommentImageDraft;
    disabled: boolean;
    compact?: boolean;
  }>(),
  { compact: false },
);

const name = computed(() => props.image.file.name);
const uploading = computed(() => props.image.status.state === "uploading");
const error = computed(() =>
  props.image.status.state === "failed" ? props.image.status.message : null,
);
</script>

<template>
  <div class="upload-image" :class="{ compact }">
    <img
      v-if="compact"
      :alt="name"
      class="compact-thumbnail"
      :src="image.src"
    />
    <PreviewImage
      v-else
      :alt="name"
      class="upload-thumbnail"
      :disabled="uploading || !!error"
      :height="80"
      :src="image.src"
      :width="80"
    />
    <div
      v-if="uploading"
      :aria-label="`正在上传：${name}`"
      class="image-mask"
      role="progressbar"
    >
      <Loader2 class="spin" :size="compact ? 13 : 24" />
    </div>
    <button
      v-else-if="error"
      :aria-label="`重试上传：${name}`"
      class="image-mask failed retry-image"
      :disabled
      :title="error"
      type="button"
      @click="emit('retry')"
    >
      <span v-if="!compact">上传失败</span>
      <span class="retry-label">
        <RotateCcw :size="compact ? 13 : 12" />
        <span v-if="!compact">重试</span>
      </span>
      <span class="screen-reader" role="alert">{{ name }}：{{ error }}</span>
    </button>
    <button
      v-if="!uploading"
      :aria-label="`移除图片：${name}`"
      class="remove-image"
      :disabled
      type="button"
      @click="emit('remove')"
    >
      <X :size="12" />
    </button>
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
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 5px;
  padding: 0;
  border: 0;
  border-radius: 8px;
  background: rgb(23 32 43 / 55%);
  color: #fff;
  font: inherit;
  font-size: 12px;
}

.failed {
  background: rgb(23 32 43 / 78%);
}

.retry-image {
  width: 100%;
  height: 100%;
  cursor: pointer;
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
  display: grid;
  width: 22px;
  height: 22px;
  place-items: center;
  padding: 0;
  border: 1px solid var(--guild-border);
  border-radius: 50%;
  background: var(--guild-surface);
  color: inherit;
  cursor: pointer;
}

.retry-image:focus-visible,
.remove-image:focus-visible {
  outline: 2px solid var(--guild-primary);
  outline-offset: 2px;
}

button:disabled {
  cursor: not-allowed;
  opacity: 0.55;
}

.compact {
  width: 34px;
  height: 34px;
  overflow: hidden;
  border: 1px solid rgba(108, 148, 155, 0.34);
  border-radius: 2px;
  background: #0b1418;
}

.compact-thumbnail {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.compact .image-mask {
  gap: 0;
  border-radius: 0;
  background: rgba(3, 10, 13, 0.76);
  color: #d7e4e2;
}

.compact .retry-image:hover:not(:disabled) {
  color: #f0c46c;
}

.compact .remove-image {
  top: 2px;
  right: 2px;
  width: 16px;
  height: 16px;
  border-color: rgba(218, 185, 119, 0.3);
  background: rgba(4, 10, 12, 0.84);
  color: #cad7d5;
}

[data-theme="light"] .compact {
  border-color: rgba(80, 108, 120, 0.26);
  background: #eef3f5;
}

[data-theme="light"] .compact .image-mask {
  background: rgba(238, 244, 246, 0.86);
  color: #455b66;
}

[data-theme="light"] .compact .remove-image {
  border-color: rgba(121, 91, 47, 0.28);
  background: rgba(255, 255, 255, 0.9);
  color: #506570;
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

.spin {
  animation: image-spin 0.9s linear infinite;
}

@keyframes image-spin {
  to {
    transform: rotate(360deg);
  }
}

@media (prefers-reduced-motion: reduce) {
  .spin {
    animation: none;
  }
}
</style>
