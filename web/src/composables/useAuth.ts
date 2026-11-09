import type { Profile } from "@/api";

const user = ref<Profile | null>(null);
const state = ref<"anonymous" | "active">("anonymous");
const loading = ref(true);
const error = ref<string | null>(null);
let inflight: Promise<boolean> | null = null;

const isManager = computed(() => user.value?.permissions?.manager ?? false);
const isAdmin = computed(() => user.value?.permissions?.systemAdmin ?? false);

async function refresh(): Promise<boolean> {
  if (inflight) {
    return inflight;
  }
  inflight = (async () => {
    try {
      const status = await fetchAuthStatus();
      state.value = status.state;
      user.value = status.user;
      error.value = null;

      return status.state === "active";
    } catch (err) {
      error.value = err instanceof Error ? err.message : "无法连接服务器";
      state.value = "anonymous";
      user.value = null;

      return false;
    } finally {
      loading.value = false;
      inflight = null;
    }
  })();

  return inflight;
}

function login(
  returnTo = window.location.pathname + window.location.search,
): void {
  window.location.href = loginUrl(safeReturnTo(returnTo));
}

async function signOut(): Promise<void> {
  await logout();
  user.value = null;
  state.value = "anonymous";
}

export const useAuth = () => ({
  user,
  state,
  loading,
  error,
  isManager,
  isAdmin,
  refresh,
  login,
  logout: signOut,
});
