import { apiFetch } from "./client";
import type {
  EffectiveMembers,
  NodeCreateInput,
  NodeDetail,
  NodeSummary,
  NodeUpdateInput,
  Overview,
} from "./types";

export const fetchOverview = (): Promise<Overview> =>
  apiFetch<Overview>("/api/overview", { cache: "no-store" });

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
