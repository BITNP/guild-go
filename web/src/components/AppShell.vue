<script setup lang="ts">
import { computed, h, onMounted, ref, useSlots, watch, type Slots } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import {
  NAlert,
  NAvatar,
  NButton,
  NDrawer,
  NDrawerContent,
  NDropdown,
  NFlex,
  NLayoutHeader,
  NMenu,
  NScrollbar,
  NSpace,
  NTag,
  NText,
} from 'naive-ui'
import {
  ChevronDown,
  LogOut,
  Menu,
  Moon,
  PanelRight,
  Sun,
  UserRound,
} from '@lucide/vue'
import NodeTree from '@/components/NodeTree.vue'
import UserAvatar from '@/components/UserAvatar.vue'
import { useAuth } from '@/composables/useAuth'
import { useNodeStore } from '@/composables/useNodeStore'
import { useTheme } from '@/composables/useTheme'
const props = withDefaults(
  defineProps<{ railTitle?: string; immersive?: boolean }>(),
  { railTitle: '节点信息', immersive: false },
)
const slots: Slots = useSlots(),
  route = useRoute(),
  router = useRouter()
const { user, logout } = useAuth(),
  { load } = useNodeStore()
const { isDark, toggleTheme } = useTheme()
const treeOpen = ref(false),
  railOpen = ref(false),
  accountOpen = ref(false),
  leaving = ref(false),
  error = ref<string | null>(null)
const hasRail = computed<boolean>(() => !props.immersive && Boolean(slots.rail))
const menuOptions = [
  {
    key: 'home',
    label: () => h(RouterLink, { to: { name: 'home' } }, () => '总览'),
  },
  {
    key: 'graph',
    label: () => h(RouterLink, { to: { name: 'graph' } }, () => '关系图谱'),
  },
  {
    key: 'account',
    label: () => h(RouterLink, { to: { name: 'account' } }, () => '我的账号'),
  },
]
const accountOptions = computed(() => [
  {
    key: 'identity',
    type: 'render' as const,
    render: () =>
      h(
        NFlex,
        {
          vertical: true,
          size: 4,
          style: 'padding:8px 12px;overflow-wrap:anywhere',
        },
        () => [
          h(NText, { strong: true }, () => user.value?.userName),
          user.value?.email
            ? h(
                NText,
                { depth: 3, style: 'font-size:12px' },
                () => user.value?.email,
              )
            : null,
          user.value?.permissions?.systemAdmin || user.value?.permissions?.manager
            ? h(
                NTag,
                {
                  size: 'small',
                  bordered: false,
                  style: 'align-self:flex-start;margin-top:4px',
                },
                () =>
                  user.value?.permissions?.systemAdmin
                    ? '系统管理员'
                    : '部门管理员',
              )
            : null,
        ],
      ),
  },
  { key: 'divider', type: 'divider' as const },
  { key: 'account', label: '我的账号', icon: () => h(UserRound, { size: 16 }) },
  {
    key: 'logout',
    label: leaving.value ? '退出中…' : '退出登录',
    disabled: leaving.value,
    icon: () => h(LogOut, { size: 16 }),
  },
])
async function selectAccount(key: string) {
  if (key === 'account') {
    void router.push({ name: 'account' })
    return
  }
  if (key !== 'logout' || leaving.value) return
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
watch(
  () => route.fullPath,
  () => {
    treeOpen.value = false
    railOpen.value = false
    accountOpen.value = false
  },
)
onMounted(() => {
  if (!props.immersive) void load()
})
</script>
<template>
  <div class="application">
    <a href="#main-content" class="skip-link">跳到主要内容</a>
    <NLayoutHeader bordered class="application-header"
      ><div class="header-navigation">
        <NButton
          v-if="!immersive"
          quaternary
          class="tree-toggle"
          aria-label="打开节点树"
          @click="treeOpen = true"
          ><Menu :size="18" /></NButton
        ><RouterLink class="brand" :to="{ name: 'home' }"
          ><NAvatar :size="28" color="#1f5fbf" class="brand-mark">公</NAvatar
          ><strong>公会</strong></RouterLink
        ><NMenu
          class="main-menu"
          mode="horizontal"
          :value="String(route.name)"
          :options="menuOptions"
        />
      </div>
      <NSpace align="center" :wrap="false" :size="6"
        ><NButton
          quaternary
          class="theme-toggle"
          :aria-label="isDark ? '切换到浅色主题' : '切换到深色主题'"
          :aria-pressed="isDark"
          :title="isDark ? '切换到浅色主题' : '切换到深色主题'"
          @click="toggleTheme"
        >
          <Sun v-if="isDark" :size="17" />
          <Moon v-else :size="17" />
        </NButton
        ><NButton
          v-if="hasRail"
          quaternary
          class="rail-toggle"
          :aria-label="`打开${railTitle}`"
          @click="railOpen = true"
          ><PanelRight :size="18" /></NButton
        ><NDropdown
          v-model:show="accountOpen"
          trigger="click"
          placement="bottom-end"
          :width="240"
          :options="accountOptions"
          @select="selectAccount"
        >
          <NButton
            quaternary
            class="account-trigger"
            aria-label="账号菜单"
            aria-haspopup="menu"
            :aria-expanded="accountOpen"
          >
            <span class="account-trigger-content">
              <UserAvatar
                :name="user?.userName ?? ''"
                :size="28"
                :src="user?.avatar"
              />
              <span class="account-name">{{ user?.userName }}</span>
              <ChevronDown :size="14" class="account-chevron" aria-hidden="true" />
            </span>
          </NButton>
        </NDropdown
        ></NSpace
      ></NLayoutHeader
    >
    <div
      class="workspace"
      :class="{ 'has-rail': hasRail, 'is-immersive': immersive }"
    >
      <nav v-if="!immersive" class="sidebar" aria-label="节点导航"><NodeTree /></nav>
      <NScrollbar v-if="!immersive" class="content-scroll"
        ><main id="main-content" tabindex="-1">
          <NAlert v-if="error" type="error" closable @close="error = null">{{
            error
          }}</NAlert
          ><slot /></main
      ></NScrollbar>
      <main v-else id="main-content" tabindex="-1" class="content-immersive">
        <NAlert v-if="error" type="error" closable @close="error = null">{{
          error
        }}</NAlert
        ><slot />
      </main>
      <aside v-if="hasRail" class="rail" :aria-label="railTitle">
        <h2 class="rail-heading">{{ railTitle }}</h2>
        <NScrollbar class="rail-scroll"><slot name="rail" /></NScrollbar>
      </aside>
    </div>
    <NDrawer
      :show="treeOpen"
      placement="left"
      :width="320"
      @update:show="(value) => (treeOpen = value)"
      ><NDrawerContent
        title="节点导航"
        closable
        :native-scrollbar="false"
        body-content-style="padding:0"
        ><NodeTree @navigate="treeOpen = false" /></NDrawerContent
    ></NDrawer>
    <NDrawer
      v-if="hasRail"
      :show="railOpen"
      placement="right"
      :width="320"
      @update:show="(value) => (railOpen = value)"
      ><NDrawerContent
        :title="railTitle"
        closable
        :native-scrollbar="false"
        body-content-style="padding:0"
        ><slot name="rail" /></NDrawerContent
    ></NDrawer>
  </div>
</template>
<style scoped>
.application {
  height: 100dvh;
  display: grid;
  grid-template-rows: 52px minmax(0, 1fr);
  background: var(--guild-app-bg);
}
.application-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 16px;
  gap: 12px;
  background: var(--guild-header-bg);
  backdrop-filter: blur(14px);
}
.header-navigation,
.brand {
  display: flex;
  align-items: center;
  gap: 12px;
}
.brand {
  color: inherit;
  text-decoration: none;
  white-space: nowrap;
}
.brand-mark {
  color: white;
}
.account-trigger {
  height: 36px;
  padding: 0 8px;
}
.account-trigger-content {
  display: inline-flex;
  align-items: center;
  gap: 8px;
}
.account-name {
  max-width: 160px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-weight: 500;
}
.account-chevron {
  color: var(--guild-muted);
}
.workspace {
  display: grid;
  grid-template-columns: 304px minmax(0, 1fr);
  min-height: 0;
}
.workspace.has-rail {
  grid-template-columns: 304px minmax(0, 1fr) 280px;
}
.workspace.is-immersive {
  grid-template-columns: minmax(0, 1fr);
}
.content-immersive {
  display: flex;
  flex-direction: column;
  height: 100%;
  min-height: 0;
  overflow: hidden;
}
.sidebar {
  background: var(--guild-surface);
  border-right: 1px solid var(--guild-border);
  min-height: 0;
}
.content-scroll,
.rail-scroll {
  height: 100%;
}
.rail {
  background: var(--guild-surface);
  border-left: 1px solid var(--guild-border);
  display: flex;
  flex-direction: column;
  min-height: 0;
}
.rail-heading {
  margin: 0;
  padding: 12px 16px;
  border-bottom: 1px solid var(--guild-border);
  font-size: 14px;
  line-height: 20px;
  font-weight: 600;
  background: var(--guild-surface-subtle);
}
.rail-scroll {
  flex: 1;
  min-height: 0;
}
.tree-toggle,
.rail-toggle {
  display: none;
}
.skip-link {
  position: fixed;
  top: -100px;
  left: 16px;
  z-index: 3000;
  background: var(--guild-surface);
  padding: 8px;
  color: var(--guild-primary);
}
.skip-link:focus {
  top: 8px;
}
@media (max-width: 1279px) {
  .workspace,
  .workspace.has-rail {
    grid-template-columns: 280px minmax(0, 1fr);
  }
  .rail {
    display: none;
  }
  .rail-toggle {
    display: inline-flex;
  }
}
@media (max-width: 1023px) {
  .workspace,
  .workspace.has-rail {
    grid-template-columns: minmax(0, 1fr);
  }
  .sidebar {
    display: none;
  }
  .tree-toggle {
    display: inline-flex;
  }
}
@media (max-width: 639px) {
  .main-menu,
  .account-name,
  .account-chevron {
    display: none;
  }
  .application-header {
    padding: 0 8px;
  }
}
</style>
