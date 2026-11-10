export function useNodePage() {
  const route = useRoute();
  const router = useRouter();
  const { reload: reloadNodes } = useNodeStore();
  const nodeId = computed(() => Number(route.params.id));
  const {
    detail,
    loading,
    error,
    load: loadDetail,
    setDetail,
    setError,
  } = useNodeDetailState(true);
  const dialogs = useNodeDialogs();
  const {
    savingCompletion: toggling,
    completionError: actionError,
    toggleCompletion,
  } = useNodeInlineEditing(
    () => detail.value,
    (node) => {
      if (node.id === nodeId.value) {
        setDetail(node);
      }
      void reloadNodes();
    },
  );

  async function load(silent = false) {
    const requested = nodeId.value;

    if (!Number.isSafeInteger(requested) || requested <= 0) {
      setError("节点不存在");

      return;
    }

    actionError.value = null;

    await loadDetail(requested, {
      silent,
      errorFallback: "无法加载节点",
      shouldApply: () => requested === nodeId.value,
    });
  }

  function onSaved(node: NodeDetail) {
    if (node.id === nodeId.value) {
      setDetail(node);
    }
    dialogs.closeAll();
    void reloadNodes();
  }

  async function onCreated(node: NodeDetail) {
    dialogs.closeAll();
    await reloadNodes();
    await router.push({ name: "node", params: { id: node.id } });
  }

  async function onOrdered() {
    dialogs.closeAll();
    await Promise.all([load(true), reloadNodes()]);
  }

  async function onDeleted() {
    const parentId = detail.value?.parentId;
    dialogs.closeAll();
    await reloadNodes();
    await router.push(
      parentId ? { name: "node", params: { id: parentId } } : { name: "home" },
    );
  }

  async function onCommentChanged() {
    await Promise.all([load(true), reloadNodes()]);
  }

  function goHome() {
    void router.push({ name: "home" });
  }

  watch(
    nodeId,
    () => {
      dialogs.closeAll();
      void load();
    },
    { immediate: true },
  );

  return {
    detail,
    loading,
    error,
    actionError,
    toggling,
    editOpen: dialogs.editOpen,
    createOpen: dialogs.createOpen,
    membersOpen: dialogs.membersOpen,
    moveOpen: dialogs.moveOpen,
    orderOpen: dialogs.orderOpen,
    deleteOpen: dialogs.deleteOpen,
    openEdit: dialogs.openEdit,
    openCreate: dialogs.openCreate,
    openDelete: dialogs.openDelete,
    openMove: dialogs.openMove,
    openOrder: dialogs.openOrder,
    openMembers: dialogs.openMembers,
    closeAll: dialogs.closeAll,
    load,
    toggleCompletion,
    onSaved,
    onCreated,
    onOrdered,
    onDeleted,
    onCommentChanged,
    goHome,
  };
}
