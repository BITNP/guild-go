import { computed, ref } from 'vue'
import {
  fetchAuthStatus,
  logout as apiLogout,
  loginUrl,
  type Profile,
} from '@/lib/api'
import { safeReturnTo } from '@/lib/utils'

const user = ref<Profile | null>(null)
const state = ref<'anonymous' | 'active'>('anonymous')
const loading = ref(true)
const error = ref<string | null>(null)
let inflight: Promise<boolean> | null = null

const isManager = computed(() => user.value?.permissions?.manager ?? false)
const isAdmin = computed(() => user.value?.permissions?.systemAdmin ?? false)

async function refresh(): Promise<boolean> {
  if (inflight) return inflight
  inflight = (async () => {
    try {
      const status = await fetchAuthStatus()
      state.value = status.state
      user.value = status.user
      error.value = null
      return status.state === 'active'
    } catch (err) {
      error.value = err instanceof Error ? err.message : '无法连接服务器'
      state.value = 'anonymous'
      user.value = null
      return false
    } finally {
      loading.value = false
      inflight = null
    }
  })()
  return inflight
}

function login(returnTo = window.location.pathname + window.location.search): void {
  window.location.href = loginUrl(safeReturnTo(returnTo))
}

async function logout(): Promise<void> {
  await apiLogout()
  user.value = null
  state.value = 'anonymous'
}

export function useAuth() {
  return {
    user,
    state,
    loading,
    error,
    isManager,
    isAdmin,
    refresh,
    login,
    logout,
  }
}