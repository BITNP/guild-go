<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  NAlert,
  NAvatar,
  NButton,
  NCard,
  NGrid,
  NGi,
  NH1,
  NH2,
  NList,
  NListItem,
  NSpace,
  NText,
} from 'naive-ui'
import { ListTree, Users, MessageSquare } from '@lucide/vue'
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
    <NCard class="login-card" :content-style="{ padding: 0 }"
      ><NGrid cols="1 s:2" responsive="screen"
        ><NGi
          ><section class="login-product">
            <NSpace align="center" :size="16"
              ><NAvatar :size="48" color="#1f5fbf" class="login-mark"
                >公</NAvatar
              >
              <div>
                <NH1>公会</NH1><NText depth="3">Guild of Pioneers</NText>
              </div></NSpace
            >
            <p>网协的工作协作空间</p>
            <NList :show-divider="false" class="product-features"
              ><NListItem
                ><template #prefix><ListTree :size="18" /></template
                >拆分节点，安排工作</NListItem
              ><NListItem
                ><template #prefix><Users :size="18" /></template
                >分配负责人和成员</NListItem
              ><NListItem
                ><template #prefix><MessageSquare :size="18" /></template
                >在评论中同步进展</NListItem
              ></NList
            >
          </section></NGi
        ><NGi
          ><section class="login-action">
            <NSpace vertical :size="20"
              ><NH2>登录公会</NH2
              ><NAlert v-if="error" type="error">{{ error }}</NAlert
              ><NButton
                type="primary"
                size="large"
                block
                :disabled="state === 'active'"
                @click="login(returnTo)"
                >使用网协通行证登录</NButton
              ><NText depth="3">将跳转至网协通行证完成登录。</NText></NSpace
            >
          </section></NGi
        ></NGrid
      ></NCard
    >
  </main>
</template>
<style scoped>
.login-stage {
  min-height: 100dvh;
  display: grid;
  place-items: center;
  padding: 24px;
  background: #f3f5f8;
}
.login-card {
  max-width: 800px;
  overflow: hidden;
}
.login-product {
  padding: 32px;
  background: #e8f0ff;
  height: 100%;
}
.login-product h1,
.login-action h2 {
  margin: 0;
}
.login-mark {
  color: white;
}
.product-features {
  background: transparent;
}
.login-action {
  height: 100%;
  padding: 32px;
  display: flex;
  flex-direction: column;
  justify-content: center;
}
@media (max-width: 639px) {
  .login-stage {
    padding: 16px;
  }
  .login-product,
  .login-action {
    padding: 24px;
  }
}
</style>
