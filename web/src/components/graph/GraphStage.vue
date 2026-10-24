<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import type { Comment } from '@/lib/api'
import type { GraphNeighborhood, GraphNode } from '@/graph/types'
import { GraphScene } from '@/graph/GraphScene'

const props = defineProps<{
  nodes: GraphNode[]
  focus: GraphNeighborhood | null
  comments: Comment[]
  commentsLoading: boolean
}>()

const emit = defineEmits<{
  select: [id: number]
  blank: []
  commentsEdge: []
}>()

const container = ref<HTMLDivElement | null>(null)
const error = ref<string | null>(null)
let scene: GraphScene | null = null

onMounted(() => {
  if (!container.value) return
  try {
    scene = new GraphScene(container.value, {
      onSelect: (id) => emit('select', id),
      onBlank: () => emit('blank'),
      onCommentsNeedOlder: () => emit('commentsEdge'),
    })
    scene.setNodes(props.nodes)
    scene.setFocus(props.focus)
    scene.setComments(props.comments, props.commentsLoading)
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

onBeforeUnmount(() => {
  scene?.destroy()
  scene = null
})

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
    <div ref="container" class="graph-canvas" />
    <div class="graph-vignette" />
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
  background: #fff;
}

.graph-canvas {
  position: absolute;
  inset: 0;
}

.graph-vignette {
  position: absolute;
  inset: 0;
  pointer-events: none;
  background: radial-gradient(
    circle at 50% 42%,
    transparent 18%,
    rgba(241, 245, 249, 0.08) 65%,
    rgba(226, 232, 240, 0.28) 100%
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
  border: 1px solid rgba(194, 59, 59, 0.3);
  border-radius: 8px;
  background: rgba(253, 242, 242, 0.95);
  color: var(--guild-danger);
  font-size: 14px;
}
</style>
