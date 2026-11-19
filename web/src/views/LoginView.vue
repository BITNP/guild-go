<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  NAlert,
  NAvatar,
  NButton,
  NCard,
  NH1,
  NSpace,
  NText,
} from 'naive-ui'
import { useAuth } from '@/composables/useAuth'
import { safeReturnTo } from '@/lib/utils'
const route = useRoute(),
  router = useRouter(),
  { state, error: connectionError, refresh, login } = useAuth()
const returnTo = computed(() => safeReturnTo(route.query.redirect))
const error = computed(() => {
  const code = route.query.error
  if (code === 'provider_unavailable' || code === 'unavailable')
    return '登录服务暂时不可用，请稍后重试。'
  if (code === 'not_member')
    return '此账号暂无公会访问权限，请联系部门管理员开通。'
  if (code === 'no_roles') return '此账号尚未分配网协角色，请联系管理员。'
  if (code) return '登录已取消或无法完成，请重试。'
  return connectionError.value ? '无法连接登录服务，请稍后重试。' : ''
})
onMounted(async () => {
  if (await refresh()) await router.replace({ name: 'home' })
})
</script>
<template>
  <main class="login-stage">
    <NCard class="login-card">
      <NSpace vertical :size="24">
        <header class="login-header">
          <NAvatar :size="44" color="#1f5fbf" class="login-mark">公</NAvatar>
          <div>
            <NH1>公会</NH1>
            <NText depth="3">网协的工作协作空间</NText>
          </div>
        </header>
        <NAlert v-if="error" type="error">{{ error }}</NAlert>
        <NButton
          type="primary"
          size="large"
          block
          :disabled="state === 'active'"
          @click="login(returnTo)"
          >使用网协通行证登录</NButton
        >
      </NSpace>
    </NCard>
  </main>
</template>
<style scoped>
.login-stage {
  min-height: 100dvh;
  display: grid;
  place-items: center;
  padding: 24px;
  background: var(--guild-app-bg);
}
.login-card {
  max-width: 400px;
  width: 100%;
}
.login-header {
  display: flex;
  align-items: center;
  gap: 16px;
}
.login-header h1 {
  margin: 0 0 2px;
  font-size: 24px;
  line-height: 30px;
  font-weight: 650;
}
.login-mark {
  flex: none;
  color: white;
}
@media (max-width: 639px) {
  .login-stage {
    padding: 16px;
  }
}
</style>
