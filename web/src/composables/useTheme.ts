import { computed, ref } from 'vue'

export type ThemeMode = 'light' | 'dark'

const storageKey = 'guild-theme'

function initialTheme(): ThemeMode {
  if (typeof window === 'undefined') return 'light'
  try {
    const saved = window.localStorage.getItem(storageKey)
    if (saved === 'light' || saved === 'dark') return saved
  } catch {
    // Storage can be unavailable in restricted browser contexts.
  }
  return window.matchMedia('(prefers-color-scheme: dark)').matches
    ? 'dark'
    : 'light'
}

const mode = ref<ThemeMode>(initialTheme())

function applyTheme(next: ThemeMode) {
  if (typeof document === 'undefined') return
  document.documentElement.dataset.theme = next
  document.documentElement.style.colorScheme = next
  const meta = document.querySelector<HTMLMetaElement>('meta[name="theme-color"]')
  if (meta) meta.content = next === 'dark' ? '#10161d' : '#1f5fbf'
}

applyTheme(mode.value)

export function useTheme() {
  const isDark = computed(() => mode.value === 'dark')

  function setTheme(next: ThemeMode) {
    mode.value = next
    try {
      window.localStorage.setItem(storageKey, next)
    } catch {
      // The active theme still applies for the current session.
    }
    applyTheme(next)
  }

  function toggleTheme() {
    setTheme(isDark.value ? 'light' : 'dark')
  }

  return {
    mode,
    isDark,
    setTheme,
    toggleTheme,
  }
}
