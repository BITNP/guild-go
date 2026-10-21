<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { Loader2, LogOut, ShieldCheck } from '@lucide/vue'
import AppShell from '@/components/AppShell.vue'
import UserAvatar from '@/components/UserAvatar.vue'
import { useAuth } from '@/composables/useAuth'
import { departmentLabel } from '@/lib/utils'

const router = useRouter()
const { user, logout } = useAuth()
const leaving = ref(false)
const error = ref<string | null>(null)

async function onLogout() {
  leaving.value = true
  error.value = null
  try {
    await logout()
    await router.replace({ name: 'login' })
  } catch (err) {
    error.value = err instanceof Error ? err.message : '退出失败，请重试'
  } finally {
    leaving.value = false
  }
}
</script>

<template>
  <AppShell>
    <div class="mx-auto flex w-full max-w-3xl flex-col gap-4 p-4 lg:p-6">
      <header>
        <h1 class="text-lg font-semibold tracking-tight text-ink">我的账号</h1>
        <p class="mt-0.5 text-[12px] text-muted">本站只保存 Keycloak subject；显示资料来自登录 token 与 Keycloak Admin API。</p>
      </header>

      <template v-if="user">
        <section class="panel flex items-center gap-4 px-4 py-4">
          <UserAvatar :name="user.userName" :avatar="user.avatar" size-class="h-14 w-14 text-lg" />
          <div class="min-w-0 flex-1">
            <h2 class="truncate text-lg font-semibold text-ink">{{ user.userName }}</h2>
            <p v-if="user.email" class="truncate text-[12px] text-muted">{{ user.email }}</p>
            <p class="mt-0.5 truncate text-[10px] text-faint">{{ user.subject }}</p>
          </div>
          <button type="button" class="btn text-danger" :disabled="leaving" @click="onLogout">
            <Loader2 v-if="leaving" class="h-3.5 w-3.5 animate-spin" />
            <LogOut v-else class="h-3.5 w-3.5" />
            {{ leaving ? '退出中…' : '退出登录' }}
          </button>
        </section>

        <p v-if="error" class="rounded-md border border-danger/20 bg-danger-soft px-3 py-2 text-[12px] text-danger">{{ error }}</p>

        <section class="panel px-4 py-3">
          <h2 class="section-label">职务</h2>
          <div v-if="user.positions.length" class="mt-2 flex flex-wrap gap-1.5">
            <span v-for="position in user.positions" :key="position.code" class="tag tag-accent">{{ position.label }}</span>
          </div>
          <p v-else class="mt-2 text-[12px] text-faint">没有行政职务。</p>
        </section>

        <section v-if="user.permissions" class="panel px-4 py-3">
          <h2 class="section-label">权限</h2>
          <dl class="mt-2 grid grid-cols-1 gap-2 text-[12px] sm:grid-cols-2">
            <div class="flex items-center justify-between rounded border border-line bg-surface-2 px-2.5 py-1.5">
              <dt class="text-muted">系统管理员</dt>
              <dd class="font-medium text-ink">{{ user.permissions.systemAdmin ? '是' : '否' }}</dd>
            </div>
            <div class="flex items-center justify-between rounded border border-line bg-surface-2 px-2.5 py-1.5">
              <dt class="text-muted">部门管理员</dt>
              <dd class="font-medium text-ink">{{ user.permissions.manager ? '是' : '否' }}</dd>
            </div>
            <div class="rounded border border-line bg-surface-2 px-2.5 py-1.5">
              <dt class="text-muted">部门访问</dt>
              <dd class="mt-0.5 font-medium text-ink">
                {{ user.permissions.accessDepartments.length
                  ? user.permissions.accessDepartments.map(departmentLabel).join('、')
                  : '—' }}
              </dd>
            </div>
            <div class="rounded border border-line bg-surface-2 px-2.5 py-1.5">
              <dt class="text-muted">部门管理</dt>
              <dd class="mt-0.5 font-medium text-ink">
                {{ user.permissions.managedDepartments.length
                  ? user.permissions.managedDepartments.map(departmentLabel).join('、')
                  : '—' }}
              </dd>
            </div>
          </dl>
        </section>

        <p class="flex items-start gap-1.5 rounded-md border border-line bg-surface-2 px-3 py-2 text-[10px] leading-5 text-muted">
          <ShieldCheck class="mt-0.5 h-3.5 w-3.5 shrink-0" />
          退出只结束本站会话，不会退出 Keycloak。站点会话保存在服务端内存中，服务重启后需要重新登录。
        </p>
      </template>
      <p v-else class="text-[13px] text-muted">未登录。</p>
    </div>
  </AppShell>
</template>