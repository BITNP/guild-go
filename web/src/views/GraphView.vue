<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { Focus, Loader2, Network } from '@lucide/vue'
import AppShell from '@/components/AppShell.vue'
import DeleteNodeDialog from '@/components/DeleteNodeDialog.vue'
import GraphNodePanel from '@/components/graph/GraphNodePanel.vue'
import GraphProgressDetail from '@/components/graph/GraphProgressDetail.vue'
import GraphStage from '@/components/graph/GraphStage.vue'
import '@/components/graph/graph-theme.css'
import NodeFormDialog from '@/components/NodeFormDialog.vue'
import OrderDialog from '@/components/OrderDialog.vue'
import MoveNodeDialog from '@/components/MoveNodeDialog.vue'
import { useGraphSelection } from '@/composables/useGraphSelection'
import { useNodeStore } from '@/composables/useNodeStore'
import { useTheme } from '@/composables/useTheme'
import { VIRTUAL_ROOT_ID } from '@/graph/layout'
import type { NodeDetail } from '@/lib/api'

interface GraphStageExpose {
  fitView: () => void
  scrollProgress: (delta: number) => void
}

const {
  nodes,
  loading,
  error: dataError,
  load,
  reload,
  removeSubtree,
} = useNodeStore()
const { mode: themeMode } = useTheme()
const {
  active,
  detail,
  detailLoading,
  detailError,
  selectionLoading,
  progress,
  progressLoading,
  progressHaveMore,
  progressError,
  loadProgress,
  loadOlderProgress,
  expandedProgressKey,
  expandedProgress,
  reacting,
  reactionError,
  selectNode,
  clearSelection,
  loadDetail,
  syncUpdatedNode,
  onCommentCreated,
  onCommentDeleted,
  reactToComment,
} = useGraphSelection()

const stage = ref<GraphStageExpose | null>(null)
const editOpen = ref(false)
const createOpen = ref(false)
const createParentId = ref<number | null>(null)
const createParentTitle = ref('')
const deleteOpen = ref(false)
const deleteTarget = ref<NodeDetail | null>(null)
const orderOpen = ref(false)
const moveOpen = ref(false)

function openProgress(key: string) {
  expandedProgressKey.value = key
}

function onDetailUpdated(node: NodeDetail) {
  syncUpdatedNode(node)
  void reload()
}

function openCreateRoot() {
  createParentId.value = null
  createParentTitle.value = ''
  createOpen.value = true
}

function openCreateChild() {
  if (!detail.value) return
  createParentId.value = detail.value.id
  createParentTitle.value = detail.value.title
  createOpen.value = true
}

function openDelete() {
  if (!detail.value?.permissions.canDelete) return
  deleteTarget.value = detail.value
  deleteOpen.value = true
}

async function onNodeSaved(node: NodeDetail) {
  editOpen.value = false
  syncUpdatedNode(node)
  await reload()
  if (active.value?.node.id === node.id) {
    await loadProgress(node.id)
  }
}

async function onNodeCreated(node: NodeDetail) {
  createOpen.value = false
  await reload()
  selectNode(node.id)
}

async function onNodeMoved(node: NodeDetail) {
  moveOpen.value = false
  await reload()
  if (active.value?.node.id === node.id) {
    selectNode(node.id)
    syncUpdatedNode(node)
    await loadProgress(node.id)
  }
}

async function onNodeDeleted() {
  const deletedId = deleteTarget.value?.id
  const parentId = deleteTarget.value?.parentId ?? null
  deleteOpen.value = false
  deleteTarget.value = null
  if (deletedId != null) removeSubtree(deletedId)
  clearSelection()
  await reload()
  selectNode(parentId ?? VIRTUAL_ROOT_ID)
}

async function onNodesOrdered() {
  orderOpen.value = false
  await reload()
  if (active.value?.node.id === VIRTUAL_ROOT_ID) {
    selectNode(VIRTUAL_ROOT_ID)
  } else {
    await loadDetail()
  }
}

function onProgressCommentDeleted(commentId: number) {
  onCommentDeleted(commentId)
  void reload()
}

function onWheel(event: WheelEvent) {
  if (
    !active.value ||
    active.value.node.id === VIRTUAL_ROOT_ID ||
    expandedProgress.value
  ) {
    return
  }
  if (
    event.target instanceof Element &&
    event.target.closest(
      '[data-graph-node-panel], [data-graph-comment-composer], [data-graph-progress-detail]',
    )
  ) {
    return
  }
  event.preventDefault()
  event.stopPropagation()
  stage.value?.scrollProgress(event.deltaY)
}

function onWindowKeydown(event: KeyboardEvent) {
  if (
    event.key !== 'Escape' ||
    event.defaultPrevented ||
    event.isComposing ||
    event.keyCode === 229 ||
    !active.value
  ) {
    return
  }
  if (
    editOpen.value ||
    createOpen.value ||
    deleteOpen.value ||
    moveOpen.value ||
    orderOpen.value ||
    expandedProgress.value ||
    document.querySelector(
      '.n-modal-container, .n-image-preview-container, .n-drawer-container, .n-dropdown-menu',
    )
  ) {
    return
  }
  clearSelection()
}

onMounted(() => {
  window.addEventListener('keydown', onWindowKeydown)
  void load()
})

onBeforeUnmount(() => {
  window.removeEventListener('keydown', onWindowKeydown)
})

watch(
  () => active.value?.node.id ?? null,
  () => {
    editOpen.value = false
    createOpen.value = false
    deleteOpen.value = false
    orderOpen.value = false
    deleteTarget.value = null
    moveOpen.value = false
  },
)
</script>

<template>
  <AppShell immersive>
    <div
      class="graph-view"
      :data-graph-theme="themeMode"
      @wheel.capture="onWheel"
    >
      <GraphStage
        ref="stage"
        :nodes="nodes"
        :focus="active"
        :progress="progress"
        :progress-loading="progressLoading"
        :theme="themeMode"
        :can-comment="
          !!(
            active &&
            detail?.id === active.node.id &&
            detail.permissions.canComment
          )
        "
        @select="selectNode"
        @blank="clearSelection"
        @reaction="reactToComment"
        @comment-created="onCommentCreated"
        @progress-open="openProgress"
        @progress-edge="loadOlderProgress"
      />

      <Transition name="progress-detail">
        <GraphProgressDetail
          v-if="expandedProgress"
          :item="expandedProgress"
          :node-title="active?.node.title ?? ''"
          :busy="
            expandedProgress.kind === 'comment' &&
            reacting.has(expandedProgress.comment.id)
          "
          @close="expandedProgressKey = null"
          @reaction="reactToComment"
          @deleted="onProgressCommentDeleted"
        />
      </Transition>

      <Transition name="node-panel">
        <GraphNodePanel
          v-if="active"
          :selected="active.node"
          :detail="detail?.id === active.node.id ? detail : null"
          :parent-id="active.parent?.id ?? null"
          :parent-title="active.parent?.title ?? null"
          :loading="detailLoading"
          :error="detailError"
          :is-virtual-root="active.node.id === VIRTUAL_ROOT_ID"
          @close="clearSelection"
          @navigate="selectNode"
          @retry="loadDetail"
          @updated="onDetailUpdated"
          @edit="editOpen = true"
          @move="moveOpen = true"
          @remove="openDelete"
          @create-child="openCreateChild"
          @create-root="openCreateRoot"
          @reorder="orderOpen = true"
        />
      </Transition>

      <header class="graph-header">
        <div class="graph-title">
          <span class="graph-title-mark"><Network :size="16" /></span>
          <h1 class="graph-heading">关系图谱</h1>
        </div>
        <div class="graph-status">
          <span>{{ nodes.length }} 个节点</span>
          <template
            v-if="loading || (active && active.node.id !== VIRTUAL_ROOT_ID)"
          >
            <span class="graph-dot" />
            <span v-if="loading" class="graph-inline">
              <Loader2 :size="12" class="graph-spin" /> 正在加载
            </span>
            <span v-else-if="selectionLoading" class="graph-inline">
              <Loader2 :size="12" class="graph-spin" /> 正在加载进展
            </span>
            <span v-else>
              {{ progress.length }}{{ progressHaveMore ? '+' : '' }} 条进展
            </span>
          </template>
        </div>
        <div class="graph-toolbar">
          <button
            v-if="active"
            type="button"
            class="graph-button"
            @click="clearSelection"
          >
            <Focus :size="14" aria-hidden="true" /> 返回全图
          </button>
          <button
            v-else
            type="button"
            class="graph-button"
            @click="stage?.fitView()"
          >
            <Focus :size="14" aria-hidden="true" /> 重置视角
          </button>
        </div>
      </header>

      <div v-if="dataError" class="graph-error-overlay">
        <div class="graph-error-card">
          <p>{{ dataError }}</p>
          <button type="button" class="graph-error-action" @click="reload">
            重新加载图谱
          </button>
        </div>
      </div>

      <p
        v-if="progressError || reactionError"
        class="graph-toast"
        role="alert"
      >
        {{ progressError || reactionError }}
      </p>
    </div>

    <NodeFormDialog
      v-if="detail"
      v-model:open="editOpen"
      mode="edit"
      :node="detail"
      @saved="onNodeSaved"
    />
    <NodeFormDialog
      v-model:open="createOpen"
      mode="create"
      :parent-id="createParentId"
      :parent-title="createParentTitle"
      @saved="onNodeCreated"
    />
    <DeleteNodeDialog
      v-if="deleteTarget"
      v-model:open="deleteOpen"
      :node="deleteTarget"
      @deleted="onNodeDeleted"
    />
    <MoveNodeDialog
      v-if="detail && detail.id !== VIRTUAL_ROOT_ID"
      v-model:open="moveOpen"
      :node="detail"
      @saved="onNodeMoved"
    />
    <OrderDialog
      v-if="detail"
      v-model:open="orderOpen"
      :parent-id="active?.node.id === VIRTUAL_ROOT_ID ? null : detail.id"
      :parent-title="
        active?.node.id === VIRTUAL_ROOT_ID ? '开拓者公会' : detail.title
      "
      :children="detail.children"
      @saved="onNodesOrdered"
    />
  </AppShell>
</template>

<style scoped>
.graph-view {
  position: relative;
  flex: 1 1 auto;
  height: 100%;
  min-height: 0;
  overflow: hidden;
  background: var(--graph-bg);
}

.graph-header {
  position: absolute;
  z-index: 10;
  top: 28px;
  left: 30px;
  max-width: min(420px, calc(100% - 32px));
  color: var(--graph-header-color);
  pointer-events: none;
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
  border: 1px solid var(--graph-title-mark-border);
  border-radius: 8px;
  background: var(--graph-title-mark-bg);
  box-shadow: var(--graph-title-mark-shadow);
  color: var(--graph-title-mark-color);
}

.graph-heading {
  margin: 0;
  font-size: 16px;
  font-weight: 700;
  color: var(--graph-heading-color);
}

.graph-status {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 10px;
  font-size: 12px;
  color: var(--graph-status-color);
}

.graph-dot {
  width: 4px;
  height: 4px;
  border-radius: 999px;
  background: var(--graph-dot-color);
  box-shadow: var(--graph-dot-shadow);
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
  display: flex;
  gap: 6px;
  margin-top: 12px;
  pointer-events: auto;
}

.graph-button {
  display: inline-flex;
  height: 34px;
  align-items: center;
  gap: 6px;
  padding: 0 12px;
  border: 1px solid var(--graph-toolbar-btn-border);
  border-radius: 6px;
  background: var(--graph-toolbar-btn-bg);
  color: var(--graph-toolbar-btn-color);
  font-size: 13px;
  line-height: 1;
  box-shadow: var(--graph-toolbar-btn-shadow);
  backdrop-filter: blur(12px);
  cursor: pointer;
  transition:
    border-color 0.15s,
    color 0.15s,
    background 0.15s;
}

.graph-button:hover {
  border-color: var(--graph-toolbar-btn-hover-border);
  background: var(--graph-toolbar-btn-hover-bg);
  color: var(--graph-toolbar-btn-hover-color);
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
  border: 1px solid var(--graph-toast-border);
  border-radius: 3px;
  background: var(--graph-toast-bg);
  color: var(--graph-toast-color);
  text-align: center;
  font-size: 14px;
  box-shadow: var(--graph-toast-shadow);
}

.graph-error-card p {
  margin: 0;
}

.graph-error-action {
  margin-top: 12px;
  padding: 6px 12px;
  border: 1px solid var(--graph-toast-border);
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
  padding: 8px 14px;
  border: 1px solid var(--graph-toast-border);
  border-radius: 2px;
  background: var(--graph-toast-bg);
  color: var(--graph-toast-color);
  font-size: 13px;
  box-shadow: var(--graph-toast-shadow);
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
}
</style>
