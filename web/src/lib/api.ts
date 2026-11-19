export type Department = "techdept" | "clinic" | "digimedia" | "techsvc";

export interface Position {
  code: string;
  label: string;
  scope: "association" | "department";
  department: Department | null;
}

export interface Permissions {
  systemAdmin: boolean;
  manager: boolean;
  accessDepartments: Department[];
  managedDepartments: Department[];
}

export interface UserSummary {
  subject: string;
  userName: string;
  avatar: string | null;
  positions: Position[];
}

export interface Profile extends UserSummary {
  email: string | null;
  permissions: Permissions | null;
}

export interface AuthStatus {
  state: "anonymous" | "active";
  user: Profile | null;
}

export interface Reaction {
  emoji: string;
  count: number;
  reactedByMe: boolean;
}

export interface Attachment {
  id: number;
  url: string;
  contentType: string;
  sizeBytes: number;
}

export interface Comment {
  id: number;
  nodeId: number;
  author: UserSummary;
  body: string;
  createdAt: string;
  attachments: Attachment[];
  reactions: Reaction[];
  canDelete: boolean;
}

export interface CommentPreview {
  id: number;
  author: UserSummary;
  bodyPreview: string;
  createdAt: string;
  reactionSummary: Reaction[];
  attachmentCount: number;
}

export interface DescriptionChange {
  id: number;
  nodeId: number;
  author: UserSummary;
  before: string;
  after: string;
  createdAt: string;
}

export type ProgressItem =
  | { kind: "comment"; comment: Comment }
  | { kind: "description_change"; descriptionChange: DescriptionChange };

export interface ProgressCursor {
  createdAt: string;
  kind: ProgressItem["kind"];
  id: number;
}

export interface ProgressList {
  items: ProgressItem[];
  hasMore: boolean;
  nextBefore: ProgressCursor | null;
}

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
  recentActivities: RecentActivity[];
}

export type ActivityScope = "all" | "related";

export type RecentActivity =
  (RecentComment & { kind: "comment"; attachmentCount: number }) | NodeActivity;

export interface NodeActivity {
  id: number;
  kind: "node_created" | "node_updated";
  nodeId: number;
  nodeTitle: string;
  rootTitle: string;
  author: UserSummary | null;
  createdAt: string;
  dueAt: string | null;
  completedAt: string | null;
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

export class ApiError extends Error {
  public status: number;

  constructor(status: number, message: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
  }
}

interface CsrfToken {
  token: string;
  headerName: string;
}

const SAFE_METHODS = new Set(["GET", "HEAD", "OPTIONS"]);

async function fetchCsrf(): Promise<CsrfToken> {
  const response = await fetch("/api/auth/csrf", {
    credentials: "same-origin",
    cache: "no-store",
  });
  if (!response.ok) {
    throw new ApiError(response.status, "登录状态已失效");
  }

  return (await response.json()) as CsrfToken;
}

async function apiFetch<T>(
  path: string,
  options: RequestInit = {},
): Promise<T> {
  const headers = new Headers(options.headers);
  const method = (options.method ?? "GET").toUpperCase();
  if (!SAFE_METHODS.has(method)) {
    const csrf = await fetchCsrf();
    headers.set(csrf.headerName, csrf.token);
  }
  if (
    options.body &&
    !(options.body instanceof FormData) &&
    !headers.has("Content-Type")
  ) {
    headers.set("Content-Type", "application/json");
  }
  const response = await fetch(path, {
    credentials: "same-origin",
    ...options,
    headers,
  });
  if (!response.ok) {
    let message = `请求失败（${response.status}）`;
    try {
      const body = (await response.json()) as { message?: string };
      if (body.message) {
        message = body.message;
      }
    } catch {
      // non-JSON error body
    }

    throw new ApiError(response.status, message);
  }
  if (response.status === 204) {
    return undefined as T;
  }

  return (await response.json()) as T;
}

export const fetchAuthStatus = (): Promise<AuthStatus> =>
  apiFetch<AuthStatus>("/api/auth/status", { cache: "no-store" });

export const logout = (): Promise<void> =>
  apiFetch<void>("/api/auth/logout", { method: "POST" });

export const fetchOverview = (
  scope: ActivityScope = "all",
): Promise<Overview> =>
  apiFetch<Overview>(`/api/overview?scope=${scope}`, {
    cache: "no-store",
  });

export const fetchNodes = (): Promise<NodeSummary[]> =>
  apiFetch<NodeSummary[]>("/api/nodes", { cache: "no-store" });

export const fetchNode = (id: number): Promise<NodeDetail> =>
  apiFetch<NodeDetail>(`/api/nodes/${id}`, { cache: "no-store" });

export const createNode = (input: NodeCreateInput): Promise<NodeDetail> =>
  apiFetch<NodeDetail>("/api/nodes", {
    method: "POST",
    body: JSON.stringify(input),
  });

export const updateNode = (
  id: number,
  input: NodeUpdateInput,
): Promise<NodeDetail> =>
  apiFetch<NodeDetail>(`/api/nodes/${id}`, {
    method: "PUT",
    body: JSON.stringify(input),
  });

export const updateNodeFromDetail = (
  node: NodeDetail,
  patch: Partial<NodeUpdateInput>,
): Promise<NodeDetail> =>
  updateNode(node.id, {
    title: node.title,
    description: node.description,
    dueAt: node.dueAt,
    tags: node.tags,
    completed: Boolean(node.completedAt),
    ...patch,
  });

export const moveNode = (
  id: number,
  parentId: number | null,
): Promise<NodeDetail> =>
  apiFetch<NodeDetail>(`/api/nodes/${id}/move`, {
    method: "PUT",
    body: JSON.stringify({ parentId }),
  });

export const deleteNode = (id: number): Promise<void> =>
  apiFetch<void>(`/api/nodes/${id}`, { method: "DELETE" });

export const orderNodes = (
  parentId: number | null,
  ids: number[],
): Promise<void> =>
  apiFetch<void>("/api/nodes/order", {
    method: "PUT",
    body: JSON.stringify({ parentId, ids }),
  });

export const fetchNodeMembers = (id: number): Promise<EffectiveMembers> =>
  apiFetch<EffectiveMembers>(`/api/nodes/${id}/members`, {
    cache: "no-store",
  });

export const saveNodeMembers = (
  id: number,
  leaders: string[],
  members: string[],
): Promise<NodeDetail> =>
  apiFetch<NodeDetail>(`/api/nodes/${id}/members`, {
    method: "PUT",
    body: JSON.stringify({ leaders, members }),
  });

export const createComment = (
  nodeId: number,
  body: string,
  attachmentIds: number[],
): Promise<Comment> =>
  apiFetch<Comment>(`/api/nodes/${nodeId}/comments`, {
    method: "POST",
    body: JSON.stringify({ body, attachmentIds }),
  });

export function fetchProgress(
  nodeId: number,
  before?: ProgressCursor | null,
): Promise<ProgressList> {
  const params = new URLSearchParams({ limit: "30" });
  if (before) {
    params.set("before", before.createdAt);
    params.set("beforeKind", before.kind);
    params.set("beforeId", String(before.id));
  }

  return apiFetch<ProgressList>(`/api/nodes/${nodeId}/progress?${params}`, {
    cache: "no-store",
  });
}

export function uploadCommentImage(
  nodeId: number,
  file: File,
): Promise<{ id: number }> {
  const form = new FormData();
  form.set("file", file);

  return apiFetch(`/api/nodes/${nodeId}/comment-images`, {
    method: "POST",
    body: form,
  });
}

export const deleteCommentUpload = (id: number): Promise<void> =>
  apiFetch(`/api/comment-images/${id}`, { method: "DELETE" });

export const deleteComment = (id: number): Promise<void> =>
  apiFetch<void>(`/api/comments/${id}`, { method: "DELETE" });

export const toggleReaction = (
  id: number,
  emoji: string,
): Promise<{ reactions: Reaction[] }> =>
  apiFetch<{ reactions: Reaction[] }>(
    `/api/comments/${id}/reactions/${encodeURIComponent(emoji)}`,
    { method: "PUT" },
  );

export const fetchReactionUsers = (
  id: number,
  emoji: string,
): Promise<UserSummary[]> =>
  apiFetch<UserSummary[]>(
    `/api/comments/${id}/reactions/${encodeURIComponent(emoji)}`,
    { cache: "no-store" },
  );

export const fetchUsers = (): Promise<UserSummary[]> =>
  apiFetch<UserSummary[]>("/api/users", { cache: "no-store" });

export const fetchUser = (subject: string): Promise<Profile> =>
  apiFetch<Profile>(`/api/users/${encodeURIComponent(subject)}`, {
    cache: "no-store",
  });

export const loginUrl = (returnTo: string): string =>
  `/api/auth/login?returnTo=${encodeURIComponent(returnTo)}`;
