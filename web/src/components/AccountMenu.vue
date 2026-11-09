<script setup lang="ts">
const emit = defineEmits<{ error: [message: string]; clearError: [] }>();
const route = useRoute();
const router = useRouter();
const { user, logout } = useAuth();
const accountOpen = ref(false);
const leaving = ref(false);
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
  emit("clearError");
  try {
    await logout();
    await router.replace({ name: "login" });
  } catch (err) {
    emit("error", err instanceof Error ? err.message : "退出失败，请重试");
  } finally {
    leaving.value = false;
  }
}

watch(
  () => route.fullPath,
  () => {
    accountOpen.value = false;
  },
);
</script>

<template>
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
        <ChevronDown aria-hidden="true" class="account-chevron" :size="14" />
      </span>
    </NButton>
  </NDropdown>
</template>

<style scoped>
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
@media (max-width: 639px) {
  .account-name,
  .account-chevron {
    display: none;
  }
}
</style>
