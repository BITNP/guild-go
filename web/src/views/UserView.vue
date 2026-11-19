<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { NAlert, NButton, NH1, NSpin } from 'naive-ui'
import AppShell from '@/components/AppShell.vue'
import ProfileSummary from '@/components/ProfileSummary.vue'
import { useAuth } from '@/composables/useAuth'
import { fetchUser, type Profile } from '@/lib/api'
import { errorMessage } from '@/lib/utils'
const route = useRoute(),
  router = useRouter(),
  { user } = useAuth()
const subject = computed(() => String(route.params.subject ?? '')),
  isSelf = computed(() => !!user.value?.subject && subject.value === user.value.subject)
const profile = ref<Profile | null>(null),
  loading = ref(true),
  error = ref<string | null>(null)
async function load() {
  if (isSelf.value) {
    await router.replace({ name: 'account' })
    return
  }
  const requested = subject.value
  loading.value = true
  error.value = null
  try {
    const result = await fetchUser(requested)
    if (requested === subject.value) profile.value = result
  } catch (err) {
    if (requested === subject.value)
      error.value = errorMessage(err, '无法加载用户资料')
  } finally {
    if (requested === subject.value) loading.value = false
  }
}
watch([subject, isSelf], load, { immediate: true })
</script>
<template>
  <AppShell
    ><section class="user-page" :aria-busy="loading" :inert="loading || undefined">
      <NH1>用户资料</NH1
      ><NSpin :show="loading"
        ><NAlert
          v-if="error"
          type="error"
          >{{ error }} <NButton text @click="load">重试</NButton></NAlert
        ><div v-else-if="loading && !profile" class="user-placeholder" /><ProfileSummary
          v-else-if="profile"
          :profile="profile"
          :show-permissions="false"
        /></NSpin
      >
    </section></AppShell
  >
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
.user-placeholder {
  min-height: 180px;
}
@media (max-width: 639px) {
  .user-page {
    padding: 16px;
  }
}
</style>
