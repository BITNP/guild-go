<script setup lang="ts">
const route = useRoute();
const { user } = useAuth();
const subject = computed(() => String(route.params.subject ?? ""));
const isSelf = computed(() => subject.value === user.value?.subject);
const profile = ref<Profile | null>(null);
const loading = ref(true);
const error = ref<string | null>(null);
const display = computed(() => (isSelf.value ? user.value : profile.value));

async function load() {
  const requested = subject.value;
  loading.value = true;
  error.value = null;
  try {
    const result = isSelf.value ? user.value : await fetchUser(requested);
    if (requested === subject.value) {
      profile.value = result;
    }
  } catch (err) {
    if (requested === subject.value) {
      error.value = errorMessage(err, "无法加载用户资料");
    }
  } finally {
    if (requested === subject.value) {
      loading.value = false;
    }
  }
}

watch(subject, load, { immediate: true });
</script>

<template>
  <AppShell>
    <section
      :aria-busy="loading"
      class="user-page"
      :inert="loading || undefined"
    >
      <NH1>{{ isSelf ? "我的账号" : "用户资料" }}</NH1>
      <NAlert v-if="error" type="error">
        {{ error }}
        <NButton text @click="load">重试</NButton>
      </NAlert>
      <ProfileSummary
        v-else-if="display"
        :profile="display"
        :show-permissions="isSelf"
      >
        <NText v-if="isSelf" depth="3">
          如需修改资料，请前往网协通行证，修改后重新登录。
        </NText>
      </ProfileSummary>
    </section>
  </AppShell>
</template>

<style scoped>
.user-page {
  max-width: 760px;
  margin: auto;
  padding: 24px;
}
.user-page h1 {
  font-size: var(--guild-page-title-size);
  font-weight: 650;
}
@media (max-width: 639px) {
  .user-page {
    padding: 16px;
  }
}
</style>
