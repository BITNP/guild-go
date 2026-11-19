<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { Focus, Loader2, Network } from '@lucide/vue'
import AppShell from '@/components/AppShell.vue'
import DeleteNodeDialog from '@/components/DeleteNodeDialog.vue'
import GraphNodePanel from '@/components/graph/GraphNodePanel.vue'
import GraphProgressDetail from '@/components/graph/GraphProgressDetail.vue'
import GraphStage from '@/components/graph/GraphStage.vue'
import NodeFormDialog from '@/components/NodeFormDialog.vue'
import OrderDialog from '@/components/OrderDialog.vue'
import MoveNodeDialog from '@/components/MoveNodeDialog.vue'
import { useAuth } from '@/composables/useAuth'
import { useGraphProgress } from '@/composables/useGraphProgress'
import { useNodeStore } from '@/composables/useNodeStore'
import { useTheme } from '@/composables/useTheme'
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
const { mode: themeMode } = useTheme()
const {
  items: progress,
  loading: progressLoading,
  hasMore: progressHaveMore,
  error: progressError,
  load: loadProgress,
  loadOlder: loadOlderProgress,
  add: addProgressComment,
  updateReactions,
  removeProgress,
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
const moveOpen = ref(false)
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

async function onNodeMoved(node: NodeDetail) {
  moveOpen.value = false
  await reload()
  if (active.value?.node.id === node.id) {
    selectNode(node.id)
    detail.value = node
    await loadProgress(node.id)
  }
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

function onProgressCommentDeleted(commentId: number) {
  removeProgress(`comment:${commentId}`)
  expandedProgressKey.value = null
  if (detail.value) {
    detail.value = {
      ...detail.value,
      subtreeCommentCount: Math.max(0, detail.value.subtreeCommentCount - 1),
    }
  }
  if (active.value) {
    active.value = {
      ...active.value,
      node: {
        ...active.value.node,
        commentCount: Math.max(0, active.value.node.commentCount - 1),
      },
    }
  }
  void reload()
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

watch(() => active.value?.node.id ?? null, (id) => {
  editOpen.value = false
  createOpen.value = false
  deleteOpen.value = false
  orderOpen.value = false
  deleteTarget.value = null
  moveOpen.value = false
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
    <div
      class="graph-view"
      :data-theme="themeMode"
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
          <template v-if="loading || (active && active.node.id !== VIRTUAL_ROOT_ID)">
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
    <MoveNodeDialog
      v-if="detail && detail.id !== VIRTUAL_ROOT_ID"
      :open="moveOpen"
      :node="detail"
      @close="moveOpen = false"
      @saved="onNodeMoved"
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

.graph-header {
  position: absolute;
  z-index: 10;
  pointer-events: none;
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
  border-radius: 8px;
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
  font-size: 16px;
  font-weight: 700;
  color: #f1ede2;
}

.graph-status {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 10px;
  font-size: 12px;
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
  border: 1px solid rgba(221, 176, 83, 0.42);
  border-radius: 6px;
  background: rgba(8, 14, 17, 0.78);
  color: #e6c37b;
  font-size: 13px;
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
  padding: 8px 14px;
  border: 1px solid rgba(240, 182, 82, 0.3);
  border-radius: 2px;
  background: rgba(8, 14, 17, 0.94);
  color: #f2c66f;
  font-size: 13px;
  box-shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.2);
}

:global([data-theme='light'] .graph-view) {
  background: #eef3f5;
}

:global([data-theme='light'] .graph-header) {
  color: #526572;
}

:global([data-theme='light'] .graph-title-mark) {
  border-color: rgba(163, 104, 17, 0.62);
  background:
    linear-gradient(135deg, rgba(192, 133, 40, 0.15), transparent 62%),
    rgba(255, 255, 255, 0.88);
  box-shadow:
    inset 0 0 18px rgba(173, 113, 22, 0.06),
    0 0 24px rgba(173, 113, 22, 0.08);
  color: #9d6411;
}

:global([data-theme='light'] .graph-heading) {
  color: #1c2932;
}

:global([data-theme='light'] .graph-status) {
  color: #637784;
}

:global([data-theme='light'] .graph-dot) {
  background: #a96c12;
  box-shadow: 0 0 8px rgba(169, 108, 18, 0.45);
}

:global([data-theme='light'] .graph-button) {
  border-color: rgba(159, 103, 18, 0.32);
  background: rgba(255, 255, 255, 0.8);
  color: #8c5b12;
  box-shadow:
    inset 0 0 16px rgba(225, 173, 70, 0.04),
    0 8px 24px rgba(40, 52, 61, 0.1);
}

:global([data-theme='light'] .graph-button:hover) {
  border-color: rgba(142, 89, 10, 0.62);
  background: rgba(255, 250, 239, 0.96);
  color: #734707;
}

:global([data-theme='light'] .graph-error-card),
:global([data-theme='light'] .graph-toast) {
  border-color: rgba(157, 100, 18, 0.28);
  background: rgba(255, 255, 255, 0.95);
  color: #85570f;
  box-shadow: 0 20px 50px rgba(47, 61, 70, 0.16);
}

:global([data-theme='light'] .graph-error-action) {
  border-color: rgba(157, 100, 18, 0.3);
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
