<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { NAlert, NButton, NEmpty, NH1, NSpace, NText } from 'naive-ui'
import AppShell from '@/components/AppShell.vue'
import ProfileSummary from '@/components/ProfileSummary.vue'
import { useAuth } from '@/composables/useAuth'
import { errorMessage } from '@/lib/utils'
const router = useRouter(),
  { user, logout } = useAuth()
const leaving = ref(false),
  error = ref<string | null>(null)
async function onLogout() {
  if (leaving.value) return
  leaving.value = true
  error.value = null
  try {
    await logout()
    await router.replace({ name: 'login' })
  } catch (err) {
    error.value = errorMessage(err, '退出失败，请重试')
  } finally {
    leaving.value = false
  }
}
</script>
<template>
  <AppShell
    ><section class="account-page">
      <NH1>我的账号</NH1
      ><ProfileSummary v-if="user" :profile="user" show-permissions
        ><NText depth="3"
          >如需修改资料，请前往网协通行证，修改后重新登录。</NText
        ><NAlert v-if="error" type="error">{{ error }}</NAlert
        ><NSpace justify="space-between" align="center"
          ><NText depth="3">退出公会后，网协通行证仍会保持登录。</NText
          ><NButton type="error" secondary :loading="leaving" @click="onLogout"
            >退出登录</NButton
          ></NSpace
        ></ProfileSummary
      ><NEmpty v-else description="未登录" /></section
  ></AppShell>
</template>
<style scoped>
.account-page {
  max-width: 760px;
  margin: auto;
  padding: 24px;
}
.account-page h1 {
  font-size: var(--guild-page-title-size);
  font-weight: 650;
}
@media (max-width: 639px) {
  .account-page {
    padding: 16px;
  }
}
</style>
