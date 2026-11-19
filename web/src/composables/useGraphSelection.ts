import { computed, ref } from "vue";

import { useAuth } from "@/composables/useAuth";
import { useGraphProgress } from "@/composables/useGraphProgress";
import { useNodeStore } from "@/composables/useNodeStore";
import { VIRTUAL_ROOT_ID } from "@/graph/layout";
import type { GraphNeighborhood, GraphNode } from "@/graph/types";
import type { ChildNode, Comment, NodeDetail } from "@/lib/api";
import { fetchNode, toggleReaction } from "@/lib/api";
import { errorMessage, progressKey } from "@/lib/utils";

export function useGraphSelection() {
  const { nodes, roots, byId, childrenOf } = useNodeStore();
  const { isManager, isAdmin } = useAuth();
  const {
    items: progress,
    loading: progressLoading,
    hasMore: progressHaveMore,
    error: progressError,
    load: loadProgress,
    loadOlder: loadOlderProgress,
    add: addProgressComment,
    updateReactions,
    removeProgress,
    clear: clearProgress,
  } = useGraphProgress();

  const active = ref<GraphNeighborhood | null>(null);
  const detail = ref<NodeDetail | null>(null);
  const detailLoading = ref(false);
  const detailError = ref<string | null>(null);
  const reactionError = ref<string | null>(null);
  const reacting = ref(new Set<number>());
  const expandedProgressKey = ref<string | null>(null);
  let detailGeneration = 0;

  const canManageRoots = computed(() => isManager.value || isAdmin.value);
  const expandedProgress = computed(() => {
    const key = expandedProgressKey.value;

    return key == null
      ? null
      : (progress.value.find((item) => progressKey(item) === key) ?? null);
  });
  const selectionLoading = computed(() => {
    const selected = active.value;
    if (!selected) {
      return false;
    }
    const detailPending =
      detailLoading.value ||
      (detail.value?.id !== selected.node.id && !detailError.value);

    return detailPending || progressLoading.value;
  });

  const virtualRootNode = (): GraphNode => ({
    id: VIRTUAL_ROOT_ID,
    parentId: null,
    title: "开拓者公会",
    sortOrder: -1,
    completedAt: null,
    childCount: roots.value.length,
    commentCount: roots.value.reduce(
      (total, node) => total + node.commentCount,
      0,
    ),
  });

  const virtualRootDetail = (): NodeDetail => ({
    id: VIRTUAL_ROOT_ID,
    parentId: null,
    title: "开拓者公会",
    description: "",
    dueAt: null,
    completedAt: null,
    tags: [],
    sortOrder: -1,
    createdBySubject: "",
    createdBy: {
      subject: "",
      userName: "",
      avatar: null,
      positions: [],
    },
    createdAt: "",
    updatedAt: "",
    breadcrumb: [],
    directLeaders: [],
    directMembers: [],
    inheritedLeaderCount: 0,
    inheritedMemberCount: 0,
    effectiveMemberCount: 0,
    subtreeNodeCount: nodes.value.length,
    subtreeCommentCount: nodes.value.reduce(
      (total, node) => total + node.commentCount,
      0,
    ),
    subtreeAttachmentCount: 0,
    permissions: {
      canEdit: false,
      canDelete: false,
      canComment: false,
      canManageMembers: false,
      canAddChild: canManageRoots.value,
      canReorderChildren: canManageRoots.value && roots.value.length > 1,
    },
    children: roots.value.map((node): ChildNode => ({
      ...node,
      latestComment: null,
    })),
  });

  function resetTransientState() {
    detailGeneration++;
    detail.value = null;
    detailError.value = null;
    reactionError.value = null;
    expandedProgressKey.value = null;
    clearProgress();
  }

  async function loadDetail() {
    const id = active.value?.node.id;
    if (id == null || id === VIRTUAL_ROOT_ID) {
      return;
    }
    const current = ++detailGeneration;
    detailLoading.value = true;
    detailError.value = null;
    try {
      const node = await fetchNode(id);
      if (current === detailGeneration && active.value?.node.id === id) {
        detail.value = node;
        detailLoading.value = false;
        await loadProgress(id);
      }
    } catch (err) {
      if (current === detailGeneration && active.value?.node.id === id) {
        detailError.value = errorMessage(err, "无法加载节点详情");
      }
    } finally {
      if (current === detailGeneration) {
        detailLoading.value = false;
      }
    }
  }

  // The whole tree is already in the node store, so the focused neighborhood is
  // a pure lookup. childrenOf keeps the same (sortOrder, id) order as the server.
  function selectNode(id: number) {
    const changed = active.value?.node.id !== id;
    if (id === VIRTUAL_ROOT_ID) {
      if (changed) {
        resetTransientState();
        detailLoading.value = false;
      }
      active.value = {
        node: virtualRootNode(),
        parent: null,
        children: roots.value,
      };
      detail.value = virtualRootDetail();

      return;
    }
    const node = byId.value.get(id);
    if (!node) {
      return;
    }
    if (changed) {
      resetTransientState();
      detailLoading.value = true;
    }
    const parent =
      node.parentId == null
        ? virtualRootNode()
        : (byId.value.get(node.parentId) ?? null);
    active.value = { node, parent, children: childrenOf(id) };
    if (changed) {
      void loadDetail();
    }
  }

  function clearSelection() {
    active.value = null;
    detailLoading.value = false;
    resetTransientState();
  }

  function syncUpdatedNode(node: NodeDetail) {
    if (active.value?.node.id !== node.id) {
      return;
    }
    detail.value = node;
    active.value = {
      ...active.value,
      node: {
        ...active.value.node,
        title: node.title,
        parentId: node.parentId,
        completedAt: node.completedAt,
      },
    };
  }

  function onCommentCreated(comment: Comment) {
    addProgressComment(comment);
    reactionError.value = null;
    if (detail.value?.id === comment.nodeId) {
      detail.value = {
        ...detail.value,
        subtreeCommentCount: detail.value.subtreeCommentCount + 1,
      };
    }
    if (active.value?.node.id === comment.nodeId) {
      active.value = {
        ...active.value,
        node: {
          ...active.value.node,
          commentCount: active.value.node.commentCount + 1,
        },
      };
    }
  }

  function onCommentDeleted(commentId: number) {
    removeProgress(`comment:${commentId}`);
    expandedProgressKey.value = null;
    if (detail.value) {
      detail.value = {
        ...detail.value,
        subtreeCommentCount: Math.max(0, detail.value.subtreeCommentCount - 1),
      };
    }
    if (active.value) {
      active.value = {
        ...active.value,
        node: {
          ...active.value.node,
          commentCount: Math.max(0, active.value.node.commentCount - 1),
        },
      };
    }
  }

  async function reactToComment(commentId: number, emoji: string) {
    if (reacting.value.has(commentId)) {
      return;
    }
    reacting.value = new Set([...reacting.value, commentId]);
    reactionError.value = null;
    try {
      const result = await toggleReaction(commentId, emoji);
      updateReactions(commentId, result.reactions);
    } catch (err) {
      reactionError.value = errorMessage(err, "表情操作失败");
    } finally {
      const next = new Set(reacting.value);
      next.delete(commentId);
      reacting.value = next;
    }
  }

  return {
    active,
    detail,
    detailLoading,
    detailError,
    selectionLoading,
    progress,
    progressLoading,
    progressHaveMore,
    progressError,
    loadProgress,
    loadOlderProgress,
    expandedProgressKey,
    expandedProgress,
    reacting,
    reactionError,
    selectNode,
    clearSelection,
    loadDetail,
    syncUpdatedNode,
    onCommentCreated,
    onCommentDeleted,
    reactToComment,
  };
}
