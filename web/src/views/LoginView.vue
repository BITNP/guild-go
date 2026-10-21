<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ArrowRight, ShieldCheck } from '@lucide/vue'
import { useAuth } from '@/composables/useAuth'
import { safeReturnTo } from '@/lib/utils'

const route = useRoute()
const router = useRouter()
const { state, refresh, login } = useAuth()

const returnTo = computed(() => safeReturnTo(route.query.redirect))
const errorMessage = computed(() => {
  const code = route.query.error
  if (code === 'provider_unavailable' || code === 'unavailable') {
    return '登录服务暂时不可用，请稍后重试。'
  }
  if (code === 'not_member') {
    return '这个账号没有公会访问权限。如果你应该拥有权限，请联系管理员检查 Keycloak 角色。'
  }
  if (code === 'no_roles') {
    return '身份提供方没有返回 realm 角色。请联系管理员检查 roles scope。'
  }
  if (code) {
    return '登录已取消或无法完成，请重试。'
  }
  return ''
})

onMounted(async () => {
  try {
    const active = await refresh()
    if (active) await router.replace({ name: 'home' })
  } catch {
    // The guard already handles the unavailable case.
  }
})
</script>

<template>
  <div class="flex min-h-full items-center justify-center bg-canvas p-5">
    <div class="w-full max-w-sm">
      <div class="panel px-6 py-7">
        <div class="flex flex-col items-center text-center">
          <div class="flex h-11 w-11 items-center justify-center rounded-xl bg-accent text-lg font-bold text-white shadow-sm">先</div>
          <h1 class="mt-3 text-lg font-semibold tracking-tight text-ink">公会</h1>
          <p class="mt-1 text-[12px] text-muted">用 BITNP 统一身份登录，继续管理节点与讨论。</p>
        </div>

        <p
          v-if="errorMessage"
          class="mt-4 rounded-md border border-danger/20 bg-danger-soft px-3 py-2 text-[12px] leading-5 text-danger"
          role="alert"
        >
          {{ errorMessage }}
        </p>

        <button
          type="button"
          class="btn btn-primary mt-5 h-9 w-full"
          :disabled="state === 'active'"
          @click="login(returnTo)"
        >
          使用 BITNP 登录
          <ArrowRight class="h-4 w-4" />
        </button>

        <p class="mt-4 flex items-start gap-1.5 text-[10px] leading-5 text-faint">
          <ShieldCheck class="mt-0.5 h-3.5 w-3.5 shrink-0" />
          <span>访问权限来自 Keycloak 的 permission-* realm 角色；职务仅用于展示。登录会话保存在本站内存中。</span>
        </p>
      </div>
      <p class="mt-3 text-center text-[10px] text-faint">Guild of Pioneers · guild-go</p>
    </div>
  </div>
</template>