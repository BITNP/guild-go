<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { NAlert, NButton, NSpin, useThemeVars } from 'naive-ui'
import type { DiffSurfaceController } from 'stream-diffs'

const props = defineProps<{ before: string; after: string }>()
const themeVars = useThemeVars()
const container = ref<HTMLElement>()
const loading = ref(true)
const failed = ref(false)
let controller: DiffSurfaceController | undefined
let disposed = false

onMounted(async () => {
  try {
    const { createDiffSurface } = await import('stream-diffs')
    if (disposed) return
    controller = createDiffSurface({
      kind: 'diff',
      // Descriptions are trimmed on save; terminate display lines to omit Git's EOF metadata.
      oldFile: { name: 'description.txt', contents: props.before ? `${props.before}\n` : '', lang: 'text' },
      newFile: { name: 'description.txt', contents: props.after ? `${props.after}\n` : '', lang: 'text' },
      options: {
        diffStyle: 'unified',
        diffIndicators: 'classic',
        lineDiffType: 'char',
        overflow: 'wrap',
        disableFileHeader: true,
        disableLineNumbers: true,
        expandUnchanged: true,
        themeType: 'light',
        preferredHighlighter: 'shiki-js',
      },
    })
    await controller.mount(container.value!)
    await controller.whenVisualReady()
  } catch {
    if (!disposed) {
      controller?.dispose()
      failed.value = true
    }
  } finally {
    if (!disposed) loading.value = false
  }
})

onBeforeUnmount(() => {
  disposed = true
  controller?.dispose()
})

function reloadPage() {
  window.location.reload()
}
</script>

<template>
  <div
    class="description-diff"
    :style="{
      '--diffs-font-family': themeVars.fontFamily,
      '--diffs-header-font-family': themeVars.fontFamily,
    }"
  >
    <NAlert v-if="failed" type="error" :show-icon="false">
      差异加载失败
      <NButton size="small" class="diff-retry" @click="reloadPage">刷新重试</NButton>
    </NAlert>
    <NSpin v-show="!failed" :show="loading" size="small" description="正在加载变更">
      <div
        ref="container"
        class="diff-content"
        role="region"
        aria-label="描述差异"
        :aria-busy="loading"
        tabindex="0"
      />
    </NSpin>
  </div>
</template>

<style scoped>
.description-diff {
  --diffs-font-size: 14px;
  --diffs-line-height: 24px;
  --diffs-gap-style: none;
}
.diff-content {
  min-height: 64px;
  max-height: 480px;
  overflow: auto;
}
.diff-retry {
  margin-left: 12px;
}
.diff-content:focus-visible {
  outline: 2px solid var(--guild-primary);
  outline-offset: 2px;
}
</style>
