import type { NodeSummary } from "@/api";
import { VIRTUAL_ROOT_ID } from "@/graph/layout";
import type { GraphNeighborhood } from "@/graph/types";
import {
  buildVirtualRootDetail,
  buildVirtualRootNode,
} from "@/graph/virtualRoot";

import { useNodeDetailState } from "./useNodeDetailState";

interface GraphSelectionOptions {
  byId: MaybeRefOrGetter<ReadonlyMap<number, NodeSummary>>;
  canManageRoots: MaybeRefOrGetter<boolean>;
  childrenOf: (parentId: number) => NodeSummary[];
  loadProgress: (nodeId: number) => Promise<void>;
  nodes: MaybeRefOrGetter<NodeSummary[]>;
  roots: MaybeRefOrGetter<NodeSummary[]>;
}

export function useGraphSelection(options: GraphSelectionOptions) {
  const selectedId = ref<number | null>(null);
  const {
    detail,
    loading: detailLoading,
    error: detailError,
    load: fetchDetail,
    setDetail,
    reset: resetDetail,
  } = useNodeDetailState();
  const virtualRootNode = computed(() => {
    const roots = toValue(options.roots);

    return buildVirtualRootNode(roots.length);
  });
  const active = computed<GraphNeighborhood | null>(() => {
    const id = selectedId.value;
    if (id == null) {
      return null;
    }
    if (id === VIRTUAL_ROOT_ID) {
      return {
        node: virtualRootNode.value,
        parent: null,
        children: toValue(options.roots),
      };
    }

    const node = toValue(options.byId).get(id);
    if (!node) {
      return null;
    }

    const parent =
      node.parentId == null
        ? virtualRootNode.value
        : (toValue(options.byId).get(node.parentId) ?? null);

    return {
      node,
      parent,
      children: options.childrenOf(id),
    };
  });

  async function loadDetail() {
    const selected = active.value;
    if (!selected || selected.node.id === VIRTUAL_ROOT_ID) {
      return;
    }

    const id = selected.node.id;

    await fetchDetail(id, {
      errorFallback: "无法加载节点详情",
      shouldApply: () => selectedId.value === id,
      afterLoad: async () => {
        await options.loadProgress(id);
      },
    });
  }

  function showVirtualDetail() {
    setDetail(
      buildVirtualRootDetail(
        toValue(options.nodes),
        toValue(options.roots),
        toValue(options.canManageRoots),
      ),
    );
  }

  function select(id: number) {
    selectedId.value = id;
  }

  function clear() {
    selectedId.value = null;
  }

  async function refresh() {
    if (selectedId.value === VIRTUAL_ROOT_ID) {
      showVirtualDetail();

      return;
    }

    await loadDetail();
  }

  watch(selectedId, (id) => {
    resetDetail(id != null);

    if (id == null) {
      return;
    }
    if (id === VIRTUAL_ROOT_ID) {
      showVirtualDetail();

      return;
    }

    void loadDetail();
  });

  watch(
    [options.nodes, options.roots, options.canManageRoots],
    () => {
      if (selectedId.value === VIRTUAL_ROOT_ID) {
        showVirtualDetail();
      }
    },
    { deep: false },
  );

  return {
    active,
    detail,
    detailError,
    detailLoading,
    loadDetail,
    refresh,
    select,
    selectedId,
    setDetail,
    clear,
  };
}
