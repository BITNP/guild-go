<script setup lang="ts">
import type { DiffSurfaceController } from "stream-diffs";

const props = defineProps<{ before: string; after: string }>();
const themeVars = useThemeVars();
const { isDark } = useTheme();
const container = ref<HTMLElement>();
const loading = ref(true);
const failed = ref(false);
let controller: DiffSurfaceController | undefined;
let disposed = false;
let generation = 0;

async function mountDiff() {
  controller?.dispose();
  controller = undefined;
  failed.value = false;
  loading.value = true;
  const current = ++generation;
  try {
    const { createDiffSurface } = await import("stream-diffs");
    if (disposed || current !== generation) {
      return;
    }
    controller = createDiffSurface({
      kind: "diff",
      // Descriptions are trimmed on save; terminate display lines to omit Git's EOF metadata.
      oldFile: {
        name: "description.txt",
        contents: props.before ? `${props.before}\n` : "",
        lang: "text",
      },
      newFile: {
        name: "description.txt",
        contents: props.after ? `${props.after}\n` : "",
        lang: "text",
      },
      options: {
        diffStyle: "unified",
        diffIndicators: "classic",
        lineDiffType: "char",
        overflow: "wrap",
        disableFileHeader: true,
        disableLineNumbers: true,
        expandUnchanged: true,
        themeType: isDark.value ? "dark" : "light",
        preferredHighlighter: "shiki-js",
      },
    });
    await controller.mount(container.value!);
    await controller.whenVisualReady();
  } catch {
    if (!disposed && current === generation) {
      controller?.dispose();
      failed.value = true;
    }
  } finally {
    if (!disposed && current === generation) {
      loading.value = false;
    }
  }
}

onMounted(mountDiff);
watch(isDark, mountDiff);

onBeforeUnmount(() => {
  disposed = true;
  generation++;
  controller?.dispose();
});

function reloadPage() {
  window.location.reload();
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
    <NAlert v-if="failed" :show-icon="false" type="error">
      差异加载失败
      <NButton class="diff-retry" size="small" @click="reloadPage">
        刷新重试
      </NButton>
    </NAlert>
    <NSpin
      v-show="!failed"
      description="正在加载变更"
      :show="loading"
      size="small"
    >
      <div
        ref="container"
        :aria-busy="loading"
        aria-label="描述差异"
        class="diff-content"
        role="region"
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
