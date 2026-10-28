<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import type { Comment } from '@/lib/api'
import type { GraphNeighborhood, GraphNode } from '@/graph/types'
import { GraphScene } from '@/graph/GraphScene'
import GraphCommentComposer from '@/components/graph/GraphCommentComposer.vue'

const props = defineProps<{
  nodes: GraphNode[]
  focus: GraphNeighborhood | null
  comments: Comment[]
  commentsLoading: boolean
  canComment: boolean
}>()

const emit = defineEmits<{
  select: [id: number]
  blank: []
  reaction: [commentId: number, emoji: string]
  commentCreated: [comment: Comment]
  commentsEdge: []
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
      onCommentsNeedOlder: () => emit('commentsEdge'),
    })
    scene.setNodes(props.nodes)
    scene.setFocus(props.focus)
    scene.setComments(props.comments, props.commentsLoading)
    syncComposer()
  } catch (err) {
    error.value = err instanceof Error ? err.message : '当前浏览器无法初始化 3D 画布'
  }
})

watch(() => props.nodes, (nodes) => scene?.setNodes(nodes))
watch(() => props.focus, (focus) => scene?.setFocus(focus))
watch(
  [() => props.comments, () => props.commentsLoading],
  ([comments, loading]) => scene?.setComments(comments, loading),
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

function scrollComments(delta: number) {
  scene?.scrollComments(delta)
}

defineExpose({ fitView, scrollComments })
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
    <div class="graph-hud-frame" aria-hidden="true">
      <span class="graph-corner is-tl" />
      <span class="graph-corner is-tr" />
      <span class="graph-corner is-bl" />
      <span class="graph-corner is-br" />
    </div>
    <div class="graph-scanline" aria-hidden="true" />
    <div v-if="error" class="graph-stage-error">
      <p class="graph-stage-error-card">无法创建关系图谱：{{ error }}</p>
    </div>
  </div>
</template>

<style scoped>
.graph-stage {
  position: relative;
  width: 100%;
  height: 100%;
  overflow: hidden;
  background: #070a0d;
}

.graph-canvas {
  position: absolute;
  inset: 0;
}

.graph-horizon,
.graph-vignette,
.graph-hud-frame,
.graph-scanline {
  position: absolute;
  inset: 0;
  pointer-events: none;
}

.graph-horizon {
  background:
    radial-gradient(circle at 72% 24%, rgba(232, 170, 58, 0.14), transparent 28%),
    radial-gradient(circle at 46% 48%, rgba(77, 167, 169, 0.08), transparent 38%);
  mix-blend-mode: screen;
}

.graph-vignette {
  background: radial-gradient(
    ellipse at 50% 43%,
    transparent 20%,
    rgba(3, 7, 9, 0.1) 58%,
    rgba(2, 5, 7, 0.78) 100%
  );
}

.graph-hud-frame {
  inset: 12px;
  opacity: 0.62;
}

.graph-corner {
  position: absolute;
  width: 34px;
  height: 34px;
  border-color: rgba(225, 178, 79, 0.48);
  border-style: solid;
}

.graph-corner.is-tl {
  top: 0;
  left: 0;
  border-width: 1px 0 0 1px;
}

.graph-corner.is-tr {
  top: 0;
  right: 0;
  border-width: 1px 1px 0 0;
}

.graph-corner.is-bl {
  bottom: 0;
  left: 0;
  border-width: 0 0 1px 1px;
}

.graph-corner.is-br {
  right: 0;
  bottom: 0;
  border-width: 0 1px 1px 0;
}

.graph-scanline {
  inset: auto 7% 9%;
  height: 1px;
  background: linear-gradient(90deg, transparent, rgba(239, 183, 72, 0.42), transparent);
  box-shadow: 0 0 18px rgba(239, 183, 72, 0.28);
  opacity: 0.5;
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
  border: 1px solid rgba(245, 174, 82, 0.36);
  border-radius: 8px;
  background: rgba(11, 16, 18, 0.94);
  color: #f3c66f;
  font-size: 14px;
}
</style>
