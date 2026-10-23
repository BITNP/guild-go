<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ArrowRight, ListTree, MessageSquare, Users } from '@lucide/vue'
import { useAuth } from '@/composables/useAuth'
import { safeReturnTo } from '@/lib/utils'

const route = useRoute()
const router = useRouter()
const { state, error: connectionError, refresh, login } = useAuth()

const returnTo = computed(() => safeReturnTo(route.query.redirect))
const errorMessage = computed(() => {
  const code = route.query.error
  if (code === 'provider_unavailable' || code === 'unavailable') {
    return '登录服务暂时不可用，请稍后重试。'
  }
  if (code === 'not_member') {
    return '此账号暂无公会访问权限，请联系部门管理员开通。'
  }
  if (code === 'no_roles') {
    return '未能获取账号权限，请联系管理员检查登录配置。'
  }
  if (code) {
    return '登录已取消或无法完成，请重试。'
  }
  return connectionError.value ? '无法连接登录服务，请稍后重试。' : ''
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
  <main class="flex min-h-full items-center justify-center bg-canvas p-4 sm:p-8">
    <div class="panel grid w-full max-w-3xl overflow-hidden sm:grid-cols-2">
      <section class="border-b border-line bg-accent-soft/50 p-6 sm:border-b-0 sm:border-r sm:p-8" aria-labelledby="product-name">
        <div class="flex items-center gap-3">
          <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-lg bg-accent text-xl font-bold text-white" aria-hidden="true">公</div>
          <div>
            <h1 id="product-name" class="text-2xl font-semibold tracking-tight text-ink">公会</h1>
            <p class="mt-0.5 text-xs text-muted">Guild of Pioneers</p>
          </div>
        </div>
        <p class="mt-5 text-sm text-ink-soft">网协的工作协作空间</p>
        <ul class="mt-4 flex flex-col gap-3 text-[13px] text-ink-soft">
          <li class="flex items-center gap-2.5"><ListTree class="h-4 w-4 shrink-0 text-accent" aria-hidden="true" />拆分节点，安排工作</li>
          <li class="flex items-center gap-2.5"><Users class="h-4 w-4 shrink-0 text-accent" aria-hidden="true" />分配负责人和成员</li>
          <li class="flex items-center gap-2.5"><MessageSquare class="h-4 w-4 shrink-0 text-accent" aria-hidden="true" />在评论中同步进展</li>
        </ul>
      </section>

      <section class="flex flex-col justify-center p-6 sm:p-8" aria-labelledby="login-title">
        <h2 id="login-title" class="text-lg font-semibold text-ink">登录公会</h2>

        <p
          v-if="errorMessage"
          class="mt-4 rounded-md border border-danger/20 bg-danger-soft px-3 py-2 text-[12px] leading-5 text-danger"
          role="alert"
        >
          {{ errorMessage }}
        </p>

        <button
          type="button"
          class="btn btn-primary mt-5 min-h-11 w-full text-sm"
          :disabled="state === 'active'"
          @click="login(returnTo)"
        >
          使用网协通行证登录
          <ArrowRight class="h-4 w-4" />
        </button>
        <p class="mt-3 text-xs leading-5 text-muted">将跳转至网协通行证完成登录。</p>
      </section>
    </div>
  </main>
</template>
