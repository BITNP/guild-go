import { apiFetch } from "./client";
import type {
  Comment,
  CommentList,
  ProgressCursor,
  ProgressList,
  Reaction,
  UserSummary,
} from "./types";

export function fetchComments(
  nodeId: number,
  before?: number | null,
): Promise<CommentList> {
  const params = new URLSearchParams();
  params.set("limit", "30");
  if (before != null) {
    params.set("before", String(before));
  }

  return apiFetch<CommentList>(
    `/api/nodes/${nodeId}/comments?${params.toString()}`,
    {
      cache: "no-store",
    },
  );
}

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

  return apiFetch<{ id: number }>(`/api/nodes/${nodeId}/comment-images`, {
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

export const attachmentUrl = (id: number): string => `/api/attachments/${id}`;
