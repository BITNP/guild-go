import type { Comment } from "@/api";

interface ProgressSectionOptions {
  nodeId: MaybeRefOrGetter<number>;
  updatedAt: MaybeRefOrGetter<string>;
  onChanged: () => void;
}

export function useProgressSection(options: ProgressSectionOptions) {
  const items = ref<ProgressItem[]>([]);
  const loading = ref(true);
  const error = ref<string | null>(null);
  const loadError = ref<string | null>(null);
  const hasMore = ref(false);
  const before = ref<ProgressCursor | null>(null);
  const loadingMore = ref(false);
  const deleteTarget = ref<Comment | null>(null);
  const deleting = ref(false);
  const deleteError = ref<string | null>(null);

  function updateReactions(commentId: number, reactions: Reaction[]) {
    const target = items.value.find(
      (item) => item.kind === "comment" && item.comment.id === commentId,
    );
    if (target?.kind === "comment") {
      target.comment.reactions = reactions;
    }
  }

  const {
    reacting,
    error: reactionError,
    react,
  } = useCommentReactions(updateReactions, "表情操作失败");
  const displayError = computed(() => error.value ?? reactionError.value);
  const count = computed(() =>
    !loading.value && !loadError.value
      ? `${items.value.length}${hasMore.value ? "+" : ""}`
      : undefined,
  );

  async function load() {
    const requestedNode = toValue(options.nodeId);
    const requestedUpdate = toValue(options.updatedAt);
    const current = () =>
      requestedNode === toValue(options.nodeId) &&
      requestedUpdate === toValue(options.updatedAt);

    loading.value = true;
    loadError.value = null;
    try {
      const page = await fetchProgress(requestedNode);
      if (!current()) {
        return;
      }
      items.value = page.items;
      hasMore.value = page.hasMore;
      before.value = page.nextBefore;
    } catch (err) {
      if (current()) {
        loadError.value =
          err instanceof ApiError
            ? err.message
            : "无法加载进展，请检查网络后重试";
      }
    } finally {
      if (current()) {
        loading.value = false;
      }
    }
  }

  async function loadMore() {
    if (loading.value || loadingMore.value || before.value === null) {
      return;
    }

    const requestedNode = toValue(options.nodeId);
    const requestedUpdate = toValue(options.updatedAt);

    loadingMore.value = true;
    error.value = null;
    try {
      const page = await fetchProgress(requestedNode, before.value);
      if (
        requestedNode !== toValue(options.nodeId) ||
        requestedUpdate !== toValue(options.updatedAt)
      ) {
        return;
      }
      items.value.push(...page.items);
      hasMore.value = page.hasMore;
      before.value = page.nextBefore;
    } catch (err) {
      error.value =
        err instanceof ApiError ? err.message : "加载更多进展失败，请重试";
    } finally {
      loadingMore.value = false;
    }
  }

  function created(comment: Comment) {
    items.value.unshift({ kind: "comment", comment });
    options.onChanged();
  }

  function reaction(comment: Comment, emoji: string) {
    void react(comment.id, emoji);
  }

  function askDelete(comment: Comment) {
    deleteTarget.value = comment;
    deleteError.value = null;
  }

  async function confirmDelete() {
    if (!deleteTarget.value || deleting.value) {
      return;
    }

    const id = deleteTarget.value.id;

    deleting.value = true;
    deleteError.value = null;
    try {
      await deleteComment(id);
      items.value = items.value.filter(
        (item) => item.kind !== "comment" || item.comment.id !== id,
      );
      deleteTarget.value = null;
      options.onChanged();
    } catch (err) {
      deleteError.value = errorMessage(err, "删除进展失败");
    } finally {
      deleting.value = false;
    }
  }

  function closeDelete() {
    deleteTarget.value = null;
  }

  watch(
    [() => toValue(options.nodeId), () => toValue(options.updatedAt)],
    load,
    { immediate: true },
  );

  return {
    items,
    loading,
    displayError,
    loadError,
    hasMore,
    loadingMore,
    deleteTarget,
    deleting,
    deleteError,
    reacting,
    count,
    load,
    loadMore,
    created,
    reaction,
    askDelete,
    confirmDelete,
    closeDelete,
  };
}
