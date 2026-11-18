<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import type { Comment, ProgressItem } from '@/lib/api'
import type { GraphNeighborhood, GraphNode } from '@/graph/types'
import { GraphScene } from '@/graph/GraphScene'
import GraphCommentComposer from '@/components/graph/GraphCommentComposer.vue'
import type { ThemeMode } from '@/composables/useTheme'

const props = defineProps<{
  nodes: GraphNode[]
  focus: GraphNeighborhood | null
  progress: ProgressItem[]
  progressLoading: boolean
  canComment: boolean
  theme: ThemeMode
}>()

const emit = defineEmits<{
  select: [id: number]
  blank: []
  reaction: [commentId: number, emoji: string]
  commentCreated: [comment: Comment]
  progressOpen: [key: string]
  progressEdge: []
}>()

const container = ref<HTMLDivElement | null>(null)
const composer = ref<{ $el: Element } | null>(null)
const error = ref<string | null>(null)
let scene: GraphScene | null = null

onMounted(() => {
  if (!container.value) return
  try {
    scene = new GraphScene(container.value, {
      onSelect: (id) => emit('select', id),
      onBlank: () => emit('blank'),
      onReaction: (commentId, emoji) => emit('reaction', commentId, emoji),
      onProgressOpen: (key) => emit('progressOpen', key),
      onProgressNeedOlder: () => emit('progressEdge'),
    }, props.theme)
    scene.setNodes(props.nodes)
    scene.setFocus(props.focus)
    scene.setProgress(props.progress, props.progressLoading)
    syncComposer()
  } catch (err) {
    console.error(err)
    error.value = '当前浏览器无法显示 3D 图谱。请开启浏览器的硬件加速后刷新页面。'
  }
})

watch(() => props.nodes, (nodes) => scene?.setNodes(nodes))
watch(() => props.focus, (focus) => scene?.setFocus(focus))
watch(() => props.theme, (theme) => scene?.setTheme(theme))
watch(
  [() => props.progress, () => props.progressLoading],
  ([progress, loading]) => scene?.setProgress(progress, loading),
)
watch(
  [composer, () => props.focus, () => props.canComment],
  syncComposer,
  { flush: 'post' },
)

onBeforeUnmount(() => {
  scene?.attachCommentComposer(null)
  scene?.destroy()
  scene = null
})

function syncComposer() {
  const element = composer.value?.$el
  scene?.attachCommentComposer(element instanceof HTMLElement ? element : null)
  scene?.setCommentComposerVisible(Boolean(props.focus && props.canComment))
}

function fitView() {
  scene?.fitView()
}

function scrollProgress(delta: number) {
  scene?.scrollComments(delta)
}

defineExpose({ fitView, scrollProgress })
</script>

<template>
  <div class="graph-stage">
    <div ref="container" class="graph-canvas">
      <GraphCommentComposer
        v-if="focus && canComment"
        ref="composer"
        :key="focus.node.id"
        :node-id="focus.node.id"
        @created="(comment) => emit('commentCreated', comment)"
      />
    </div>
    <div class="graph-horizon" aria-hidden="true" />
    <div class="graph-vignette" />
    <div v-if="error" class="graph-stage-error">
      <p class="graph-stage-error-card" role="alert">{{ error }}</p>
    </div>
  </div>
</template>

<style scoped>
.graph-stage {
  position: relative;
  width: 100%;
  height: 100%;
  overflow: hidden;
  background: var(--graph-bg);
}

.graph-canvas {
  position: absolute;
  inset: 0;
}

.graph-horizon,
.graph-vignette {
  position: absolute;
  inset: 0;
  pointer-events: none;
}

.graph-horizon {
  background: var(--graph-horizon-bg);
  mix-blend-mode: var(--graph-horizon-blend);
}

.graph-vignette {
  opacity: var(--graph-vignette-opacity);
  background: radial-gradient(
    ellipse at 50% 43%,
    transparent 20%,
    rgba(3, 7, 9, 0.1) 58%,
    rgba(2, 5, 7, 0.78) 100%
  );
}

.graph-stage-error {
  position: absolute;
  inset: 0;
  display: grid;
  place-items: center;
  padding: 24px;
}

.graph-stage-error-card {
  max-width: 28rem;
  margin: 0;
  padding: 12px 16px;
  border: 1px solid var(--graph-stage-error-border);
  border-radius: 8px;
  background: var(--graph-stage-error-bg);
  color: var(--graph-stage-error-color);
  font-size: 14px;
}
</style>
