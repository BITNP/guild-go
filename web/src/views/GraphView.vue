<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import {
  Focus,
  Loader2,
  MousePointer2,
  Network,
} from '@lucide/vue'
import AppShell from '@/components/AppShell.vue'
import GraphStage from '@/components/graph/GraphStage.vue'
import { useGraphData } from '@/composables/useGraphData'
import { useGraphComments } from '@/composables/useGraphComments'
import { fetchGraphNeighborhood, type GraphNeighborhood } from '@/lib/api'
import { errorMessage } from '@/lib/utils'

interface GraphStageExpose {
  fitView: () => void
  scrollComments: (delta: number) => void
}

const { nodes, loading, complete, error: dataError, loaded, load } = useGraphData()
const {
  items: comments,
  loading: commentsLoading,
  hasMore: commentsHaveMore,
  error: commentsError,
  load: loadComments,
  loadOlder: loadOlderComments,
} = useGraphComments()
const stage = ref<GraphStageExpose | null>(null)
const active = ref<GraphNeighborhood | null>(null)
const pendingId = ref<number | null>(null)
const selectionError = ref<string | null>(null)
let requestGeneration = 0

async function selectNode(id: number) {
  if (pendingId.value === id) return
  const current = ++requestGeneration
  pendingId.value = id
  selectionError.value = null
  try {
    const neighborhood = await fetchGraphNeighborhood(id)
    if (current !== requestGeneration) return
    active.value = neighborhood
  } catch (err) {
    if (current === requestGeneration) {
      selectionError.value = errorMessage(err, '无法聚焦该节点')
    }
  } finally {
    if (current === requestGeneration) pendingId.value = null
  }
}

function clearSelection() {
  requestGeneration++
  pendingId.value = null
  active.value = null
  selectionError.value = null
}

function onWheel(event: WheelEvent) {
  if (!active.value) return
  event.preventDefault()
  event.stopPropagation()
  stage.value?.scrollComments(event.deltaY)
}

onMounted(() => {
  void load()
})

watch(() => active.value?.node.id ?? null, (id) => {
  if (id == null) {
    commentsError.value = null
    return
  }
  void loadComments(id)
})
</script>

<template>
  <AppShell immersive>
    <div class="relative h-full min-h-0 overflow-hidden" @wheel.capture="onWheel">
      <GraphStage
        ref="stage"
        :nodes="nodes"
        :focus="active"
        :comments="comments"
        :comments-loading="commentsLoading"
        @select="selectNode"
        @blank="clearSelection"
        @comments-edge="loadOlderComments"
      />

      <header class="pointer-events-none absolute left-4 top-4 z-10 max-w-[min(420px,calc(100%-2rem))]">
        <div class="flex items-center gap-2 text-slate-600">
          <span class="grid h-8 w-8 place-items-center rounded-lg border border-slate-300/70 bg-white/82 shadow-sm backdrop-blur">
            <Network class="h-4 w-4 text-accent" />
          </span>
          <div>
            <h1 class="text-[13px] font-semibold tracking-[0.16em] text-ink">关系图谱</h1>
            <p class="mt-0.5 text-[10px] tracking-[0.12em] text-slate-400">GUILD NARRATIVE FIELD · 01</p>
          </div>
        </div>
        <div class="mt-3 flex items-center gap-2 text-[10px] text-slate-500">
          <span>{{ loaded }} 个节点</span>
          <span class="h-1 w-1 rounded-full bg-slate-300" />
          <span v-if="loading" class="inline-flex items-center gap-1">
            <Loader2 class="h-2.5 w-2.5 animate-spin" /> 正在展开档案
          </span>
          <span v-else-if="active && commentsLoading" class="inline-flex items-center gap-1">
            <Loader2 class="h-2.5 w-2.5 animate-spin" /> 正在展开评注
          </span>
          <span v-else-if="active">{{ comments.length }}{{ commentsHaveMore ? '+' : '' }} 条评注</span>
          <span v-else-if="complete">图谱已完整展开</span>
        </div>
      </header>

      <div v-if="!active" class="pointer-events-auto absolute right-4 top-4 z-10 flex items-center gap-1.5">
        <button
          type="button"
          class="inline-flex h-8 items-center gap-1.5 rounded-md border border-slate-300/70 bg-white/85 px-2.5 text-[11px] text-slate-600 shadow-sm backdrop-blur transition-colors hover:border-slate-400 hover:text-ink"
          title="让全部已加载节点回到视野"
          @click="stage?.fitView()"
        >
          <Focus class="h-3.5 w-3.5" /> 复位视野
        </button>
      </div>

      <aside class="pointer-events-none absolute bottom-4 left-4 z-10 max-w-[min(460px,calc(100%-8rem))] rounded-lg border border-slate-200/90 bg-white/82 px-3 py-2 text-[10px] leading-5 text-slate-500 shadow-sm backdrop-blur-md">
        <div class="flex flex-wrap items-center gap-x-3 gap-y-1">
          <span class="inline-flex items-center gap-1"><MousePointer2 class="h-3 w-3" /> 点击节点聚焦</span>
          <span v-if="!active">拖拽平移 · 滚轮缩放 · 点击空白淡出</span>
          <span v-else>滚轮翻阅节点评注 · 点击空白返回</span>
        </div>
        <p class="mt-0.5 text-slate-400">父子关系构成地面路径；鼠标移动会影响空间重心。</p>
      </aside>

      <div v-if="dataError" class="absolute inset-0 z-30 grid place-items-center p-6">
        <div class="rounded-xl border border-rose-300/20 bg-[#1a1620]/92 px-5 py-4 text-center text-sm text-rose-200 shadow-2xl">
          <p>{{ dataError }}</p>
          <button type="button" class="mt-3 rounded-md border border-rose-200/25 px-3 py-1.5 text-xs hover:bg-rose-200/10" @click="load">
            重新加载图谱
          </button>
        </div>
      </div>

      <p
        v-if="selectionError || commentsError"
        class="absolute bottom-16 left-1/2 z-40 -translate-x-1/2 rounded-md border border-rose-300/20 bg-[#24151b]/92 px-3 py-1.5 text-xs text-rose-200 shadow-xl"
        role="alert"
      >
        {{ selectionError || commentsError }}
      </p>

      <div
        v-if="pendingId != null && active == null"
        class="pointer-events-none absolute left-1/2 top-1/2 z-20 flex -translate-x-1/2 items-center gap-2 rounded-full border border-slate-300/70 bg-white/85 px-3 py-1.5 text-[11px] text-slate-600 shadow-sm backdrop-blur"
      >
        <Loader2 class="h-3 w-3 animate-spin" /> 正在靠近节点…
      </div>
    </div>
  </AppShell>
</template>
