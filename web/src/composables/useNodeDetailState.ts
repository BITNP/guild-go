import type { NodeDetail } from "@/api";

interface LoadNodeOptions {
  silent?: boolean;
  errorFallback?: string;
  shouldApply?: () => boolean;
  afterLoad?: (node: NodeDetail) => Promise<void> | void;
}

export function useNodeDetailState(initialLoading = false) {
  const detail = ref<NodeDetail | null>(null);
  const loading = ref(initialLoading);
  const error = ref<string | null>(null);

  let generation = 0;

  async function load(id: number, options: LoadNodeOptions = {}) {
    const current = ++generation;

    if (!options.silent) {
      loading.value = true;
    }

    error.value = null;

    try {
      const node = await fetchNode(id);

      if (current !== generation || options.shouldApply?.() === false) {
        return;
      }

      detail.value = node;
      await options.afterLoad?.(node);
    } catch (err) {
      if (current === generation && options.shouldApply?.() !== false) {
        detail.value = null;
        error.value = errorMessage(
          err,
          options.errorFallback ?? "无法加载节点详情",
        );
      }
    } finally {
      if (current === generation) {
        loading.value = false;
      }
    }
  }

  function setDetail(node: NodeDetail) {
    generation++;
    detail.value = node;
    error.value = null;
    loading.value = false;
  }

  function setError(message: string) {
    generation++;
    detail.value = null;
    error.value = message;
    loading.value = false;
  }

  function reset(nextLoading = false) {
    generation++;
    detail.value = null;
    error.value = null;
    loading.value = nextLoading;
  }

  return {
    detail,
    loading,
    error,
    load,
    setDetail,
    setError,
    reset,
  };
}
