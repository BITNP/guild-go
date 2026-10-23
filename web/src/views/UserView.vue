<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { Loader2 } from '@lucide/vue'
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
    <div class="mx-auto flex w-full max-w-2xl flex-col gap-4 p-4 lg:p-6">
      <header>
        <h1 class="text-lg font-semibold tracking-tight text-ink">{{ isSelf ? '我的账号' : '用户资料' }}</h1>
      </header>

      <p v-if="loading" class="flex items-center gap-2 text-[13px] text-muted"><Loader2 class="h-4 w-4 animate-spin" /> 正在加载…</p>
      <p v-else-if="error" class="rounded-md border border-danger/20 bg-danger-soft px-3 py-2 text-[12px] text-danger" role="alert">{{ error }} <button type="button" class="ml-2 underline" @click="load">重试</button></p>

      <div v-else-if="display" class="panel overflow-hidden">
        <section class="flex flex-wrap items-center gap-3 p-5" :class="isSelf ? 'border-b border-line bg-surface-2' : ''">
          <UserAvatar :name="display.userName" :avatar="display.avatar" size-class="h-14 w-14 text-lg" />
          <div class="min-w-0 flex-1 basis-40">
            <h2 class="break-words text-lg font-semibold text-ink">{{ display.userName }}</h2>
            <p v-if="display.email" class="break-all text-[13px] text-muted">{{ display.email }}</p>
            <div v-if="display.positions.length" class="mt-2 flex flex-wrap gap-1.5" aria-label="职务">
              <span v-for="position in display.positions" :key="position.code" class="tag tag-accent">
                {{ position.label }}
              </span>
            </div>
            <p v-else class="mt-2 text-xs text-muted">暂无职务信息</p>
          </div>
        </section>

        <section v-if="isSelf && display.permissions" class="p-5">
          <h2 class="section-label">权限</h2>
          <dl class="mt-3 grid grid-cols-1 gap-x-8 gap-y-2 text-[13px] sm:grid-cols-2">
            <div class="flex items-start justify-between gap-4 py-1.5">
              <dt class="text-muted">系统管理员</dt>
              <dd class="font-medium text-ink">{{ display.permissions.systemAdmin ? '是' : '否' }}</dd>
            </div>
            <div class="flex items-start justify-between gap-4 py-1.5">
              <dt class="text-muted">可创建根节点</dt>
              <dd class="font-medium text-ink">{{ display.permissions.manager ? '是' : '否' }}</dd>
            </div>
            <div class="py-1.5">
              <dt class="text-muted">部门访问</dt>
              <dd class="mt-0.5 font-medium text-ink">
                {{ display.permissions.accessDepartments.length
                  ? display.permissions.accessDepartments.map(departmentLabel).join('、')
                  : '—' }}
              </dd>
            </div>
            <div class="py-1.5">
              <dt class="text-muted">部门管理</dt>
              <dd class="mt-0.5 font-medium text-ink">
                {{ display.permissions.managedDepartments.length
                  ? display.permissions.managedDepartments.map(departmentLabel).join('、')
                  : '—' }}
              </dd>
            </div>
          </dl>
        </section>

        <p v-if="isSelf" class="border-t border-line px-5 py-4 text-xs leading-5 text-muted">如需修改资料，请前往网协通行证，修改后重新登录。</p>
      </div>
    </div>
  </AppShell>
</template>
