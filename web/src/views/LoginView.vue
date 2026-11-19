<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { NAlert } from 'naive-ui'
import { Moon, Sun } from '@lucide/vue'
import { useAuth } from '@/composables/useAuth'
import { useTheme } from '@/composables/useTheme'
import { safeReturnTo } from '@/lib/utils'

const route = useRoute()
const router = useRouter()
const { state, error: connectionError, refresh, login } = useAuth()
const { isDark, toggleTheme } = useTheme()

const returnTo = computed(() => safeReturnTo(route.query.redirect))
const error = computed(() => {
  const code = route.query.error
  if (code === 'provider_unavailable' || code === 'unavailable')
    return '登录服务暂时不可用，请稍后重试。'
  if (code === 'not_member')
    return '此账号暂无公会访问权限，请联系部门管理员开通。'
  if (code === 'no_roles') return '此账号尚未分配网协角色，请联系管理员。'
  if (code) return '登录已取消或无法完成，请重试。'
  return connectionError.value ? '无法连接登录服务，请稍后重试。' : ''
})

onMounted(async () => {
  if (await refresh()) await router.replace({ name: 'home' })
})
</script>

<template>
  <main class="login-stage" :class="{ 'is-dark': isDark }">
    <button
      type="button"
      class="theme-toggle"
      :aria-label="isDark ? '切换到浅色主题' : '切换到深色主题'"
      :aria-pressed="isDark"
      :title="isDark ? '切换到浅色主题' : '切换到深色主题'"
      @click="toggleTheme"
    >
      <Sun v-if="isDark" :size="17" aria-hidden="true" />
      <Moon v-else :size="17" aria-hidden="true" />
    </button>

    <div class="login-stack">
      <div class="guild-emblem" aria-hidden="true">
        <svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
          <path d="M12 2.6L16.2 6.8L12 11L7.8 6.8L12 2.6Z" fill="currentColor" />
          <path
            d="M6.8 7.8L11 12L6.8 16.2L2.6 12L6.8 7.8Z"
            fill="currentColor"
            fill-opacity="0.72"
          />
          <path
            d="M17.2 7.8L21.4 12L17.2 16.2L13 12L17.2 7.8Z"
            fill="currentColor"
            fill-opacity="0.72"
          />
          <path
            d="M12 13L16.2 17.2L12 21.4L7.8 17.2L12 13Z"
            fill="currentColor"
            fill-opacity="0.46"
          />
        </svg>
      </div>

      <h1 class="login-title">公会</h1>
      <p class="login-org">网络开拓者协会</p>

      <NAlert v-if="error" type="error" :show-icon="false" class="login-alert">
        {{ error }}
      </NAlert>

      <button
        type="button"
        class="login-button"
        :disabled="state === 'active'"
        @click="login(returnTo)"
      >
        使用网协通行证登录
      </button>
    </div>
  </main>
</template>

<style scoped>
.login-stage {
  --stage-bg: #f4f6f9;
  --stage-wash: rgba(255, 255, 255, 0.92);
  --emblem-bg: linear-gradient(180deg, #ffffff 0%, #f5f7fb 100%);
  --emblem-border: rgba(23, 32, 43, 0.1);
  --emblem-shadow:
    0 8px 20px rgba(23, 32, 43, 0.06), inset 0 1px 0 rgba(255, 255, 255, 0.95);
  --emblem-fg: #1f5fbf;
  --toggle-hover: rgba(23, 32, 43, 0.06);

  position: relative;
  min-height: 100dvh;
  display: grid;
  place-items: center;
  padding: 24px 20px calc(24px + 6vh);
  background-color: var(--stage-bg);
  background-image: radial-gradient(
    ellipse 68% 42% at 50% 0%,
    var(--stage-wash),
    transparent 70%
  );
  color: var(--guild-text);
}

.login-stage.is-dark {
  --stage-bg: #0e141b;
  --stage-wash: rgba(88, 142, 230, 0.075);
  --emblem-bg: linear-gradient(180deg, #1b2531 0%, #131b24 100%);
  --emblem-border: rgba(255, 255, 255, 0.09);
  --emblem-shadow:
    0 12px 28px rgba(0, 0, 0, 0.42), inset 0 1px 0 rgba(255, 255, 255, 0.08);
  --emblem-fg: #dde6f0;
  --toggle-hover: rgba(255, 255, 255, 0.06);
}

.theme-toggle {
  position: absolute;
  top: 20px;
  right: 20px;
  width: 34px;
  height: 34px;
  padding: 0;
  border: 0;
  border-radius: 8px;
  background: transparent;
  color: var(--guild-muted);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition:
    background-color 0.15s ease,
    color 0.15s ease;
}

.theme-toggle:hover {
  background: var(--toggle-hover);
  color: var(--guild-text);
}

.login-stack {
  width: min(304px, 100%);
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
}

.guild-emblem {
  width: 52px;
  height: 52px;
  border-radius: 13px;
  background: var(--emblem-bg);
  border: 1px solid var(--emblem-border);
  box-shadow: var(--emblem-shadow);
  color: var(--emblem-fg);
  display: grid;
  place-items: center;
}

.guild-emblem svg {
  width: 24px;
  height: 24px;
}

.login-title {
  margin: 22px 0 0;
  font-size: 24px;
  line-height: 30px;
  font-weight: 650;
  letter-spacing: -0.015em;
}

.login-org {
  margin: 6px 0 0;
  font-size: 13.5px;
  line-height: 20px;
  color: var(--guild-muted);
  letter-spacing: 0.02em;
}

.login-alert {
  width: 100%;
  margin-top: 20px;
  text-align: left;
}

.login-button {
  width: 100%;
  height: 42px;
  margin-top: 28px;
  padding: 0 16px;
  border-radius: 9px;
  border: 1px solid rgba(255, 255, 255, 0.12);
  background: linear-gradient(180deg, #2966d4 0%, #1f56b8 100%);
  color: #ffffff;
  font: inherit;
  font-size: 14px;
  font-weight: 560;
  letter-spacing: 0.01em;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  box-shadow:
    0 2px 6px rgba(23, 56, 117, 0.24),
    inset 0 1px 0 rgba(255, 255, 255, 0.16);
  transition:
    background 0.15s ease,
    box-shadow 0.15s ease,
    transform 0.1s ease;
}

.login-button:hover:not(:disabled) {
  background: linear-gradient(180deg, #3170e0 0%, #235ec4 100%);
  box-shadow:
    0 4px 12px rgba(23, 56, 117, 0.32),
    inset 0 1px 0 rgba(255, 255, 255, 0.2);
}

.login-button:active:not(:disabled) {
  background: #1b4da6;
  transform: translateY(0.5px);
}

.login-button:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}
</style>
