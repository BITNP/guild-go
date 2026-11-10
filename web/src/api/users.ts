import { apiFetch } from "./client";
import type { Profile, UserSummary } from "./types";

export const fetchUsers = (): Promise<UserSummary[]> =>
  apiFetch<UserSummary[]>("/api/users", { cache: "no-store" });

export const fetchUser = (subject: string): Promise<Profile> =>
  apiFetch<Profile>(`/api/users/${encodeURIComponent(subject)}`, {
    cache: "no-store",
  });
