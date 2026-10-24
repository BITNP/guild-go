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
  NLayoutHeader,
  NMenu,
  NScrollbar,
  NSpace,
  NThing,
} from 'naive-ui'
import { Menu, PanelRight, X } from '@lucide/vue'
import NodeTree from '@/components/NodeTree.vue'
import { useAuth } from '@/composables/useAuth'
import { useNodeStore } from '@/composables/useNodeStore'
withDefaults(defineProps<{ railTitle?: string }>(), { railTitle: '节点信息' })
const slots: Slots = useSlots(),
  route = useRoute(),
  router = useRouter()
const { user, logout } = useAuth(),
  { load } = useNodeStore()
const treeOpen = ref(false),
  railOpen = ref(false),
  leaving = ref(false),
  error = ref<string | null>(null)
const hasRail = computed<boolean>(() => Boolean(slots.rail))
const menuOptions = [
  {
    key: 'home',
    label: () => h(RouterLink, { to: { name: 'home' } }, () => '总览'),
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
        NThing,
        {
          title: user.value?.userName,
          description: user.value?.email ?? '',
          style: 'padding:8px 16px;max-width:280px;overflow-wrap:anywhere',
        },
        () =>
          user.value?.permissions?.systemAdmin
            ? '系统管理员'
            : user.value?.permissions?.manager
              ? '部门管理员'
              : '',
      ),
  },
  { key: 'divider', type: 'divider' as const },
  { key: 'account', label: '我的账号' },
  {
    key: 'logout',
    label: leaving.value ? '退出中…' : '退出登录',
    disabled: leaving.value,
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
  },
)
onMounted(() => void load())
</script>
<template>
  <div class="application">
    <a href="#main-content" class="skip-link">跳到主要内容</a>
    <NLayoutHeader bordered class="application-header"
      ><div class="header-navigation">
        <NButton
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
          v-if="hasRail"
          quaternary
          class="rail-toggle"
          :aria-label="`打开${railTitle}`"
          @click="railOpen = true"
          ><PanelRight :size="18" /></NButton
        ><NDropdown
          trigger="click"
          :options="accountOptions"
          @select="selectAccount"
          ><NButton quaternary aria-label="账号菜单"
            ><NSpace align="center" :size="8"
              ><NAvatar round :size="26" :src="user?.avatar ?? undefined">{{
                user?.userName.slice(0, 1)
              }}</NAvatar
              ><span class="account-name">{{ user?.userName }}</span></NSpace
            ></NButton
          ></NDropdown
        ></NSpace
      ></NLayoutHeader
    >
    <div class="workspace" :class="{ 'has-rail': hasRail }">
      <nav class="sidebar" aria-label="节点导航"><NodeTree /></nav>
      <NScrollbar class="content-scroll"
        ><main id="main-content" tabindex="-1">
          <NAlert v-if="error" type="error" closable @close="error = null">{{
            error
          }}</NAlert
          ><slot /></main
      ></NScrollbar>
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
        :native-scrollbar="false"
        body-content-style="padding:0"
        header-style="padding:12px 16px"
        ><template #header
          ><div class="drawer-heading">
            <h2>节点导航</h2>
            <NButton
              quaternary
              circle
              aria-label="关闭节点导航"
              @click="treeOpen = false"
              ><X :size="18"
            /></NButton></div></template
        ><NodeTree @navigate="treeOpen = false" /></NDrawerContent
    ></NDrawer>
    <NDrawer
      v-if="hasRail"
      :show="railOpen"
      placement="right"
      :width="320"
      @update:show="(value) => (railOpen = value)"
      ><NDrawerContent
        :native-scrollbar="false"
        body-content-style="padding:0"
        header-style="padding:12px 16px"
        ><template #header
          ><div class="drawer-heading">
            <h2>{{ railTitle }}</h2>
            <NButton
              quaternary
              circle
              :aria-label="`关闭${railTitle}`"
              @click="railOpen = false"
              ><X :size="18"
            /></NButton></div></template
        ><slot name="rail" /></NDrawerContent
    ></NDrawer>
  </div>
</template>
<style scoped>
.application {
  height: 100dvh;
  display: grid;
  grid-template-rows: 52px minmax(0, 1fr);
  background: #f3f5f8;
}
.application-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 16px;
  gap: 12px;
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
.workspace {
  display: grid;
  grid-template-columns: 248px minmax(0, 1fr);
  min-height: 0;
}
.workspace.has-rail {
  grid-template-columns: 248px minmax(0, 1fr) 280px;
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
  background: white;
  padding: 8px;
  color: var(--guild-primary);
}
.skip-link:focus {
  top: 8px;
}
.drawer-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  width: 100%;
}
.drawer-heading h2 {
  margin: 0;
  font-size: 16px;
}
@media (max-width: 1279px) {
  .workspace.has-rail {
    grid-template-columns: 248px minmax(0, 1fr);
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
  .account-name {
    display: none;
  }
  .application-header {
    padding: 0 8px;
  }
}
</style>
