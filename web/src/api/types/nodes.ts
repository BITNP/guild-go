import type { UserSummary } from "./auth";
import type { CommentPreview, Reaction } from "./comments";

export interface NodeSummary {
  id: number;
  parentId: number | null;
  title: string;
  dueAt: string | null;
  completedAt: string | null;
  tags: string[];
  sortOrder: number;
  childCount: number;
  commentCount: number;
  latestCommentAt: string | null;
  myRelation: "leader" | "member" | null;
  updatedAt: string;
}

export interface ChildNode extends NodeSummary {
  latestComment: CommentPreview | null;
}

export interface Breadcrumb {
  id: number;
  title: string;
}

export interface NodePermissions {
  canEdit: boolean;
  canDelete: boolean;
  canComment: boolean;
  canManageMembers: boolean;
  canAddChild: boolean;
  canReorderChildren: boolean;
}

export interface NodeDetail {
  id: number;
  parentId: number | null;
  title: string;
  description: string;
  dueAt: string | null;
  completedAt: string | null;
  tags: string[];
  sortOrder: number;
  createdBySubject: string;
  createdBy: UserSummary;
  createdAt: string;
  updatedAt: string;
  breadcrumb: Breadcrumb[];
  directLeaders: UserSummary[];
  directMembers: UserSummary[];
  inheritedLeaderCount: number;
  inheritedMemberCount: number;
  effectiveMemberCount: number;
  subtreeNodeCount: number;
  subtreeCommentCount: number;
  subtreeAttachmentCount: number;
  permissions: NodePermissions;
  children: ChildNode[];
}

export interface RootSummary extends NodeSummary {
  leaders: UserSummary[];
  latestComment: CommentPreview | null;
}

export interface RecentComment {
  id: number;
  nodeId: number;
  nodeTitle: string;
  rootTitle: string;
  author: UserSummary;
  bodyPreview: string;
  createdAt: string;
  reactionSummary: Reaction[];
}

export interface Overview {
  roots: RootSummary[];
  recentComments: RecentComment[];
}

export interface MemberSource {
  id: number;
  title: string;
}

export interface EffectiveMember {
  user: UserSummary;
  direct: boolean;
  sources: MemberSource[];
}

export interface EffectiveMembers {
  leaders: EffectiveMember[];
  members: EffectiveMember[];
}

export interface NodeCreateInput {
  parentId: number | null;
  title: string;
  description: string;
  dueAt: string | null;
  tags: string[];
  leaderSubjects: string[];
  memberSubjects: string[];
}

export interface NodeUpdateInput {
  title: string;
  description: string;
  dueAt: string | null;
  tags: string[];
  completed: boolean;
}
