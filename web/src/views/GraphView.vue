<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { Focus, Loader2, MousePointer2, Network } from '@lucide/vue'
import AppShell from '@/components/AppShell.vue'
import GraphStage from '@/components/graph/GraphStage.vue'
import { useGraphComments } from '@/composables/useGraphComments'
import { useNodeStore } from '@/composables/useNodeStore'
import type { GraphNeighborhood } from '@/graph/types'

interface GraphStageExpose {
  fitView: () => void
  scrollComments: (delta: number) => void
}

const { nodes, byId, childrenOf, loading, error: dataError, load, reload } = useNodeStore()
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

// The whole tree is already in the node store, so the focused neighborhood is a
// pure lookup. childrenOf keeps the same (sortOrder, id) order as the server.
function selectNode(id: number) {
  const node = byId.value.get(id)
  if (!node) return
  const parent = node.parentId == null ? null : byId.value.get(node.parentId) ?? null
  active.value = { node, parent, children: childrenOf(id) }
}

function clearSelection() {
  active.value = null
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
    <div class="graph-view" @wheel.capture="onWheel">
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

      <header class="graph-header">
        <div class="graph-title">
          <span class="graph-title-mark"><Network :size="16" /></span>
          <div>
            <h1 class="graph-heading">关系图谱</h1>
            <p class="graph-kicker">GUILD NARRATIVE FIELD · 01</p>
          </div>
        </div>
        <div class="graph-status">
          <span>{{ nodes.length }} 个节点</span>
          <span class="graph-dot" />
          <span v-if="loading" class="graph-inline">
            <Loader2 :size="10" class="graph-spin" /> 正在展开档案
          </span>
          <span v-else-if="active && commentsLoading" class="graph-inline">
            <Loader2 :size="10" class="graph-spin" /> 正在展开评注
          </span>
          <span v-else-if="active">{{ comments.length }}{{ commentsHaveMore ? '+' : '' }} 条评注</span>
          <span v-else>图谱已完整展开</span>
        </div>
      </header>

      <div v-if="!active" class="graph-toolbar">
        <button
          type="button"
          class="graph-button"
          title="让全部已加载节点回到视野"
          @click="stage?.fitView()"
        >
          <Focus :size="14" /> 复位视野
        </button>
      </div>

      <aside class="graph-help">
        <div class="graph-help-row">
          <span class="graph-inline"><MousePointer2 :size="12" /> 点击节点聚焦</span>
          <span v-if="!active">拖拽平移 · 滚轮缩放 · 点击空白淡出</span>
          <span v-else>滚轮翻阅节点评注 · 点击空白返回</span>
        </div>
        <p class="graph-help-note">父子关系构成地面路径；鼠标移动会影响空间重心。</p>
      </aside>

      <div v-if="dataError" class="graph-error-overlay">
        <div class="graph-error-card">
          <p>{{ dataError }}</p>
          <button type="button" class="graph-error-action" @click="reload">重新加载图谱</button>
        </div>
      </div>

      <p v-if="commentsError" class="graph-toast" role="alert">{{ commentsError }}</p>
    </div>
  </AppShell>
</template>

<style scoped>
.graph-view {
  position: relative;
  flex: 1 1 auto;
  height: 100%;
  min-height: 0;
  overflow: hidden;
  background: #fff;
}

.graph-header,
.graph-toolbar,
.graph-help {
  position: absolute;
  z-index: 10;
  pointer-events: none;
}

.graph-header {
  top: 16px;
  left: 16px;
  max-width: min(420px, calc(100% - 32px));
  color: #64748b;
}

.graph-title {
  display: flex;
  align-items: center;
  gap: 8px;
}

.graph-title-mark {
  display: grid;
  place-items: center;
  width: 32px;
  height: 32px;
  border: 1px solid rgba(203, 213, 225, 0.7);
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.82);
  box-shadow: 0 1px 2px rgba(15, 23, 42, 0.08);
  backdrop-filter: blur(6px);
  color: var(--guild-primary);
}

.graph-heading {
  margin: 0;
  font-size: 13px;
  font-weight: 600;
  letter-spacing: 0.16em;
  color: var(--guild-text);
}

.graph-kicker {
  margin: 2px 0 0;
  font-size: 10px;
  letter-spacing: 0.12em;
  color: #94a3b8;
}

.graph-status {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 12px;
  font-size: 10px;
  color: #64748b;
}

.graph-dot {
  width: 4px;
  height: 4px;
  border-radius: 999px;
  background: #cbd5e1;
}

.graph-inline {
  display: inline-flex;
  align-items: center;
  gap: 4px;
}

.graph-spin {
  animation: graph-spin 1s linear infinite;
}

@keyframes graph-spin {
  to {
    transform: rotate(360deg);
  }
}

.graph-toolbar {
  top: 16px;
  right: 16px;
  display: flex;
  gap: 6px;
  pointer-events: auto;
}

.graph-button {
  display: inline-flex;
  height: 32px;
  align-items: center;
  gap: 6px;
  padding: 0 10px;
  border: 1px solid rgba(203, 213, 225, 0.7);
  border-radius: 6px;
  background: rgba(255, 255, 255, 0.85);
  color: #475569;
  font-size: 11px;
  line-height: 1;
  box-shadow: 0 1px 2px rgba(15, 23, 42, 0.08);
  backdrop-filter: blur(6px);
  cursor: pointer;
  transition: border-color 0.15s, color 0.15s;
}

.graph-button:hover {
  border-color: #94a3b8;
  color: var(--guild-text);
}

.graph-help {
  bottom: 16px;
  left: 16px;
  max-width: min(460px, calc(100% - 128px));
  padding: 8px 12px;
  border: 1px solid rgba(226, 232, 240, 0.9);
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.82);
  color: #64748b;
  font-size: 10px;
  line-height: 20px;
  box-shadow: 0 1px 2px rgba(15, 23, 42, 0.08);
  backdrop-filter: blur(12px);
}

.graph-help-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 4px 12px;
}

.graph-help-note {
  margin: 2px 0 0;
  color: #94a3b8;
}

.graph-error-overlay {
  position: absolute;
  inset: 0;
  z-index: 30;
  display: grid;
  place-items: center;
  padding: 24px;
}

.graph-error-card {
  padding: 16px 20px;
  border: 1px solid rgba(253, 164, 175, 0.2);
  border-radius: 12px;
  background: rgba(26, 22, 32, 0.92);
  color: #fecdd3;
  text-align: center;
  font-size: 14px;
  box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.25);
}

.graph-error-card p {
  margin: 0;
}

.graph-error-action {
  margin-top: 12px;
  padding: 6px 12px;
  border: 1px solid rgba(254, 205, 211, 0.25);
  border-radius: 6px;
  background: transparent;
  color: inherit;
  font-size: 12px;
  cursor: pointer;
}

.graph-error-action:hover {
  background: rgba(254, 205, 211, 0.1);
}

.graph-toast {
  position: absolute;
  bottom: 64px;
  left: 50%;
  z-index: 40;
  transform: translateX(-50%);
  margin: 0;
  padding: 6px 12px;
  border: 1px solid rgba(253, 164, 175, 0.2);
  border-radius: 6px;
  background: rgba(36, 21, 27, 0.92);
  color: #fecdd3;
  font-size: 12px;
  box-shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.2);
}

@media (prefers-reduced-motion: reduce) {
  .graph-spin {
    animation: none;
  }
}
</style>