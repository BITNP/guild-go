import type { Comment } from "@/api";
import { VIRTUAL_ROOT_ID } from "@/graph/layout";

export function useGraphView() {
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
  const progress = useGraphProgress();
  const reactions = useCommentReactions(
    progress.updateReactions,
    "表情操作失败",
  );
  const canManageRoots = computed(() => isManager.value || isAdmin.value);
  const selection = useGraphSelection({
    byId,
    canManageRoots,
    childrenOf,
    loadProgress: progress.load,
    nodes,
    roots,
  });
  const dialogs = useNodeDialogs();
  const expandedProgressKey = ref<string | null>(null);
  const expandedProgress = computed(() => {
    const key = expandedProgressKey.value;

    return key == null
      ? null
      : (progress.items.value.find((item) => progressKey(item) === key) ??
          null);
  });
  const selectionLoading = computed(() => {
    const selected = selection.active.value;
    if (!selected) {
      return false;
    }
    const detailPending =
      selection.detailLoading.value ||
      (selection.detail.value?.id !== selected.node.id &&
        !selection.detailError.value);

    return detailPending || progress.loading.value;
  });

  function onCommentCreated(comment: Comment) {
    progress.add(comment);
    reactions.clearError();

    if (selection.detail.value?.id === comment.nodeId) {
      selection.setDetail({
        ...selection.detail.value,
        subtreeCommentCount: selection.detail.value.subtreeCommentCount + 1,
      });
    }

    void reload();
  }

  function reactToComment(commentId: number, emoji: string) {
    void reactions.react(commentId, emoji);
  }

  function openProgress(key: string) {
    expandedProgressKey.value = key;
  }

  function onDetailUpdated(node: NodeDetail) {
    if (selection.active.value?.node.id === node.id) {
      selection.setDetail(node);
    }
    void reload();
  }

  function openCreateRoot() {
    dialogs.openCreate(null);
  }

  function openCreateChild() {
    if (!selection.detail.value) {
      return;
    }
    dialogs.openCreate(selection.detail.value);
  }

  function openDelete() {
    dialogs.openDelete(selection.detail.value);
  }

  async function onNodeSaved(node: NodeDetail) {
    dialogs.closeAll();

    if (selection.active.value?.node.id === node.id) {
      selection.setDetail(node);
    }
    await reload();
    if (selection.active.value?.node.id === node.id) {
      await progress.load(node.id);
    }
  }

  async function onNodeCreated(node: NodeDetail) {
    dialogs.closeAll();
    await reload();
    selection.select(node.id);
  }

  async function onNodeMoved(node: NodeDetail) {
    dialogs.closeAll();
    await reload();

    if (selection.active.value?.node.id === node.id) {
      await selection.refresh();
    }
  }

  async function onNodeDeleted() {
    const deletedId = dialogs.deleteTarget.value?.id;
    const parentId = dialogs.deleteTarget.value?.parentId ?? null;

    dialogs.closeAll();
    if (deletedId != null) {
      removeSubtree(deletedId);
    }
    await reload();
    selection.select(parentId ?? VIRTUAL_ROOT_ID);
  }

  async function onNodesOrdered() {
    dialogs.closeAll();
    await reload();
    await selection.refresh();
  }

  watch(selection.selectedId, () => {
    dialogs.closeAll();
    reactions.clearError();
    expandedProgressKey.value = null;
    progress.clear();
  });

  onMounted(() => {
    void load();
  });

  return {
    nodes,
    loading,
    dataError,
    reload,
    themeMode,
    active: selection.active,
    detail: selection.detail,
    detailError: selection.detailError,
    detailLoading: selection.detailLoading,
    loadDetail: selection.loadDetail,
    selectNode: selection.select,
    clearSelection: selection.clear,
    progress: progress.items,
    progressLoading: progress.loading,
    progressHaveMore: progress.hasMore,
    progressError: progress.error,
    loadOlderProgress: progress.loadOlder,
    reacting: reactions.reacting,
    reactionError: reactions.error,
    editOpen: dialogs.editOpen,
    createOpen: dialogs.createOpen,
    createParent: dialogs.createParent,
    deleteOpen: dialogs.deleteOpen,
    deleteTarget: dialogs.deleteTarget,
    moveOpen: dialogs.moveOpen,
    orderOpen: dialogs.orderOpen,
    openEdit: dialogs.openEdit,
    openCreate: dialogs.openCreate,
    openDeleteDialog: dialogs.openDelete,
    openMove: dialogs.openMove,
    openOrder: dialogs.openOrder,
    closeAll: dialogs.closeAll,
    expandedProgress,
    expandedProgressKey,
    selectionLoading,
    onCommentCreated,
    reactToComment,
    openProgress,
    onDetailUpdated,
    openCreateRoot,
    openCreateChild,
    openDelete,
    onNodeSaved,
    onNodeCreated,
    onNodeMoved,
    onNodeDeleted,
    onNodesOrdered,
  };
}
