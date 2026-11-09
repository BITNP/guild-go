import type { NodeDetail } from "@/api";

export function useNodeInlineEditing(
  node: MaybeRefOrGetter<NodeDetail | null>,
  onUpdated: (node: NodeDetail) => void,
) {
  const editingDue = ref(false);
  const dueValue = ref<number | null>(null);
  const savingDue = ref(false);
  const dueError = ref<string | null>(null);
  const savingCompletion = ref(false);
  const completionError = ref<string | null>(null);

  const busy = computed(() => savingDue.value || savingCompletion.value);

  function resetEditing() {
    if (busy.value) {
      return;
    }

    editingDue.value = false;
    dueError.value = null;
    completionError.value = null;
  }

  function startDueEdit() {
    const current = toValue(node);
    if (!current || savingCompletion.value) {
      return;
    }

    dueValue.value = current.dueAt ? new Date(current.dueAt).getTime() : null;
    dueError.value = null;
    editingDue.value = true;
  }

  function cancelDueEdit() {
    if (savingDue.value) {
      return;
    }

    editingDue.value = false;
    dueError.value = null;
  }

  async function saveDue() {
    const current = toValue(node);
    if (!current || savingDue.value) {
      return;
    }

    savingDue.value = true;
    dueError.value = null;

    try {
      const updated = await updateNode(current.id, {
        title: current.title,
        description: current.description,
        dueAt:
          dueValue.value === null
            ? null
            : new Date(dueValue.value).toISOString(),
        tags: current.tags,
        completed: !!current.completedAt,
      });

      editingDue.value = false;
      onUpdated(updated);
    } catch (err) {
      dueError.value = errorMessage(err, "保存到期时间失败");
    } finally {
      savingDue.value = false;
    }
  }

  async function toggleCompletion() {
    const current = toValue(node);
    if (
      !current ||
      !current.permissions.canEdit ||
      busy.value ||
      editingDue.value
    ) {
      return;
    }

    savingCompletion.value = true;
    completionError.value = null;

    try {
      const updated = await updateNode(current.id, {
        title: current.title,
        description: current.description,
        dueAt: current.dueAt,
        tags: current.tags,
        completed: !current.completedAt,
      });

      onUpdated(updated);
    } catch (err) {
      completionError.value = errorMessage(err, "更新完成状态失败");
    } finally {
      savingCompletion.value = false;
    }
  }

  return {
    editingDue,
    dueValue,
    savingDue,
    dueError,
    savingCompletion,
    completionError,
    startDueEdit,
    cancelDueEdit,
    saveDue,
    toggleCompletion,
    resetEditing,
  };
}
