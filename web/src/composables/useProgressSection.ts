import type { Comment } from "@/api";

interface ProgressSectionOptions {
  nodeId: MaybeRefOrGetter<number>;
  updatedAt: MaybeRefOrGetter<string>;
  onChanged: () => void;
}

export function useProgressSection(options: ProgressSectionOptions) {
  const feed = useProgressFeed(options.nodeId);
  const deleteTarget = ref<Comment | null>(null);
  const deleting = ref(false);
  const deleteError = ref<string | null>(null);
  const count = computed(() =>
    !feed.loading.value && !feed.loadError.value
      ? `${feed.items.value.length}${feed.hasMore.value ? "+" : ""}`
      : undefined,
  );

  function created(comment: Comment) {
    feed.add(comment);
    options.onChanged();
  }

  function reaction(comment: Comment, emoji: string) {
    void feed.react(comment.id, emoji);
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
      feed.remove(id);
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
    ([nodeId], [previousNodeId]) => {
      if (nodeId === previousNodeId) {
        void feed.reload();
      }
    },
  );

  return {
    items: feed.items,
    loading: feed.loading,
    displayError: feed.error,
    loadError: feed.loadError,
    hasMore: feed.hasMore,
    loadingMore: feed.loadingMore,
    deleteTarget,
    deleting,
    deleteError,
    reacting: feed.reacting,
    count,
    load: feed.reload,
    loadMore: feed.loadMore,
    created,
    reaction,
    askDelete,
    confirmDelete,
    closeDelete,
  };
}
