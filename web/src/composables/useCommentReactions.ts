export function useCommentReactions(
  update: (commentId: number, reactions: Reaction[]) => void,
  fallbackError: string,
) {
  const reacting = ref(new Set<number>());
  const error = ref<string | null>(null);

  async function react(commentId: number, emoji: string) {
    if (reacting.value.has(commentId)) {
      return;
    }

    reacting.value = new Set([...reacting.value, commentId]);
    error.value = null;

    try {
      const result = await toggleReaction(commentId, emoji);

      update(commentId, result.reactions);
    } catch (err) {
      error.value = errorMessage(err, fallbackError);
    } finally {
      const next = new Set(reacting.value);

      next.delete(commentId);
      reacting.value = next;
    }
  }

  function clearError() {
    error.value = null;
  }

  return {
    reacting,
    error,
    react,
    clearError,
  };
}
