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
        <p class="graph-help-note">同层同环，子节点沿父节点扇区向外展开；聚焦时从外圈回望公会核心。</p>
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
  background: #070a0d;
}

.graph-header,
.graph-toolbar,
.graph-help {
  position: absolute;
  z-index: 10;
  pointer-events: none;
}

.graph-header {
  top: 28px;
  left: 30px;
  max-width: min(420px, calc(100% - 32px));
  color: #a6b8be;
}

.graph-title {
  display: flex;
  align-items: center;
  gap: 8px;
}

.graph-title-mark {
  display: grid;
  place-items: center;
  width: 38px;
  height: 38px;
  border: 1px solid rgba(236, 181, 75, 0.72);
  clip-path: polygon(22% 0, 100% 0, 100% 78%, 78% 100%, 0 100%, 0 22%);
  background:
    linear-gradient(135deg, rgba(238, 181, 73, 0.2), transparent 62%),
    rgba(10, 17, 20, 0.88);
  box-shadow:
    inset 0 0 18px rgba(237, 180, 72, 0.08),
    0 0 24px rgba(237, 180, 72, 0.08);
  color: #f0bd57;
}

.graph-heading {
  margin: 0;
  font-size: 15px;
  font-weight: 700;
  letter-spacing: 0.2em;
  color: #f1ede2;
}

.graph-kicker {
  margin: 2px 0 0;
  font-size: 9px;
  font-weight: 600;
  letter-spacing: 0.18em;
  color: rgba(232, 182, 86, 0.72);
}

.graph-status {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 12px;
  font-size: 10px;
  color: #9cafb6;
}

.graph-dot {
  width: 4px;
  height: 4px;
  border-radius: 999px;
  background: #e8b44c;
  box-shadow: 0 0 8px rgba(232, 180, 76, 0.8);
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
  top: 28px;
  right: 28px;
  display: flex;
  gap: 6px;
  pointer-events: auto;
}

.graph-button {
  display: inline-flex;
  height: 32px;
  align-items: center;
  gap: 6px;
  padding: 0 12px;
  border: 1px solid rgba(221, 176, 83, 0.42);
  border-radius: 3px;
  background: rgba(8, 14, 17, 0.78);
  color: #e6c37b;
  font-size: 11px;
  line-height: 1;
  box-shadow:
    inset 0 0 16px rgba(225, 173, 70, 0.04),
    0 8px 24px rgba(0, 0, 0, 0.2);
  backdrop-filter: blur(12px);
  cursor: pointer;
  transition: border-color 0.15s, color 0.15s, background 0.15s;
}

.graph-button:hover {
  border-color: rgba(242, 190, 83, 0.8);
  background: rgba(25, 27, 20, 0.9);
  color: #fff3d1;
}

.graph-help {
  bottom: 26px;
  left: 28px;
  max-width: min(460px, calc(100% - 128px));
  padding: 9px 13px;
  border: 1px solid rgba(142, 164, 168, 0.22);
  border-left-color: rgba(230, 178, 73, 0.64);
  border-radius: 2px;
  background: rgba(7, 13, 16, 0.76);
  color: #9eb0b4;
  font-size: 10px;
  line-height: 20px;
  box-shadow: 0 12px 36px rgba(0, 0, 0, 0.22);
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
  color: rgba(224, 184, 101, 0.68);
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
  border: 1px solid rgba(240, 182, 82, 0.32);
  border-radius: 3px;
  background: rgba(7, 13, 16, 0.94);
  color: #f2c66f;
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
  border: 1px solid rgba(240, 182, 82, 0.36);
  border-radius: 2px;
  background: transparent;
  color: inherit;
  font-size: 12px;
  cursor: pointer;
}

.graph-error-action:hover {
  background: rgba(240, 182, 82, 0.1);
}

.graph-toast {
  position: absolute;
  bottom: 64px;
  left: 50%;
  z-index: 40;
  transform: translateX(-50%);
  margin: 0;
  padding: 6px 12px;
  border: 1px solid rgba(240, 182, 82, 0.3);
  border-radius: 2px;
  background: rgba(8, 14, 17, 0.94);
  color: #f2c66f;
  font-size: 12px;
  box-shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.2);
}

@media (prefers-reduced-motion: reduce) {
  .graph-spin {
    animation: none;
  }
}

@media (max-width: 640px) {
  .graph-header {
    top: 20px;
    left: 20px;
  }

  .graph-toolbar {
    top: 20px;
    right: 20px;
  }

  .graph-help {
    right: 20px;
    bottom: 20px;
    left: 20px;
    max-width: none;
  }

  .graph-help-note {
    display: none;
  }
}
</style>
