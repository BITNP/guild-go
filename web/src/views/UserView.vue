<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { Loader2, ShieldCheck } from '@lucide/vue'
import AppShell from '@/components/AppShell.vue'
import UserAvatar from '@/components/UserAvatar.vue'
import { useAuth } from '@/composables/useAuth'
import { fetchUser, type Profile } from '@/lib/api'
import { departmentLabel, errorMessage } from '@/lib/utils'

const route = useRoute()
const { user: currentUser } = useAuth()
const profile = ref<Profile | null>(null)
const loading = ref(true)
const error = ref<string | null>(null)

const subject = computed(() => String(route.params.subject ?? ''))
const isSelf = computed(() => subject.value === currentUser.value?.subject)
const display = computed(() => isSelf.value ? currentUser.value : profile.value)

async function load() {
  loading.value = true
  error.value = null
  if (isSelf.value && currentUser.value) {
    profile.value = currentUser.value
    loading.value = false
    return
  }
  try {
    profile.value = await fetchUser(subject.value)
  } catch (err) {
    profile.value = null
    error.value = errorMessage(err, '无法加载用户资料')
  } finally {
    loading.value = false
  }
}

onMounted(load)
watch(subject, load)
</script>

<template>
  <AppShell>
    <div class="mx-auto flex w-full max-w-3xl flex-col gap-4 p-4 lg:p-6">
      <header>
        <h1 class="text-lg font-semibold tracking-tight text-ink">{{ isSelf ? '我的账号' : '用户' }}</h1>
        <p class="mt-0.5 text-[12px] text-muted">资料、职务与权限由 BITNP Keycloak 管理。</p>
      </header>

      <p v-if="loading" class="flex items-center gap-2 text-[13px] text-muted"><Loader2 class="h-4 w-4 animate-spin" /> 正在加载…</p>
      <p v-else-if="error" class="rounded-md border border-danger/20 bg-danger-soft px-3 py-2 text-[12px] text-danger">{{ error }}</p>

      <template v-else-if="display">
        <section class="panel flex items-center gap-4 px-4 py-4">
          <UserAvatar :name="display.userName" :avatar="display.avatar" size-class="h-14 w-14 text-lg" />
          <div class="min-w-0">
            <h2 class="truncate text-lg font-semibold text-ink">{{ display.userName }}</h2>
            <p v-if="display.email" class="truncate text-[12px] text-muted">{{ display.email }}</p>
            <p class="mt-0.5 truncate text-[10px] text-faint">subject: {{ display.subject }}</p>
          </div>
        </section>

        <section class="panel px-4 py-3">
          <h2 class="section-label">行政职务</h2>
          <div v-if="display.positions.length" class="mt-2 flex flex-wrap gap-1.5">
            <span v-for="position in display.positions" :key="position.code" class="tag tag-accent">
              {{ position.label }}
            </span>
          </div>
          <p v-else class="mt-2 text-[12px] text-faint">没有行政职务，或 Keycloak Admin API 暂不可用。</p>
        </section>

        <section v-if="isSelf && display.permissions" class="panel px-4 py-3">
          <h2 class="section-label">权限</h2>
          <dl class="mt-2 grid grid-cols-1 gap-2 text-[12px] sm:grid-cols-2">
            <div class="flex items-center justify-between rounded border border-line bg-surface-2 px-2.5 py-1.5">
              <dt class="text-muted">系统管理员</dt>
              <dd class="font-medium text-ink">{{ display.permissions.systemAdmin ? '是' : '否' }}</dd>
            </div>
            <div class="flex items-center justify-between rounded border border-line bg-surface-2 px-2.5 py-1.5">
              <dt class="text-muted">部门管理员（可建根节点）</dt>
              <dd class="font-medium text-ink">{{ display.permissions.manager ? '是' : '否' }}</dd>
            </div>
            <div class="rounded border border-line bg-surface-2 px-2.5 py-1.5">
              <dt class="text-muted">部门访问</dt>
              <dd class="mt-0.5 font-medium text-ink">
                {{ display.permissions.accessDepartments.length
                  ? display.permissions.accessDepartments.map(departmentLabel).join('、')
                  : '—' }}
              </dd>
            </div>
            <div class="rounded border border-line bg-surface-2 px-2.5 py-1.5">
              <dt class="text-muted">部门管理</dt>
              <dd class="mt-0.5 font-medium text-ink">
                {{ display.permissions.managedDepartments.length
                  ? display.permissions.managedDepartments.map(departmentLabel).join('、')
                  : '—' }}
              </dd>
            </div>
          </dl>
        </section>

        <p class="flex items-start gap-1.5 rounded-md border border-line bg-surface-2 px-3 py-2 text-[10px] leading-5 text-muted">
          <ShieldCheck class="mt-0.5 h-3.5 w-3.5 shrink-0" />
          姓名、邮箱、头像和职务来自 Keycloak；权限来自已验证 ID token 的 realm 角色。修改资料请前往 BITNP 账号控制台，变更在下次登录后生效。
        </p>
      </template>
    </div>
  </AppShell>
</template>