import type { ChildNode } from "@/api";

import { VIRTUAL_ROOT_ID } from "./layout";
import type { GraphNode } from "./types";

export const buildVirtualRootNode = (rootCount: number): GraphNode => ({
  id: VIRTUAL_ROOT_ID,
  parentId: null,
  title: "开拓者公会",
  sortOrder: -1,
  completedAt: null,
  childCount: rootCount,
});

export function buildVirtualRootDetail(
  nodes: NodeSummary[],
  roots: NodeSummary[],
  canManageRoots: boolean,
): NodeDetail {
  const commentCount = nodes.reduce(
    (total, node) => total + node.commentCount,
    0,
  );

  return {
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
    subtreeNodeCount: nodes.length,
    subtreeCommentCount: commentCount,
    subtreeAttachmentCount: 0,
    permissions: {
      canEdit: false,
      canDelete: false,
      canComment: false,
      canManageMembers: false,
      canAddChild: canManageRoots,
      canReorderChildren: canManageRoots && roots.length > 1,
    },
    children: roots.map((node): ChildNode => ({
      ...node,
      latestComment: null,
    })),
  };
}
