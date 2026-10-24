import { createRouter, createWebHistory } from 'vue-router'
import { useAuth } from '@/composables/useAuth'
import { safeReturnTo } from '@/lib/utils'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/login',
      name: 'login',
      component: () => import('@/views/LoginView.vue'),
      meta: { public: true, title: '登录' },
    },
    {
      path: '/',
      name: 'home',
      component: () => import('@/views/OverviewView.vue'),
      meta: { title: '总览' },
    },
    {
      path: '/nodes/:id',
      name: 'node',
      component: () => import('@/views/NodeView.vue'),
      meta: { title: '节点' },
    },
    {
      path: '/graph',
      name: 'graph',
      component: () => import('@/views/GraphView.vue'),
      meta: { title: '关系图谱' },
    },
    {
      path: '/users/:subject',
      name: 'user',
      component: () => import('@/views/UserView.vue'),
      meta: { title: '用户' },
    },
    {
      path: '/account',
      name: 'account',
      component: () => import('@/views/AccountView.vue'),
      meta: { title: '我的账号' },
    },
    {
      path: '/:pathMatch(.*)*',
      redirect: '/',
    },
  ],
  scrollBehavior: () => ({ top: 0 }),
})

router.beforeEach(async (to) => {
  const auth = useAuth()
  try {
    const active = await auth.refresh()
    if (to.meta.public) {
      if (active && to.name === 'login') return { name: 'home' }
      return true
    }
    if (!active) {
      return {
        name: 'login',
        query: { redirect: safeReturnTo(to.fullPath) },
      }
    }
    return true
  } catch {
    if (to.meta.public) return true
    return { name: 'login', query: { error: 'unavailable', redirect: safeReturnTo(to.fullPath) } }
  }
})

router.afterEach((to) => {
  const title = typeof to.meta.title === 'string' ? to.meta.title : ''
  document.title = title ? `${title} · 公会` : '公会'
})

export default router
