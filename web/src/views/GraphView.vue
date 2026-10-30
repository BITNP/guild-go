<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { Focus, Loader2, MousePointer2, Network } from '@lucide/vue'
import AppShell from '@/components/AppShell.vue'
import DeleteNodeDialog from '@/components/DeleteNodeDialog.vue'
import GraphNodePanel from '@/components/graph/GraphNodePanel.vue'
import GraphProgressDetail from '@/components/graph/GraphProgressDetail.vue'
import GraphStage from '@/components/graph/GraphStage.vue'
import NodeFormDialog from '@/components/NodeFormDialog.vue'
import OrderDialog from '@/components/OrderDialog.vue'
import { useAuth } from '@/composables/useAuth'
import { useGraphProgress } from '@/composables/useGraphProgress'
import { useNodeStore } from '@/composables/useNodeStore'
import { VIRTUAL_ROOT_ID } from '@/graph/layout'
import type { GraphNeighborhood, GraphNode } from '@/graph/types'
import {
  fetchNode,
  toggleReaction,
  type ChildNode,
  type Comment,
  type NodeDetail,
  type ProgressItem,
} from '@/lib/api'
import { errorMessage } from '@/lib/utils'

interface GraphStageExpose {
  fitView: () => void
  scrollProgress: (delta: number) => void
}

const {
  nodes,
  byId,
  childrenOf,
  loading,
  error: dataError,
  load,
  reload,
  removeSubtree,
} = useNodeStore()
const { isManager, isAdmin } = useAuth()
const {
  items: progress,
  loading: progressLoading,
  hasMore: progressHaveMore,
  error: progressError,
  load: loadProgress,
  loadOlder: loadOlderProgress,
  add: addProgressComment,
  updateReactions,
  clear: clearProgress,
} = useGraphProgress()
const stage = ref<GraphStageExpose | null>(null)
const active = ref<GraphNeighborhood | null>(null)
const detail = ref<NodeDetail | null>(null)
const detailLoading = ref(false)
const detailError = ref<string | null>(null)
const reactionError = ref<string | null>(null)
const reacting = ref(new Set<number>())
const expandedProgressKey = ref<string | null>(null)
const editOpen = ref(false)
const createOpen = ref(false)
const createParentId = ref<number | null>(null)
const createParentTitle = ref('')
const deleteOpen = ref(false)
const deleteTarget = ref<NodeDetail | null>(null)
const orderOpen = ref(false)
let detailGeneration = 0
const expandedProgress = computed(() => {
  const key = expandedProgressKey.value
  return key == null
    ? null
    : progress.value.find((item) => progressKey(item) === key) ?? null
})
const selectionLoading = computed(() => {
  const selected = active.value
  if (!selected) return false
  const detailPending = detailLoading.value ||
    (detail.value?.id !== selected.node.id && !detailError.value)
  return detailPending || progressLoading.value
})

function progressKey(item: ProgressItem): string {
  return item.kind === 'comment'
    ? `comment:${item.comment.id}`
    : `description_change:${item.descriptionChange.id}`
}

const rootNodes = computed(() =>
  nodes.value
    .filter((node) => node.parentId == null)
    .sort((left, right) => left.sortOrder - right.sortOrder || left.id - right.id),
)
const canManageRoots = computed(() => isManager.value || isAdmin.value)

function virtualRootNode(): GraphNode {
  return {
    id: VIRTUAL_ROOT_ID,
    parentId: null,
    title: '开拓者公会',
    sortOrder: -1,
    completedAt: null,
    childCount: rootNodes.value.length,
    commentCount: rootNodes.value.reduce(
      (total, node) => total + node.commentCount,
      0,
    ),
  }
}

function virtualRootDetail(): NodeDetail | null {
  if (active.value?.node.id !== VIRTUAL_ROOT_ID) return null
  return {
    id: VIRTUAL_ROOT_ID,
    parentId: null,
    title: '开拓者公会',
    description: '',
    dueAt: null,
    completedAt: null,
    tags: [],
    sortOrder: -1,
    createdBySubject: '',
    createdBy: {
      subject: '',
      userName: '',
      avatar: null,
      positions: [],
    },
    createdAt: '',
    updatedAt: '',
    breadcrumb: [],
    directLeaders: [],
    directMembers: [],
    inheritedLeaderCount: 0,
    inheritedMemberCount: 0,
    effectiveMemberCount: 0,
    subtreeNodeCount: nodes.value.length,
    subtreeCommentCount: nodes.value.reduce(
      (total, node) => total + node.commentCount,
      0,
    ),
    subtreeAttachmentCount: 0,
    permissions: {
      canEdit: false,
      canDelete: false,
      canComment: false,
      canManageMembers: false,
      canAddChild: canManageRoots.value,
      canReorderChildren: canManageRoots.value && rootNodes.value.length > 1,
    },
    children: rootNodes.value.map((node): ChildNode => ({
      ...node,
      latestComment: null,
    })),
  }
}

// The whole tree is already in the node store, so the focused neighborhood is a
// pure lookup. childrenOf keeps the same (sortOrder, id) order as the server.
function selectNode(id: number) {
  if (id === VIRTUAL_ROOT_ID) {
    const changed = active.value?.node.id !== VIRTUAL_ROOT_ID
    if (changed) {
      detail.value = null
      detailError.value = null
      detailLoading.value = false
      reactionError.value = null
      expandedProgressKey.value = null
      clearProgress()
    }
    active.value = {
      node: virtualRootNode(),
      parent: null,
      children: rootNodes.value,
    }
    detail.value = virtualRootDetail()
    return
  }
  const node = byId.value.get(id)
  if (!node) return
  const changed = active.value?.node.id !== id
  if (changed) {
    detailGeneration++
    detail.value = null
    detailError.value = null
    detailLoading.value = true
    reactionError.value = null
    expandedProgressKey.value = null
    clearProgress()
  }
  const parent = node.parentId == null
    ? virtualRootNode()
    : byId.value.get(node.parentId) ?? null
  active.value = { node, parent, children: childrenOf(id) }
}

function clearSelection() {
  active.value = null
  reactionError.value = null
  expandedProgressKey.value = null
}

function onCommentCreated(comment: Comment) {
  addProgressComment(comment)
  reactionError.value = null
  if (detail.value?.id === comment.nodeId) {
    detail.value = {
      ...detail.value,
      subtreeCommentCount: detail.value.subtreeCommentCount + 1,
    }
  }
  if (active.value?.node.id === comment.nodeId) {
    active.value = {
      ...active.value,
      node: {
        ...active.value.node,
        commentCount: active.value.node.commentCount + 1,
      },
    }
  }
}

async function reactToComment(commentId: number, emoji: string) {
  if (reacting.value.has(commentId)) return
  reacting.value = new Set([...reacting.value, commentId])
  reactionError.value = null
  try {
    const result = await toggleReaction(commentId, emoji)
    updateReactions(commentId, result.reactions)
  } catch (err) {
    reactionError.value = errorMessage(err, '表情操作失败')
  } finally {
    const next = new Set(reacting.value)
    next.delete(commentId)
    reacting.value = next
  }
}

async function loadDetail() {
  const id = active.value?.node.id
  if (id == null || id === VIRTUAL_ROOT_ID) return
  const current = ++detailGeneration
  detailLoading.value = true
  detailError.value = null
  try {
    const node = await fetchNode(id)
    if (current === detailGeneration && active.value?.node.id === id) {
      detail.value = node
      detailLoading.value = false
      await loadProgress(id)
    }
  } catch (err) {
    if (current === detailGeneration && active.value?.node.id === id) {
      detailError.value = errorMessage(err, '无法加载节点详情')
    }
  } finally {
    if (current === detailGeneration) detailLoading.value = false
  }
}

function openProgress(key: string) {
  expandedProgressKey.value = key
}

function onDetailUpdated(node: NodeDetail) {
  if (active.value?.node.id === node.id) {
    detail.value = node
    active.value = {
      ...active.value,
      node: {
        ...active.value.node,
        title: node.title,
        parentId: node.parentId,
        completedAt: node.completedAt,
      },
    }
  }
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
  if (active.value?.node.id === node.id) {
    detail.value = node
    active.value = {
      ...active.value,
      node: {
        ...active.value.node,
        title: node.title,
        parentId: node.parentId,
        completedAt: node.completedAt,
      },
    }
    await reload()
    if (active.value?.node.id === node.id) {
      await loadProgress(node.id)
    }
  }
}

async function onNodeCreated(node: NodeDetail) {
  createOpen.value = false
  await reload()
  selectNode(node.id)
}

async function onNodeDeleted() {
  const deletedId = deleteTarget.value?.id
  const parentId = deleteTarget.value?.parentId ?? null
  deleteOpen.value = false
  deleteTarget.value = null
  if (deletedId != null) removeSubtree(deletedId)
  active.value = null
  detail.value = null
  detailGeneration++
  clearProgress()
  expandedProgressKey.value = null
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

function onWheel(event: WheelEvent) {
  if (!active.value) return
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

onMounted(() => {
  void load()
})

watch(() => active.value?.node.id ?? null, (id) => {
  editOpen.value = false
  createOpen.value = false
  deleteOpen.value = false
  orderOpen.value = false
  deleteTarget.value = null
  detailGeneration++
  detail.value = null
  detailError.value = null
  detailLoading.value = id != null
  expandedProgressKey.value = null
  if (id == null) {
    clearProgress()
    return
  }
  if (id === VIRTUAL_ROOT_ID) {
    detailLoading.value = false
    detail.value = virtualRootDetail()
    clearProgress()
    return
  }
  void loadDetail()
})
</script>

<template>
  <AppShell immersive>
    <div class="graph-view" @wheel.capture="onWheel">
      <GraphStage
        ref="stage"
        :nodes="nodes"
        :focus="active"
        :progress="progress"
        :progress-loading="progressLoading"
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

      <Transition name="network-loading">
        <div
          v-if="active && selectionLoading"
          class="graph-network-loading"
          role="status"
          aria-live="polite"
        >
          <div class="network-loading-card">
            <span class="network-loading-sigil" aria-hidden="true">
              <span />
            </span>
            <p>正在连接到开拓者网络……</p>
          </div>
        </div>
      </Transition>

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
          @remove="openDelete"
          @create-child="openCreateChild"
          @create-root="openCreateRoot"
          @reorder="orderOpen = true"
        />
      </Transition>

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
          <span v-else-if="active && selectionLoading" class="graph-inline">
            <Loader2 :size="10" class="graph-spin" /> 正在连接开拓者网络
          </span>
          <span v-else-if="active">
            {{ progress.length }}{{ progressHaveMore ? '+' : '' }} 条进展
          </span>
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
          <span v-else>滚轮翻阅进展 · 点击进展展开 · 点击底部表情回应</span>
        </div>
        <p class="graph-help-note">同层同环，子节点沿父节点扇区向外展开；聚焦时从外圈回望公会核心。</p>
      </aside>

      <div v-if="dataError" class="graph-error-overlay">
        <div class="graph-error-card">
          <p>{{ dataError }}</p>
          <button type="button" class="graph-error-action" @click="reload">重新加载图谱</button>
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
      @close="deleteOpen = false"
      @deleted="onNodeDeleted"
    />
    <OrderDialog
      v-if="detail"
      :open="orderOpen"
      :parent-id="active?.node.id === VIRTUAL_ROOT_ID ? null : detail.id"
      :parent-title="active?.node.id === VIRTUAL_ROOT_ID ? '开拓者公会' : detail.title"
      :children="detail.children"
      @close="orderOpen = false"
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

.graph-network-loading {
  position: absolute;
  inset: 0;
  z-index: 15;
  display: grid;
  place-items: center;
  pointer-events: none;
  background: radial-gradient(
    circle at 50% 46%,
    rgba(6, 17, 20, 0.3),
    rgba(4, 8, 10, 0.12) 36%,
    transparent 62%
  );
}

.network-loading-card {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 13px 18px;
  border: 1px solid rgba(230, 178, 73, 0.38);
  border-radius: 2px;
  background: rgba(6, 13, 16, 0.9);
  color: #e9ce91;
  font-size: 13px;
  letter-spacing: 0.05em;
  box-shadow:
    0 20px 50px rgba(0, 0, 0, 0.38),
    inset 0 0 24px rgba(229, 174, 58, 0.04);
  backdrop-filter: blur(14px);
}

.network-loading-card p {
  margin: 0;
}

.network-loading-sigil {
  position: relative;
  width: 28px;
  height: 28px;
  flex: none;
  border: 1px solid rgba(94, 206, 188, 0.28);
  border-top-color: #64d6be;
  border-radius: 50%;
  animation: network-loading-spin 1.1s linear infinite;
}

.network-loading-sigil::before {
  position: absolute;
  inset: 5px;
  border: 1px solid rgba(235, 184, 74, 0.42);
  border-right-color: #e8b954;
  border-radius: 50%;
  content: '';
  animation: network-loading-spin 0.7s linear infinite reverse;
}

.network-loading-sigil span {
  position: absolute;
  inset: 11px;
  border-radius: 50%;
  background: #f0c263;
  box-shadow: 0 0 12px rgba(240, 194, 99, 0.86);
  animation: network-loading-pulse 1.2s ease-in-out infinite;
}

@keyframes network-loading-spin {
  to {
    transform: rotate(360deg);
  }
}

@keyframes network-loading-pulse {
  50% {
    opacity: 0.45;
    transform: scale(0.72);
  }
}

.network-loading-enter-active,
.network-loading-leave-active {
  transition: opacity 180ms ease;
}

.network-loading-enter-from,
.network-loading-leave-to {
  opacity: 0;
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

  .network-loading-sigil,
  .network-loading-sigil::before,
  .network-loading-sigil span {
    animation: none;
  }

  .network-loading-enter-active,
  .network-loading-leave-active {
    transition: none;
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
