<script setup lang="ts">
import { computed, onMounted, ref, useSlots } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { LogOut, Menu, PanelRight, UserRound } from '@lucide/vue'
import NodeTree from '@/components/NodeTree.vue'
import UserAvatar from '@/components/UserAvatar.vue'
import { useAuth } from '@/composables/useAuth'
import { useNodeStore } from '@/composables/useNodeStore'

const slots = useSlots()
const route = useRoute()
const router = useRouter()
const { user, logout } = useAuth()
const { load } = useNodeStore()

const treeOpen = ref(false)
const railOpen = ref(false)
const menuOpen = ref(false)
const logoutError = ref<string | null>(null)
const hasRail = computed(() => Boolean(slots.rail))
const isHome = computed(() => route.name === 'home')

onMounted(() => {
  void load()
})

async function onLogout() {
  logoutError.value = null
  try {
    await logout()
    await router.replace({ name: 'login' })
  } catch (err) {
    logoutError.value = err instanceof Error ? err.message : '退出失败，请重试'
  }
}

function goHome() {
  treeOpen.value = false
  railOpen.value = false
  void router.push({ name: 'home' })
}

function onMenuToggle(event: Event) {
  menuOpen.value = (event.target as HTMLDetailsElement).open
}
</script>

<template>
  <div class="flex h-full min-h-0 flex-col bg-canvas">
    <header class="z-30 flex h-12 shrink-0 items-center gap-2 border-b border-line bg-surface px-3">
      <button class="icon-btn lg:hidden" aria-label="打开节点树" @click="treeOpen = true">
        <Menu class="h-4 w-4" />
      </button>

      <button class="flex items-center gap-2" type="button" @click="goHome">
        <span class="flex h-7 w-7 items-center justify-center rounded-md bg-accent text-[13px] font-bold text-white">公</span>
        <span class="text-[13px] font-semibold tracking-tight text-ink">公会</span>
      </button>

      <nav class="ml-3 hidden items-center gap-1 sm:flex">
        <RouterLink
          :to="{ name: 'home' }"
          class="rounded-md px-2 py-1 text-[12px] font-medium transition-colors"
          :class="isHome ? 'bg-accent-soft text-accent' : 'text-muted hover:bg-surface-2 hover:text-ink'"
        >
          总览
        </RouterLink>
        <RouterLink
          :to="{ name: 'account' }"
          class="rounded-md px-2 py-1 text-[12px] font-medium transition-colors"
          :class="route.name === 'account' ? 'bg-accent-soft text-accent' : 'text-muted hover:bg-surface-2 hover:text-ink'"
        >
          我的账号
        </RouterLink>
      </nav>

      <div class="ml-auto flex items-center gap-1">
        <button
          v-if="hasRail"
          class="icon-btn lg:hidden"
          aria-label="打开侧栏"
          @click="railOpen = true"
        >
          <PanelRight class="h-4 w-4" />
        </button>

        <details class="relative" @toggle="onMenuToggle">
          <summary class="flex cursor-pointer list-none items-center gap-1.5 rounded-md px-1.5 py-1 hover:bg-surface-2">
            <UserAvatar
              :name="user?.userName ?? '?'"
              :avatar="user?.avatar ?? null"
              size-class="h-6 w-6 text-[10px]"
            />
            <span class="hidden max-w-28 truncate text-[12px] font-medium text-ink-soft sm:block">
              {{ user?.userName ?? '…' }}
            </span>
          </summary>
          <div
            v-if="menuOpen"
            class="absolute right-0 z-50 mt-1 w-56 rounded-lg border border-line bg-surface p-1.5 shadow-lg"
          >
            <div class="border-b border-line px-2 py-1.5">
              <p class="truncate text-[12px] font-medium text-ink">{{ user?.userName }}</p>
              <p v-if="user?.email" class="truncate text-[11px] text-muted">{{ user.email }}</p>
              <p v-if="user?.permissions?.systemAdmin" class="mt-1 inline-flex rounded bg-accent-soft px-1.5 py-0.5 text-[10px] text-accent">系统管理员</p>
              <p v-else-if="user?.permissions?.manager" class="mt-1 inline-flex rounded bg-surface-3 px-1.5 py-0.5 text-[10px] text-muted">部门管理员</p>
            </div>
            <RouterLink
              :to="{ name: 'account' }"
              class="flex items-center gap-2 rounded-md px-2 py-1.5 text-[12px] text-ink-soft hover:bg-surface-2"
              @click="menuOpen = false"
            >
              <UserRound class="h-3.5 w-3.5" /> 我的账号
            </RouterLink>
            <button
              type="button"
              class="flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left text-[12px] text-ink-soft hover:bg-surface-2"
              @click="onLogout"
            >
              <LogOut class="h-3.5 w-3.5" /> 退出登录
            </button>
            <p v-if="logoutError" class="px-2 py-1 text-[11px] text-danger">{{ logoutError }}</p>
          </div>
        </details>
      </div>
    </header>

    <div class="relative flex min-h-0 flex-1">
      <div
        v-if="treeOpen"
        class="fixed inset-0 z-40 bg-ink/30 lg:hidden"
        @click="treeOpen = false"
      />
      <aside
        class="fixed bottom-0 left-0 top-12 z-40 flex w-[280px] flex-col border-r border-line bg-surface transition-transform lg:static lg:top-0 lg:z-0 lg:w-[262px] lg:translate-x-0"
        :class="treeOpen ? 'translate-x-0 shadow-xl' : '-translate-x-full lg:translate-x-0'"
      >
        <NodeTree @navigate="treeOpen = false" />
      </aside>

      <main class="min-w-0 flex-1 overflow-y-auto">
        <slot />
      </main>

      <div
        v-if="railOpen"
        class="fixed inset-0 z-40 bg-ink/30 lg:hidden"
        @click="railOpen = false"
      />
      <aside
        v-if="hasRail"
        class="fixed bottom-0 right-0 top-12 z-40 flex w-[300px] flex-col border-l border-line bg-surface transition-transform lg:static lg:top-0 lg:z-0 lg:translate-x-0"
        :class="railOpen ? 'translate-x-0 shadow-xl' : 'translate-x-full lg:translate-x-0'"
      >
        <div class="flex items-center justify-between border-b border-line px-3 py-2">
          <span class="rail-title">节点信息</span>
          <button class="icon-btn lg:hidden" aria-label="关闭侧栏" @click="railOpen = false">
            <PanelRight class="h-4 w-4" />
          </button>
        </div>
        <div class="min-h-0 flex-1 overflow-y-auto">
          <slot name="rail" />
        </div>
      </aside>
    </div>
  </div>
</template>