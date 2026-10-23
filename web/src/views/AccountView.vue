<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { Loader2, LogOut } from '@lucide/vue'
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
    <div class="mx-auto flex w-full max-w-2xl flex-col gap-4 p-4 lg:p-6">
      <header>
        <h1 class="text-lg font-semibold tracking-tight text-ink">我的账号</h1>
      </header>

      <div v-if="user" class="panel overflow-hidden">
        <section class="flex flex-wrap items-center gap-3 border-b border-line bg-surface-2 p-5">
          <UserAvatar :name="user.userName" :avatar="user.avatar" size-class="h-14 w-14 text-lg" />
          <div class="min-w-0 flex-1 basis-40">
            <h2 class="break-words text-lg font-semibold text-ink">{{ user.userName }}</h2>
            <p v-if="user.email" class="break-all text-[13px] text-muted">{{ user.email }}</p>
            <div v-if="user.positions.length" class="mt-2 flex flex-wrap gap-1.5" aria-label="职务">
              <span v-for="position in user.positions" :key="position.code" class="tag tag-accent">{{ position.label }}</span>
            </div>
            <p v-else class="mt-2 text-xs text-muted">暂无职务信息</p>
          </div>
        </section>

        <section v-if="user.permissions" class="p-5">
          <h2 class="section-label">权限</h2>
          <dl class="mt-3 grid grid-cols-1 gap-x-8 gap-y-2 text-[13px] sm:grid-cols-2">
            <div class="flex items-start justify-between gap-4 py-1.5">
              <dt class="text-muted">系统管理员</dt>
              <dd class="font-medium text-ink">{{ user.permissions.systemAdmin ? '是' : '否' }}</dd>
            </div>
            <div class="flex items-start justify-between gap-4 py-1.5">
              <dt class="text-muted">可创建根节点</dt>
              <dd class="font-medium text-ink">{{ user.permissions.manager ? '是' : '否' }}</dd>
            </div>
            <div class="py-1.5">
              <dt class="text-muted">部门访问</dt>
              <dd class="mt-0.5 font-medium text-ink">
                {{ user.permissions.accessDepartments.length
                  ? user.permissions.accessDepartments.map(departmentLabel).join('、')
                  : '—' }}
              </dd>
            </div>
            <div class="py-1.5">
              <dt class="text-muted">部门管理</dt>
              <dd class="mt-0.5 font-medium text-ink">
                {{ user.permissions.managedDepartments.length
                  ? user.permissions.managedDepartments.map(departmentLabel).join('、')
                  : '—' }}
              </dd>
            </div>
          </dl>
        </section>

        <footer class="flex flex-wrap items-center gap-3 border-t border-line px-5 py-4">
          <p class="min-w-0 flex-1 basis-56 text-xs leading-5 text-muted">退出公会后，网协通行证仍会保持登录。</p>
          <button type="button" class="btn text-danger" :disabled="leaving" @click="onLogout">
            <Loader2 v-if="leaving" class="h-3.5 w-3.5 animate-spin" />
            <LogOut v-else class="h-3.5 w-3.5" />
            {{ leaving ? '退出中…' : '退出登录' }}
          </button>
          <p v-if="error" class="w-full rounded-md border border-danger/20 bg-danger-soft px-3 py-2 text-xs text-danger" role="alert">{{ error }}</p>
        </footer>
      </div>
      <p v-else class="text-[13px] text-muted">未登录。</p>
    </div>
  </AppShell>
</template>
