<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { NAlert, NButton } from 'naive-ui'
import { ArrowRight, KeyRound, Moon, ShieldCheck, Sun } from '@lucide/vue'
import { useAuth } from '@/composables/useAuth'
import { useTheme } from '@/composables/useTheme'
import { safeReturnTo } from '@/lib/utils'

const route = useRoute()
const router = useRouter()
const { state, error: connectionError, refresh, login } = useAuth()
const { isDark, toggleTheme } = useTheme()

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

const departments = [
  {
    title: '技术部',
    y: 59,
    active: true,
    leaves: [
      { title: '公会系统改版', y: 39, status: 'active', highlight: true },
      { title: '秋季招新题库', y: 79, status: 'done', highlight: false },
    ],
  },
  {
    title: '电脑诊所',
    y: 153,
    active: false,
    leaves: [
      { title: '校区义诊排班', y: 133, status: 'done', highlight: false },
      { title: '故障速查手册', y: 173, status: 'active', highlight: false },
    ],
  },
  {
    title: '数字媒体部',
    y: 247,
    active: false,
    leaves: [
      { title: '招新主视觉海报', y: 227, status: 'done', highlight: false },
      { title: '活动摄影归档', y: 267, status: 'active', highlight: false },
    ],
  },
  {
    title: '技术服务部',
    y: 341,
    active: false,
    leaves: [
      { title: '基础设施巡检', y: 321, status: 'active', highlight: false },
      { title: '数据备份演练', y: 361, status: 'done', highlight: false },
    ],
  },
] as const

onMounted(async () => {
  if (await refresh()) await router.replace({ name: 'home' })
})
</script>

<template>
  <main class="login-page" :class="{ 'is-dark': isDark }">
    <aside class="login-showcase" aria-hidden="true">
      <div class="showcase-inner">
        <div class="showcase-header">
          <span class="org-dot" />
          <span>BITNP · 北京理工大学网络开拓者协会</span>
        </div>

        <div class="showcase-card">
          <div class="showcase-card-bar">
            <span class="showcase-card-title">节点树拓扑 · 部门与事项协作</span>
            <div class="showcase-legend">
              <span class="legend-item">
                <i class="legend-dot is-active" />进行中
              </span>
              <span class="legend-item">
                <i class="legend-dot is-done" />已完成
              </span>
            </div>
          </div>

          <svg
            viewBox="0 0 500 400"
            class="topology-svg"
            fill="none"
            xmlns="http://www.w3.org/2000/svg"
          >
            <g class="tree-links">
              <path
                v-for="dept in departments"
                :key="`root-${dept.title}`"
                :d="`M 108 200 C 138 200, 134 ${dept.y}, 164 ${dept.y}`"
                :class="['tree-link', { 'is-highlight': dept.active }]"
              />
              <template v-for="dept in departments" :key="`leaves-${dept.title}`">
                <path
                  v-for="leaf in dept.leaves"
                  :key="leaf.title"
                  :d="`M 280 ${dept.y} C 307 ${dept.y}, 307 ${leaf.y}, 334 ${leaf.y}`"
                  :class="['tree-link', { 'is-highlight': leaf.highlight }]"
                />
              </template>
            </g>

            <g class="root-node">
              <rect
                x="16"
                y="179"
                width="92"
                height="42"
                rx="10"
                class="node-box is-root"
              />
              <circle cx="38" cy="200" r="5" class="node-dot is-active" />
              <text x="52" y="205" class="node-text is-root">公会</text>
            </g>

            <g
              v-for="dept in departments"
              :key="dept.title"
              class="dept-node"
            >
              <rect
                x="164"
                :y="dept.y - 17"
                width="116"
                height="34"
                rx="8"
                :class="['node-box', { 'is-emphasis': dept.active }]"
              />
              <circle
                cx="182"
                :cy="dept.y"
                r="4"
                :class="['node-dot', dept.active ? 'is-active' : 'is-muted']"
              />
              <text x="194" :y="dept.y + 4.5" class="node-text is-dept">
                {{ dept.title }}
              </text>

              <g v-for="leaf in dept.leaves" :key="leaf.title">
                <rect
                  x="334"
                  :y="leaf.y - 15"
                  width="150"
                  height="30"
                  rx="7"
                  :class="['node-box is-leaf', { 'is-emphasis': leaf.highlight }]"
                />
                <circle
                  cx="350"
                  :cy="leaf.y"
                  r="3.8"
                  :class="[
                    'node-dot',
                    leaf.status === 'done' ? 'is-done' : 'is-active',
                  ]"
                />
                <text
                  x="362"
                  :y="leaf.y + 4"
                  :class="[
                    'node-text is-leaf',
                    { 'is-completed': leaf.status === 'done' },
                  ]"
                >
                  {{ leaf.title }}
                </text>
              </g>
            </g>
          </svg>
        </div>

        <div class="showcase-footer">
          <span>技术部</span>
          <span class="footer-sep">·</span>
          <span>电脑诊所</span>
          <span class="footer-sep">·</span>
          <span>数字媒体部</span>
          <span class="footer-sep">·</span>
          <span>技术服务部</span>
        </div>
      </div>
    </aside>

    <section class="login-panel">
      <header class="panel-topbar">
        <span class="mobile-org">BITNP · 网络开拓者协会</span>
        <NButton
          quaternary
          size="small"
          class="theme-switch"
          :aria-label="isDark ? '切换到浅色主题' : '切换到深色主题'"
          :aria-pressed="isDark"
          :title="isDark ? '切换到浅色主题' : '切换到深色主题'"
          @click="toggleTheme"
        >
          <template #icon>
            <Sun v-if="isDark" :size="16" />
            <Moon v-else :size="16" />
          </template>
          {{ isDark ? '浅色' : '深色' }}
        </NButton>
      </header>

      <div class="panel-body">
        <div class="brand-lockup">
          <svg
            class="brand-emblem"
            viewBox="0 0 44 44"
            fill="none"
            xmlns="http://www.w3.org/2000/svg"
            aria-hidden="true"
          >
            <rect width="44" height="44" rx="11" fill="url(#guild-emblem-bg)" />
            <rect
              x="0.5"
              y="0.5"
              width="43"
              height="43"
              rx="10.5"
              stroke="white"
              stroke-opacity="0.18"
            />
            <path
              d="M15 22H21.5M21.5 22C24 22 24.5 15.5 28 15.5M21.5 22C24 22 24.5 28.5 28 28.5"
              stroke="white"
              stroke-width="2.4"
              stroke-linecap="round"
            />
            <circle cx="13.5" cy="22" r="3.2" fill="white" />
            <circle cx="30" cy="15.5" r="3" fill="white" />
            <circle cx="30" cy="28.5" r="3" fill="white" fill-opacity="0.82" />
            <defs>
              <linearGradient
                id="guild-emblem-bg"
                x1="6"
                y1="4"
                x2="38"
                y2="42"
                gradientUnits="userSpaceOnUse"
              >
                <stop stop-color="#2B6DE0" />
                <stop offset="1" stop-color="#194FA3" />
              </linearGradient>
            </defs>
          </svg>
          <div class="brand-titles">
            <h1>公会</h1>
            <span class="brand-subtitle">Guild of Pioneers</span>
          </div>
        </div>

        <div class="panel-copy">
          <h2>登录工作台</h2>
          <p>使用网协通行证登录，自动同步部门与协作权限。</p>
        </div>

        <NAlert v-if="error" type="error" class="login-alert">
          {{ error }}
        </NAlert>

        <button
          type="button"
          class="sso-button"
          :disabled="state === 'active'"
          @click="login(returnTo)"
        >
          <span class="sso-button-main">
            <KeyRound :size="17" aria-hidden="true" />
            <span>使用网协通行证登录</span>
          </span>
          <ArrowRight :size="17" class="sso-arrow" aria-hidden="true" />
        </button>

        <p v-if="returnTo !== '/'" class="redirect-hint">
          登录后将返回 <code>{{ returnTo }}</code>
        </p>

        <ul class="access-notes" aria-label="访问说明">
          <li>
            <ShieldCheck :size="15" class="note-icon" aria-hidden="true" />
            <span>仅面向北京理工大学网络开拓者协会在册成员开放</span>
          </li>
          <li>
            <ShieldCheck :size="15" class="note-icon" aria-hidden="true" />
            <span>首次登录若提示暂无权限，请联系所属部门管理员分配角色</span>
          </li>
        </ul>
      </div>

      <footer class="panel-footer">
        <span>网协统一身份认证 · OIDC 安全会话</span>
      </footer>
    </section>
  </main>
</template>

<style scoped>
.login-page {
  --showcase-bg: #edf2f8;
  --showcase-grid: rgba(23, 32, 43, 0.075);
  --showcase-glow: rgba(31, 95, 191, 0.09);
  --frame-bg: rgba(255, 255, 255, 0.88);
  --frame-border: #d8e1ec;
  --frame-shadow: 0 12px 32px rgba(23, 32, 43, 0.07);
  --node-bg: #ffffff;
  --node-leaf-bg: #f8fafd;
  --node-border: #d5dfeb;
  --node-emphasis-bg: #eff5ff;
  --node-emphasis-border: #8ab4f8;
  --link-stroke: #c5d2e2;
  --link-highlight: #2b6de0;

  min-height: 100dvh;
  display: grid;
  grid-template-columns: minmax(440px, 1.12fr) minmax(380px, 0.88fr);
  background: var(--guild-surface);
  color: var(--guild-text);
}

.login-page.is-dark {
  --showcase-bg: #0d131a;
  --showcase-grid: rgba(148, 173, 201, 0.085);
  --showcase-glow: rgba(43, 109, 224, 0.16);
  --frame-bg: rgba(21, 30, 40, 0.86);
  --frame-border: #253443;
  --frame-shadow: 0 18px 42px rgba(0, 0, 0, 0.36);
  --node-bg: #182330;
  --node-leaf-bg: #151f2b;
  --node-border: #2b3c4d;
  --node-emphasis-bg: #1a2c44;
  --node-emphasis-border: #3d75d4;
  --link-stroke: #2c3e50;
  --link-highlight: #5b97ff;
}

.login-showcase {
  position: relative;
  border-right: 1px solid var(--guild-border);
  background-color: var(--showcase-bg);
  background-image:
    radial-gradient(620px circle at 48% 50%, var(--showcase-glow), transparent 72%),
    radial-gradient(var(--showcase-grid) 1px, transparent 1px);
  background-size:
    100% 100%,
    22px 22px;
  overflow: hidden;
}

.showcase-inner {
  max-width: 580px;
  width: 100%;
  min-height: 100dvh;
  margin: 0 auto;
  padding: 32px 40px;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  gap: 28px;
}

.showcase-header {
  display: inline-flex;
  align-items: center;
  gap: 9px;
  font-size: 13px;
  font-weight: 500;
  color: var(--guild-muted);
}

.org-dot {
  width: 7px;
  height: 7px;
  border-radius: 999px;
  background: var(--link-highlight);
  box-shadow: 0 0 0 3px rgba(43, 109, 224, 0.18);
}

.showcase-card {
  border-radius: 14px;
  border: 1px solid var(--frame-border);
  background: var(--frame-bg);
  box-shadow: var(--frame-shadow);
  padding: 18px 20px 14px;
}

.showcase-card-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding-bottom: 12px;
  margin-bottom: 6px;
  border-bottom: 1px solid var(--frame-border);
}

.showcase-card-title {
  font-size: 12.5px;
  font-weight: 600;
  color: var(--guild-text);
}

.showcase-legend {
  display: flex;
  align-items: center;
  gap: 12px;
  font-size: 12px;
  color: var(--guild-muted);
}

.legend-item {
  display: inline-flex;
  align-items: center;
  gap: 5px;
}

.legend-dot {
  width: 7px;
  height: 7px;
  border-radius: 999px;
  display: inline-block;
}

.legend-dot.is-active {
  background: var(--link-highlight);
}

.legend-dot.is-done {
  background: var(--guild-success);
}

.topology-svg {
  display: block;
  width: 100%;
  height: auto;
}

.tree-link {
  stroke: var(--link-stroke);
  stroke-width: 1.4;
}

.tree-link.is-highlight {
  stroke: var(--link-highlight);
  stroke-width: 1.75;
}

.node-box {
  fill: var(--node-bg);
  stroke: var(--node-border);
  stroke-width: 1.15;
}

.node-box.is-leaf {
  fill: var(--node-leaf-bg);
}

.node-box.is-root,
.node-box.is-emphasis {
  fill: var(--node-emphasis-bg);
  stroke: var(--node-emphasis-border);
  stroke-width: 1.35;
}

.node-dot.is-active {
  fill: var(--link-highlight);
}

.node-dot.is-done {
  fill: var(--guild-success);
}

.node-dot.is-muted {
  fill: var(--guild-muted);
  opacity: 0.7;
}

.node-text {
  fill: var(--guild-text);
  font-family: inherit;
}

.node-text.is-root {
  font-size: 14px;
  font-weight: 650;
}

.node-text.is-dept {
  font-size: 12.5px;
  font-weight: 600;
}

.node-text.is-leaf {
  font-size: 11.8px;
  font-weight: 500;
}

.node-text.is-completed {
  fill: var(--guild-muted);
}

.showcase-footer {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  font-size: 12.5px;
  color: var(--guild-muted);
}

.footer-sep {
  opacity: 0.5;
}

.login-panel {
  min-height: 100dvh;
  padding: 24px 36px 28px;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  background: var(--guild-surface);
}

.panel-topbar {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 12px;
}

.mobile-org {
  display: none;
  font-size: 12.5px;
  font-weight: 500;
  color: var(--guild-muted);
}

.theme-switch {
  color: var(--guild-muted);
}

.panel-body {
  max-width: 368px;
  width: 100%;
  margin: auto;
  padding: 24px 0;
}

.brand-lockup {
  display: flex;
  align-items: center;
  gap: 14px;
}

.brand-emblem {
  width: 44px;
  height: 44px;
  flex: none;
  box-shadow: 0 6px 16px rgba(31, 95, 191, 0.24);
  border-radius: 11px;
}

.brand-titles {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.brand-titles h1 {
  margin: 0;
  font-size: 24px;
  line-height: 1.2;
  font-weight: 700;
  letter-spacing: -0.01em;
}

.brand-subtitle {
  font-size: 12.5px;
  color: var(--guild-muted);
}

.panel-copy {
  margin: 28px 0 22px;
}

.panel-copy h2 {
  margin: 0 0 6px;
  font-size: 18px;
  line-height: 1.35;
  font-weight: 650;
}

.panel-copy p {
  margin: 0;
  font-size: 13.5px;
  line-height: 1.6;
  color: var(--guild-muted);
}

.login-alert {
  margin-bottom: 16px;
}

.sso-button {
  width: 100%;
  height: 46px;
  padding: 0 18px;
  border-radius: 10px;
  border: 1px solid rgba(255, 255, 255, 0.14);
  background: linear-gradient(180deg, #2b6de0 0%, #1f5fbf 100%);
  color: #ffffff;
  font: inherit;
  font-size: 14.5px;
  font-weight: 600;
  display: flex;
  align-items: center;
  justify-content: space-between;
  cursor: pointer;
  box-shadow:
    0 2px 8px rgba(31, 95, 191, 0.26),
    inset 0 1px 0 rgba(255, 255, 255, 0.16);
  transition:
    background 0.16s ease,
    box-shadow 0.16s ease,
    transform 0.12s ease;
}

.sso-button:hover:not(:disabled) {
  background: linear-gradient(180deg, #3578eb 0%, #2365c9 100%);
  box-shadow:
    0 4px 14px rgba(31, 95, 191, 0.34),
    inset 0 1px 0 rgba(255, 255, 255, 0.2);
}

.sso-button:active:not(:disabled) {
  background: #1b53a8;
  transform: translateY(0.5px);
}

.sso-button:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.sso-button-main {
  display: inline-flex;
  align-items: center;
  gap: 10px;
}

.sso-arrow {
  opacity: 0.88;
  transition: transform 0.16s ease;
}

.sso-button:hover:not(:disabled) .sso-arrow {
  transform: translateX(2px);
}

.redirect-hint {
  margin: 12px 0 0;
  font-size: 12px;
  color: var(--guild-muted);
}

.redirect-hint code {
  padding: 1px 6px;
  border-radius: 4px;
  background: var(--guild-surface-subtle);
  border: 1px solid var(--guild-border);
  font-size: 11.5px;
}

.access-notes {
  list-style: none;
  margin: 26px 0 0;
  padding: 18px 0 0;
  border-top: 1px solid var(--guild-border);
  display: flex;
  flex-direction: column;
  gap: 10px;
  font-size: 12.5px;
  line-height: 1.5;
  color: var(--guild-muted);
}

.access-notes li {
  display: flex;
  align-items: flex-start;
  gap: 8px;
}

.note-icon {
  flex: none;
  margin-top: 2px;
  color: var(--guild-primary);
  opacity: 0.85;
}

.panel-footer {
  font-size: 12px;
  color: var(--guild-muted);
  text-align: center;
  opacity: 0.85;
}

@media (max-width: 879px) {
  .login-page {
    grid-template-columns: 1fr;
    background: var(--guild-app-bg);
  }

  .login-showcase {
    display: none;
  }

  .login-panel {
    padding: 18px 20px 24px;
    background: var(--guild-app-bg);
  }

  .panel-topbar {
    justify-content: space-between;
  }

  .mobile-org {
    display: inline;
  }

  .panel-body {
    padding: 28px 24px;
    border-radius: 14px;
    border: 1px solid var(--guild-border);
    background: var(--guild-surface);
    box-shadow: 0 8px 24px var(--guild-shadow);
  }
}
</style>
