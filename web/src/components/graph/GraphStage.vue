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
  <div class="relative h-full w-full overflow-hidden bg-white">
    <div ref="container" class="absolute inset-0" />
    <div class="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_50%_42%,transparent_18%,rgba(241,245,249,0.08)_65%,rgba(226,232,240,0.28)_100%)]" />
    <div v-if="error" class="absolute inset-0 grid place-items-center p-6">
      <p class="max-w-md rounded-lg border border-danger/30 bg-danger-soft/95 px-4 py-3 text-sm text-danger">
        无法创建关系图谱：{{ error }}
      </p>
    </div>
  </div>
</template>
