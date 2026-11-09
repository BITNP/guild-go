<script setup lang="ts">
const props = withDefaults(
  defineProps<{ railTitle?: string; immersive?: boolean }>(),
  { railTitle: "节点信息", immersive: false },
);
const slots: Slots = useSlots();
const route = useRoute();
const router = useRouter();
const { user, logout } = useAuth();
const { load } = useNodeStore();
const { isDark, toggleTheme } = useTheme();
const treeOpen = ref(false);
const railOpen = ref(false);
const accountOpen = ref(false);
const leaving = ref(false);
const error = ref<string | null>(null);
const hasRail = computed<boolean>(
  () => !props.immersive && Boolean(slots.rail),
);
const menuOptions = [
  {
    key: "home",
    label: () => h(RouterLink, { to: { name: "home" } }, () => "总览"),
  },
  {
    key: "graph",
    label: () => h(RouterLink, { to: { name: "graph" } }, () => "关系图谱"),
  },
];
const accountOptions = computed(() => [
  {
    key: "identity",
    type: "render" as const,
    render: () =>
      h(
        NFlex,
        {
          vertical: true,
          size: 4,
          style: "padding:8px 12px;overflow-wrap:anywhere",
        },
        () => [
          h(NText, { strong: true }, () => user.value?.userName),
          user.value?.email
            ? h(
                NText,
                { depth: 3, style: "font-size:12px" },
                () => user.value?.email,
              )
            : null,
          user.value?.permissions?.systemAdmin ||
          user.value?.permissions?.manager
            ? h(
                NTag,
                {
                  size: "small",
                  bordered: false,
                  style: "align-self:flex-start;margin-top:4px",
                },
                () =>
                  user.value?.permissions?.systemAdmin
                    ? "系统管理员"
                    : "部门管理员",
              )
            : null,
        ],
      ),
  },
  { key: "divider", type: "divider" as const },
  { key: "account", label: "我的账号", icon: () => h(UserRound, { size: 16 }) },
  {
    key: "logout",
    label: leaving.value ? "退出中…" : "退出登录",
    disabled: leaving.value,
    icon: () => h(LogOut, { size: 16 }),
  },
]);

async function selectAccount(key: string) {
  if (key === "account") {
    void router.push({ name: "account" });

    return;
  }
  if (key !== "logout" || leaving.value) {
    return;
  }
  leaving.value = true;
  error.value = null;
  try {
    await logout();
    await router.replace({ name: "login" });
  } catch (err) {
    error.value = err instanceof Error ? err.message : "退出失败，请重试";
  } finally {
    leaving.value = false;
  }
}

watch(
  () => route.fullPath,
  () => {
    treeOpen.value = false;
    railOpen.value = false;
    accountOpen.value = false;
  },
);
onMounted(() => {
  if (!props.immersive) {
    void load();
  }
});
</script>

<template>
  <div class="application">
    <a class="skip-link" href="#main-content">跳到主要内容</a>
    <NLayoutHeader bordered class="application-header">
      <div class="header-navigation">
        <NButton
          v-if="!immersive"
          aria-label="打开节点树"
          class="tree-toggle"
          quaternary
          @click="treeOpen = true"
        >
          <Menu :size="18" />
        </NButton>
        <RouterLink class="brand" :to="{ name: 'home' }">
          <NAvatar class="brand-mark" color="#1f5fbf" :size="28">公</NAvatar>
          <strong class="brand-name">公会</strong>
        </RouterLink>
        <NMenu
          class="main-menu"
          mode="horizontal"
          :options="menuOptions"
          :value="String(route.name)"
        />
      </div>
      <NSpace align="center" :size="6" :wrap="false">
        <NButton
          :aria-label="isDark ? '切换到浅色主题' : '切换到深色主题'"
          :aria-pressed="isDark"
          class="theme-toggle"
          quaternary
          :title="isDark ? '切换到浅色主题' : '切换到深色主题'"
          @click="toggleTheme"
        >
          <Sun v-if="isDark" :size="17" />
          <Moon v-else :size="17" />
        </NButton>
        <NButton
          v-if="hasRail"
          :aria-label="`打开${railTitle}`"
          class="rail-toggle"
          quaternary
          @click="railOpen = true"
        >
          <PanelRight :size="18" />
        </NButton>
        <NDropdown
          v-model:show="accountOpen"
          :options="accountOptions"
          placement="bottom-end"
          trigger="click"
          :width="240"
          @select="selectAccount"
        >
          <NButton
            :aria-expanded="accountOpen"
            aria-haspopup="menu"
            aria-label="账号菜单"
            class="account-trigger"
            quaternary
          >
            <span class="account-trigger-content">
              <UserAvatar
                :name="user?.userName ?? ''"
                :size="28"
                :src="user?.avatar"
              />
              <span class="account-name">{{ user?.userName }}</span>
              <ChevronDown
                aria-hidden="true"
                class="account-chevron"
                :size="14"
              />
            </span>
          </NButton>
        </NDropdown>
      </NSpace>
    </NLayoutHeader>
    <div
      class="workspace"
      :class="{ 'has-rail': hasRail, 'is-immersive': immersive }"
    >
      <nav v-if="!immersive" aria-label="节点导航" class="sidebar">
        <NodeTree />
      </nav>
      <NScrollbar v-if="!immersive" class="content-scroll">
        <main id="main-content" tabindex="-1">
          <NAlert v-if="error" closable type="error" @close="error = null">
            {{ error }}
          </NAlert>
          <slot />
        </main>
      </NScrollbar>
      <main v-else id="main-content" class="content-immersive" tabindex="-1">
        <NAlert v-if="error" closable type="error" @close="error = null">
          {{ error }}
        </NAlert>
        <slot />
      </main>
      <aside v-if="hasRail" :aria-label="railTitle" class="rail">
        <h2 class="rail-heading">{{ railTitle }}</h2>
        <NScrollbar class="rail-scroll"><slot name="rail" /></NScrollbar>
      </aside>
    </div>
    <NDrawer
      placement="left"
      :show="treeOpen"
      :width="320"
      @update:show="(value) => (treeOpen = value)"
    >
      <NDrawerContent
        body-content-style="padding:0"
        closable
        :native-scrollbar="false"
        title="节点导航"
      >
        <NodeTree @navigate="treeOpen = false" />
      </NDrawerContent>
    </NDrawer>
    <NDrawer
      v-if="hasRail"
      placement="right"
      :show="railOpen"
      :width="320"
      @update:show="(value) => (railOpen = value)"
    >
      <NDrawerContent
        body-content-style="padding:0"
        closable
        :native-scrollbar="false"
        :title="railTitle"
      >
        <slot name="rail" />
      </NDrawerContent>
    </NDrawer>
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
  border-radius: 6px;
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
  .brand-name,
  .account-name,
  .account-chevron {
    display: none;
  }
  .application-header {
    padding: 0 8px;
    gap: 4px;
  }
  .header-navigation {
    gap: 4px;
    min-width: 0;
  }
  .main-menu :deep(.n-menu-item-content) {
    padding: 0 10px;
  }
  .main-menu {
    flex: none;
  }
}
@media (max-width: 399px) {
  .brand {
    display: none;
  }
}
</style>
