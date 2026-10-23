<script setup lang="ts">
import { computed, onMounted, ref, useSlots, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { LogOut, Menu, PanelRight, UserRound } from '@lucide/vue'
import BaseModal from '@/components/BaseModal.vue'
import NodeTree from '@/components/NodeTree.vue'
import UserAvatar from '@/components/UserAvatar.vue'
import { useAuth } from '@/composables/useAuth'
import { useNodeStore } from '@/composables/useNodeStore'

withDefaults(defineProps<{ railTitle?: string }>(), { railTitle: '节点信息' })

const slots = useSlots()
const route = useRoute()
const router = useRouter()
const { user, logout } = useAuth()
const { load } = useNodeStore()

const treeOpen = ref(false)
const railOpen = ref(false)
const accountMenu = ref<HTMLDetailsElement | null>(null)
const leaving = ref(false)
const logoutError = ref<string | null>(null)
const hasRail = computed(() => Boolean(slots.rail))
const isHome = computed(() => route.name === 'home')

onMounted(() => {
  void load()
})

async function onLogout() {
  if (leaving.value) return
  leaving.value = true
  logoutError.value = null
  try {
    await logout()
    await router.replace({ name: 'login' })
  } catch (err) {
    logoutError.value = err instanceof Error ? err.message : '退出失败，请重试'
  } finally {
    leaving.value = false
  }
}

function goHome() {
  treeOpen.value = false
  railOpen.value = false
  void router.push({ name: 'home' })
}

function closeMenu() {
  if (accountMenu.value) accountMenu.value.open = false
}

watch(() => route.fullPath, () => {
  treeOpen.value = false
  railOpen.value = false
  closeMenu()
})
</script>

<template>
  <div class="flex h-dvh min-h-0 flex-col bg-canvas">
    <a href="#main-content" class="sr-only focus:not-sr-only focus:fixed focus:left-3 focus:top-2 focus:z-50 focus:rounded focus:bg-surface focus:p-2">跳到主要内容</a>
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
          class="icon-btn xl:hidden"
          :aria-label="`打开${railTitle}`"
          @click="railOpen = true"
        >
          <PanelRight class="h-4 w-4" />
        </button>

        <details ref="accountMenu" class="relative" @keydown.esc.stop="closeMenu" @focusout="!accountMenu?.contains($event.relatedTarget as Node) && closeMenu()">
          <summary aria-label="账号菜单" class="flex cursor-pointer list-none items-center gap-1.5 rounded-md px-1.5 py-1 hover:bg-surface-2">
            <UserAvatar
              :name="user?.userName ?? '?'"
              :avatar="user?.avatar ?? null"
              size-class="h-6 w-6 text-xs"
            />
            <span class="hidden max-w-28 truncate text-[12px] font-medium text-ink-soft sm:block">
              {{ user?.userName ?? '…' }}
            </span>
          </summary>
          <div
            class="absolute right-0 z-50 mt-1 w-56 rounded-lg border border-line bg-surface p-1.5 shadow-md"
          >
            <div class="border-b border-line px-2 py-1.5">
              <p class="truncate text-[12px] font-medium text-ink">{{ user?.userName }}</p>
              <p v-if="user?.email" class="truncate text-xs text-muted">{{ user.email }}</p>
              <p v-if="user?.permissions?.systemAdmin" class="mt-1 inline-flex rounded bg-accent-soft px-1.5 py-0.5 text-xs text-accent">系统管理员</p>
              <p v-else-if="user?.permissions?.manager" class="mt-1 inline-flex rounded bg-surface-3 px-1.5 py-0.5 text-xs text-muted">部门管理员</p>
            </div>
            <RouterLink
              :to="{ name: 'account' }"
              class="flex items-center gap-2 rounded-md px-2 py-1.5 text-[12px] text-ink-soft hover:bg-surface-2"
              @click="closeMenu"
            >
              <UserRound class="h-3.5 w-3.5" /> 我的账号
            </RouterLink>
            <button
              type="button"
              class="flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left text-[12px] text-ink-soft hover:bg-surface-2"
              :disabled="leaving"
              @click="onLogout"
            >
              <LogOut class="h-3.5 w-3.5" /> {{ leaving ? '退出中…' : '退出登录' }}
            </button>
            <p v-if="logoutError" class="px-2 py-1 text-xs text-danger">{{ logoutError }}</p>
          </div>
        </details>
      </div>
    </header>

    <div class="flex min-h-0 flex-1">
      <aside aria-label="节点导航" class="hidden w-[248px] shrink-0 flex-col border-r border-line bg-surface lg:flex">
        <NodeTree />
      </aside>

      <main id="main-content" tabindex="-1" class="min-w-0 flex-1 overflow-y-auto">
        <slot />
      </main>

      <aside v-if="hasRail" :aria-label="railTitle" class="hidden w-[280px] shrink-0 flex-col border-l border-line bg-surface xl:flex">
        <div class="border-b border-line px-4 py-3">
          <h2 class="rail-title">{{ railTitle }}</h2>
        </div>
        <div class="min-h-0 flex-1 overflow-y-auto">
          <slot name="rail" />
        </div>
      </aside>
    </div>

    <BaseModal :open="treeOpen" title="节点导航" side="left" @close="treeOpen = false">
      <NodeTree @navigate="treeOpen = false" />
    </BaseModal>
    <BaseModal v-if="hasRail" :open="railOpen" :title="railTitle" side="right" @close="railOpen = false">
      <slot name="rail" />
    </BaseModal>
  </div>
</template>
