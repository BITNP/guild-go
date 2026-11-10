import type { Comment } from "@/api";

export function useProgressFeed(nodeId: MaybeRefOrGetter<number | null>) {
  const items = ref<ProgressItem[]>([]);
  const loading = ref(false);
  const loadingMore = ref(false);
  const loadError = ref<string | null>(null);
  const error = ref<string | null>(null);
  const reacting = ref(new Set<number>());
  const nextBefore = ref<ProgressCursor | null>(null);
  const hasMore = computed(() => nextBefore.value !== null);

  let request = 0;
  let disposed = false;

  async function reload() {
    const requestedNode = toValue(nodeId);
    const current = ++request;

    items.value = [];
    nextBefore.value = null;
    loadingMore.value = false;
    loadError.value = null;
    error.value = null;
    loading.value = requestedNode !== null;

    if (requestedNode === null) {
      return;
    }

    const isCurrent = () =>
      !disposed && current === request && requestedNode === toValue(nodeId);

    try {
      const page = await fetchProgress(requestedNode);

      if (!isCurrent()) {
        return;
      }

      const addedKeys = new Set(items.value.map(progressKey));

      items.value = [
        ...items.value,
        ...page.items.filter((item) => !addedKeys.has(progressKey(item))),
      ];
      nextBefore.value = page.nextBefore;
    } catch (err) {
      if (isCurrent()) {
        loadError.value = errorMessage(err, "无法加载进展，请重试");
      }
    } finally {
      if (isCurrent()) {
        loading.value = false;
      }
    }
  }

  async function loadMore() {
    const requestedNode = toValue(nodeId);
    const before = nextBefore.value;

    if (
      requestedNode === null ||
      loading.value ||
      loadingMore.value ||
      before === null
    ) {
      return;
    }

    const current = request;
    const isCurrent = () =>
      !disposed && current === request && requestedNode === toValue(nodeId);

    loadingMore.value = true;
    error.value = null;

    try {
      const page = await fetchProgress(requestedNode, before);

      if (!isCurrent()) {
        return;
      }

      items.value = [...items.value, ...page.items];
      nextBefore.value = page.nextBefore;
    } catch (err) {
      if (isCurrent()) {
        error.value = errorMessage(err, "加载更多进展失败，请重试");
      }
    } finally {
      if (isCurrent()) {
        loadingMore.value = false;
      }
    }
  }

  function add(comment: Comment) {
    if (comment.nodeId !== toValue(nodeId)) {
      return;
    }

    error.value = null;

    if (
      items.value.some((item) => progressKey(item) === `comment:${comment.id}`)
    ) {
      return;
    }

    items.value = [{ kind: "comment", comment }, ...items.value];
  }

  function remove(commentId: number) {
    items.value = items.value.filter(
      (item) => item.kind !== "comment" || item.comment.id !== commentId,
    );
  }

  async function react(commentId: number, emoji: string) {
    const requestedNode = toValue(nodeId);

    if (requestedNode === null || reacting.value.has(commentId)) {
      return;
    }

    const isCurrent = () => !disposed && requestedNode === toValue(nodeId);

    reacting.value.add(commentId);
    error.value = null;

    try {
      const { reactions } = await toggleReaction(commentId, emoji);

      if (isCurrent()) {
        items.value = items.value.map((item) =>
          item.kind === "comment" && item.comment.id === commentId
            ? { ...item, comment: { ...item.comment, reactions } }
            : item,
        );
      }
    } catch (err) {
      if (isCurrent()) {
        error.value = errorMessage(err, "表情操作失败");
      }
    } finally {
      reacting.value.delete(commentId);
    }
  }

  watch(() => toValue(nodeId), reload, { immediate: true, flush: "sync" });

  onScopeDispose(() => {
    disposed = true;
  });

  return {
    items,
    loading,
    loadingMore,
    loadError,
    error,
    hasMore,
    reacting,
    reload,
    loadMore,
    add,
    remove,
    react,
  };
}
