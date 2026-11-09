<script setup lang="ts">
import type { Comment, NodeDetail } from "@/api";
import { VIRTUAL_ROOT_ID } from "@/graph/layout";

interface GraphStageExpose {
  fitView: () => void;
  scrollProgress: (delta: number) => void;
}

const {
  nodes,
  byId,
  roots,
  childrenOf,
  loading,
  error: dataError,
  load,
  reload,
  removeSubtree,
} = useNodeStore();
const { isManager, isAdmin } = useAuth();
const { mode: themeMode } = useTheme();
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
} = useGraphProgress();
const {
  reacting,
  error: reactionError,
  react,
  clearError: clearReactionError,
} = useCommentReactions(updateReactions, "表情操作失败");
const canManageRoots = computed(() => isManager.value || isAdmin.value);
const {
  active,
  detail,
  detailError,
  detailLoading,
  loadDetail,
  refresh: refreshSelection,
  select: selectNode,
  selectedId,
  setDetail,
  clear: clearSelection,
} = useGraphSelection({
  byId,
  canManageRoots,
  childrenOf,
  loadProgress,
  nodes,
  roots,
});
const {
  editOpen,
  createOpen,
  createParent,
  deleteOpen,
  deleteTarget,
  moveOpen,
  orderOpen,
  openEdit,
  openCreate,
  openDelete: openDeleteDialog,
  openMove,
  openOrder,
  closeAll,
} = useNodeDialogs();
const stage = ref<GraphStageExpose | null>(null);
const expandedProgressKey = ref<string | null>(null);

const expandedProgress = computed(() => {
  const key = expandedProgressKey.value;

  return key == null
    ? null
    : (progress.value.find((item) => progressKey(item) === key) ?? null);
});
const selectionLoading = computed(() => {
  const selected = active.value;
  if (!selected) {
    return false;
  }
  const detailPending =
    detailLoading.value ||
    (detail.value?.id !== selected.node.id && !detailError.value);

  return detailPending || progressLoading.value;
});

function onCommentCreated(comment: Comment) {
  addProgressComment(comment);
  clearReactionError();

  if (detail.value?.id === comment.nodeId) {
    setDetail({
      ...detail.value,
      subtreeCommentCount: detail.value.subtreeCommentCount + 1,
    });
  }

  void reload();
}

function reactToComment(commentId: number, emoji: string) {
  void react(commentId, emoji);
}

function openProgress(key: string) {
  expandedProgressKey.value = key;
}

function onDetailUpdated(node: NodeDetail) {
  if (active.value?.node.id === node.id) {
    setDetail(node);
  }
  void reload();
}

function openCreateRoot() {
  openCreate(null);
}

function openCreateChild() {
  if (!detail.value) {
    return;
  }
  openCreate(detail.value);
}

function openDelete() {
  openDeleteDialog(detail.value);
}

async function onNodeSaved(node: NodeDetail) {
  closeAll();

  if (active.value?.node.id === node.id) {
    setDetail(node);
  }
  await reload();
  if (active.value?.node.id === node.id) {
    await loadProgress(node.id);
  }
}

async function onNodeCreated(node: NodeDetail) {
  closeAll();
  await reload();
  selectNode(node.id);
}

async function onNodeMoved(node: NodeDetail) {
  closeAll();
  await reload();

  if (active.value?.node.id === node.id) {
    await refreshSelection();
  }
}

async function onNodeDeleted() {
  const deletedId = deleteTarget.value?.id;
  const parentId = deleteTarget.value?.parentId ?? null;

  closeAll();
  if (deletedId != null) {
    removeSubtree(deletedId);
  }
  await reload();
  selectNode(parentId ?? VIRTUAL_ROOT_ID);
}

async function onNodesOrdered() {
  closeAll();
  await reload();
  await refreshSelection();
}

function onWheel(event: WheelEvent) {
  if (!active.value) {
    return;
  }
  if (
    event.target instanceof Element &&
    event.target.closest(
      "[data-graph-node-panel], [data-graph-comment-composer], [data-graph-progress-detail]",
    )
  ) {
    return;
  }
  event.preventDefault();
  event.stopPropagation();
  stage.value?.scrollProgress(event.deltaY);
}

onMounted(() => {
  void load();
});

watch(selectedId, () => {
  closeAll();
  clearReactionError();
  expandedProgressKey.value = null;
  clearProgress();
});
</script>

<template>
  <AppShell immersive>
    <div class="graph-view" :data-theme="themeMode" @wheel.capture="onWheel">
      <GraphStage
        ref="stage"
        :can-comment="
          !!(
            active &&
            detail?.id === active.node.id &&
            detail.permissions.canComment
          )
        "
        :focus="active"
        :nodes
        :progress
        :progress-loading
        :theme="themeMode"
        @blank="clearSelection"
        @comment-created="onCommentCreated"
        @progress-edge="loadOlderProgress"
        @progress-open="openProgress"
        @reaction="reactToComment"
        @select="selectNode"
      />

      <Transition name="network-loading">
        <div
          v-if="active && selectionLoading"
          aria-live="polite"
          class="graph-network-loading"
          role="status"
        >
          <div class="network-loading-card">
            <span aria-hidden="true" class="network-loading-sigil">
              <span />
            </span>
            <p>正在加载节点…</p>
          </div>
        </div>
      </Transition>

      <Transition name="progress-detail">
        <GraphProgressDetail
          v-if="expandedProgress"
          :busy="
            expandedProgress.kind === 'comment' &&
            reacting.has(expandedProgress.comment.id)
          "
          :item="expandedProgress"
          :node-title="active?.node.title ?? ''"
          @close="expandedProgressKey = null"
          @reaction="reactToComment"
        />
      </Transition>

      <Transition name="node-panel">
        <GraphNodePanel
          v-if="active"
          :detail="detail?.id === active.node.id ? detail : null"
          :error="detailError"
          :is-virtual-root="active.node.id === VIRTUAL_ROOT_ID"
          :loading="detailLoading"
          :parent-id="active.parent?.id ?? null"
          :parent-title="active.parent?.title ?? null"
          :selected="active.node"
          @close="clearSelection"
          @create-child="openCreateChild"
          @create-root="openCreateRoot"
          @edit="openEdit"
          @move="openMove"
          @navigate="selectNode"
          @remove="openDelete"
          @reorder="openOrder"
          @retry="loadDetail"
          @updated="onDetailUpdated"
        />
      </Transition>

      <header class="graph-header">
        <div class="graph-title">
          <span class="graph-title-mark"><Network :size="16" /></span>
          <h1 class="graph-heading">关系图谱</h1>
        </div>
        <div class="graph-status">
          <span>{{ nodes.length }} 个节点</span>
          <template v-if="loading || active">
            <span class="graph-dot" />
            <span v-if="loading" class="graph-inline">
              <Loader2 class="graph-spin" :size="12" />
              正在加载
            </span>
            <span v-else-if="selectionLoading" class="graph-inline">
              <Loader2 class="graph-spin" :size="12" />
              正在加载进展
            </span>
            <span v-else>
              {{ progress.length }}{{ progressHaveMore ? "+" : "" }} 条进展
            </span>
          </template>
        </div>
      </header>

      <div v-if="!active" class="graph-toolbar">
        <button class="graph-button" type="button" @click="stage?.fitView()">
          <Focus aria-hidden="true" :size="14" />
          显示全部节点
        </button>
      </div>

      <div v-if="dataError" class="graph-error-overlay">
        <div class="graph-error-card">
          <p>{{ dataError }}</p>
          <button class="graph-error-action" type="button" @click="reload">
            重新加载图谱
          </button>
        </div>
      </div>

      <p v-if="progressError || reactionError" class="graph-toast" role="alert">
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
      :parent-id="createParent?.id ?? null"
      :parent-title="createParent?.title ?? ''"
      @saved="onNodeCreated"
    />
    <DeleteNodeDialog
      v-if="deleteTarget"
      v-model:open="deleteOpen"
      :node="deleteTarget"
      @close="closeAll"
      @deleted="onNodeDeleted"
    />
    <MoveNodeDialog
      v-if="detail && detail.id !== VIRTUAL_ROOT_ID"
      :node="detail"
      :open="moveOpen"
      @close="closeAll"
      @saved="onNodeMoved"
    />
    <OrderDialog
      v-if="detail"
      :children="detail.children"
      :open="orderOpen"
      :parent-id="active?.node.id === VIRTUAL_ROOT_ID ? null : detail.id"
      :parent-title="
        active?.node.id === VIRTUAL_ROOT_ID ? '开拓者公会' : detail.title
      "
      @close="closeAll"
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
.graph-toolbar {
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
  top: 28px;
  right: 28px;
  display: flex;
  gap: 6px;
  pointer-events: auto;
}

.graph-button {
  display: inline-flex;
  height: 34px;
  align-items: center;
  gap: 6px;
  padding: 0 12px;
  border: 1px solid rgba(221, 176, 83, 0.42);
  border-radius: 3px;
  background: rgba(8, 14, 17, 0.78);
  color: #e6c37b;
  font-size: 13px;
  line-height: 1;
  box-shadow:
    inset 0 0 16px rgba(225, 173, 70, 0.04),
    0 8px 24px rgba(0, 0, 0, 0.2);
  backdrop-filter: blur(12px);
  cursor: pointer;
  transition:
    border-color 0.15s,
    color 0.15s,
    background 0.15s;
}

.graph-button:hover {
  border-color: rgba(242, 190, 83, 0.8);
  background: rgba(25, 27, 20, 0.9);
  color: #fff3d1;
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
  content: "";
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
  padding: 8px 14px;
  border: 1px solid rgba(240, 182, 82, 0.3);
  border-radius: 2px;
  background: rgba(8, 14, 17, 0.94);
  color: #f2c66f;
  font-size: 13px;
  box-shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.2);
}

:global([data-theme="light"] .graph-view) {
  background: #eef3f5;
}

:global([data-theme="light"] .graph-header) {
  color: #526572;
}

:global([data-theme="light"] .graph-title-mark) {
  border-color: rgba(163, 104, 17, 0.62);
  background:
    linear-gradient(135deg, rgba(192, 133, 40, 0.15), transparent 62%),
    rgba(255, 255, 255, 0.88);
  box-shadow:
    inset 0 0 18px rgba(173, 113, 22, 0.06),
    0 0 24px rgba(173, 113, 22, 0.08);
  color: #9d6411;
}

:global([data-theme="light"] .graph-heading) {
  color: #1c2932;
}

:global([data-theme="light"] .graph-status) {
  color: #637784;
}

:global([data-theme="light"] .graph-dot) {
  background: #a96c12;
  box-shadow: 0 0 8px rgba(169, 108, 18, 0.45);
}

:global([data-theme="light"] .graph-button) {
  border-color: rgba(159, 103, 18, 0.32);
  background: rgba(255, 255, 255, 0.8);
  color: #8c5b12;
  box-shadow:
    inset 0 0 16px rgba(225, 173, 70, 0.04),
    0 8px 24px rgba(40, 52, 61, 0.1);
}

:global([data-theme="light"] .graph-button:hover) {
  border-color: rgba(142, 89, 10, 0.62);
  background: rgba(255, 250, 239, 0.96);
  color: #734707;
}

:global([data-theme="light"] .network-loading-card) {
  border-color: rgba(158, 102, 18, 0.28);
  background: rgba(255, 255, 255, 0.9);
  color: #81520d;
  box-shadow:
    0 20px 50px rgba(48, 63, 72, 0.16),
    inset 0 0 24px rgba(181, 126, 37, 0.03);
}

:global([data-theme="light"] .graph-network-loading) {
  background: radial-gradient(
    circle at 50% 46%,
    rgba(238, 244, 246, 0.3),
    rgba(232, 238, 241, 0.12) 36%,
    transparent 62%
  );
}

:global([data-theme="light"] .graph-error-card),
:global([data-theme="light"] .graph-toast) {
  border-color: rgba(157, 100, 18, 0.28);
  background: rgba(255, 255, 255, 0.95);
  color: #85570f;
  box-shadow: 0 20px 50px rgba(47, 61, 70, 0.16);
}

:global([data-theme="light"] .graph-error-action) {
  border-color: rgba(157, 100, 18, 0.3);
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
}
</style>
