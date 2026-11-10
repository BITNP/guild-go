import { apiFetch } from "./client";
import type { AuthStatus, Profile } from "./types";

export const fetchAuthStatus = (): Promise<AuthStatus> =>
  apiFetch<AuthStatus>("/api/auth/status", { cache: "no-store" });

export const fetchMe = (): Promise<Profile> =>
  apiFetch<Profile>("/api/auth/me");

export const logout = (): Promise<void> =>
  apiFetch<void>("/api/auth/logout", { method: "POST" });

export const loginUrl = (returnTo: string): string =>
  `/api/auth/login?returnTo=${encodeURIComponent(returnTo)}`;
