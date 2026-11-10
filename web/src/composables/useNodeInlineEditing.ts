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
  let generation = 0;

  watch(
    () => toValue(node)?.id,
    () => {
      generation++;
      editingDue.value = false;
      dueValue.value = null;
      savingDue.value = false;
      dueError.value = null;
      savingCompletion.value = false;
      completionError.value = null;
    },
    { flush: "sync" },
  );

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
    const currentGeneration = generation;

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

      if (currentGeneration === generation) {
        editingDue.value = false;
      }
      onUpdated(updated);
    } catch (err) {
      if (currentGeneration === generation) {
        dueError.value = errorMessage(err, "保存到期时间失败");
      }
    } finally {
      if (currentGeneration === generation) {
        savingDue.value = false;
      }
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
    const currentGeneration = generation;

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
      if (currentGeneration === generation) {
        completionError.value = errorMessage(err, "更新完成状态失败");
      }
    } finally {
      if (currentGeneration === generation) {
        savingCompletion.value = false;
      }
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
  };
}
