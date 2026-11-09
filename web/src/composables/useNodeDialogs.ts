interface CreateParent {
  id: number | null;
  title: string;
}

export function useNodeDialogs() {
  const editOpen = ref(false);
  const createOpen = ref(false);
  const createParent = ref<CreateParent | null>(null);
  const deleteOpen = ref(false);
  const deleteTarget = ref<NodeDetail | null>(null);
  const moveOpen = ref(false);
  const orderOpen = ref(false);
  const membersOpen = ref(false);

  function openEdit() {
    editOpen.value = true;
  }

  function openCreate(parent: NodeDetail | null) {
    createParent.value = parent
      ? { id: parent.id, title: parent.title }
      : { id: null, title: "" };
    createOpen.value = true;
  }

  function openDelete(node: NodeDetail | null) {
    if (!node?.permissions.canDelete) {
      return;
    }

    deleteTarget.value = node;
    deleteOpen.value = true;
  }

  function openMove() {
    moveOpen.value = true;
  }

  function openOrder() {
    orderOpen.value = true;
  }

  function openMembers() {
    membersOpen.value = true;
  }

  function closeAll() {
    editOpen.value = false;
    createOpen.value = false;
    createParent.value = null;
    deleteOpen.value = false;
    deleteTarget.value = null;
    moveOpen.value = false;
    orderOpen.value = false;
    membersOpen.value = false;
  }

  return {
    editOpen,
    createOpen,
    createParent,
    deleteOpen,
    deleteTarget,
    moveOpen,
    orderOpen,
    membersOpen,
    openEdit,
    openCreate,
    openDelete,
    openMove,
    openOrder,
    openMembers,
    closeAll,
  };
}
