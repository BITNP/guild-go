<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { NAlert } from 'naive-ui'
import { Moon, Sun } from '@lucide/vue'
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
    return '此账号暂无开拓者公会访问权限，请联系部门管理员开通。'
  if (code === 'no_roles') return '此账号尚未分配网协角色，请联系管理员。'
  if (code) return '登录已取消或无法完成，请重试。'
  return connectionError.value ? '无法连接登录服务，请稍后重试。' : ''
})

interface GraphPoint {
  id: string
  x: number
  y: number
  r: number
  kind: 'root' | 'primary' | 'secondary' | 'leaf'
  tone: 'default' | 'focus' | 'done'
}

interface GraphEdge {
  id: string
  d: string
  focus: boolean
}

const CX = 320
const CY = 320
const RINGS = [104, 202, 286] as const

function polar(angleDeg: number, radius: number) {
  const rad = (angleDeg * Math.PI) / 180
  return {
    x: Number((CX + radius * Math.cos(rad)).toFixed(1)),
    y: Number((CY + radius * Math.sin(rad)).toFixed(1)),
  }
}

function curveBetween(
  fromAngle: number,
  fromRadius: number,
  toAngle: number,
  toRadius: number,
) {
  const p1 = polar(fromAngle, fromRadius)
  const p2 = polar(toAngle, toRadius)
  const midR = (fromRadius + toRadius) * 0.52
  const ctrl = polar(fromAngle, midR)
  return `M ${p1.x} ${p1.y} Q ${ctrl.x} ${ctrl.y} ${p2.x} ${p2.y}`
}

interface LeafSpec {
  angle: number
  tone?: 'default' | 'focus' | 'done'
}
interface SecondarySpec {
  angle: number
  tone?: 'default' | 'focus' | 'done'
  leaves?: LeafSpec[]
}
interface PrimarySpec {
  angle: number
  tone?: 'default' | 'focus' | 'done'
  children: SecondarySpec[]
}

const topologySpec: PrimarySpec[] = [
  {
    angle: -46,
    tone: 'focus',
    children: [
      {
        angle: -74,
        leaves: [{ angle: -86 }, { angle: -64, tone: 'done' }],
      },
      {
        angle: -38,
        tone: 'focus',
        leaves: [
          { angle: -46, tone: 'focus' },
          { angle: -26 },
        ],
      },
      {
        angle: -14,
        tone: 'done',
        leaves: [{ angle: -6, tone: 'done' }],
      },
    ],
  },
  {
    angle: 38,
    children: [
      {
        angle: 18,
        leaves: [{ angle: 10 }, { angle: 26, tone: 'done' }],
      },
      {
        angle: 52,
        tone: 'done',
        leaves: [{ angle: 42 }, { angle: 64 }],
      },
    ],
  },
  {
    angle: 136,
    children: [
      {
        angle: 108,
        leaves: [{ angle: 96 }, { angle: 118, tone: 'done' }],
      },
      {
        angle: 144,
        leaves: [{ angle: 134 }, { angle: 156 }],
      },
      {
        angle: 168,
        tone: 'done',
      },
    ],
  },
  {
    angle: -140,
    children: [
      {
        angle: -162,
        leaves: [{ angle: -172 }, { angle: -152 }],
      },
      {
        angle: -122,
        tone: 'done',
        leaves: [{ angle: -134 }, { angle: -112, tone: 'done' }],
      },
    ],
  },
]

const graphPoints: GraphPoint[] = [
  { id: 'origin', x: CX, y: CY, r: 7, kind: 'root', tone: 'focus' },
]
const graphEdges: GraphEdge[] = []

topologySpec.forEach((primary, pi) => {
  const p = polar(primary.angle, RINGS[0])
  const primaryFocus = primary.tone === 'focus'
  graphEdges.push({
    id: `e-p-${pi}`,
    d: `M ${CX} ${CY} L ${p.x} ${p.y}`,
    focus: primaryFocus,
  })
  graphPoints.push({
    id: `p-${pi}`,
    x: p.x,
    y: p.y,
    r: 5.5,
    kind: 'primary',
    tone: primary.tone ?? 'default',
  })

  primary.children.forEach((sec, si) => {
    const s = polar(sec.angle, RINGS[1])
    const secFocus = primaryFocus && sec.tone === 'focus'
    graphEdges.push({
      id: `e-s-${pi}-${si}`,
      d: curveBetween(primary.angle, RINGS[0], sec.angle, RINGS[1]),
      focus: secFocus,
    })
    graphPoints.push({
      id: `s-${pi}-${si}`,
      x: s.x,
      y: s.y,
      r: 4.2,
      kind: 'secondary',
      tone: sec.tone ?? 'default',
    })

    sec.leaves?.forEach((leaf, li) => {
      const l = polar(leaf.angle, RINGS[2])
      const leafFocus = secFocus && leaf.tone === 'focus'
      graphEdges.push({
        id: `e-l-${pi}-${si}-${li}`,
        d: curveBetween(sec.angle, RINGS[1], leaf.angle, RINGS[2]),
        focus: leafFocus,
      })
      graphPoints.push({
        id: `l-${pi}-${si}-${li}`,
        x: l.x,
        y: l.y,
        r: 3,
        kind: 'leaf',
        tone: leaf.tone ?? 'default',
      })
    })
  })
})

onMounted(async () => {
  if (await refresh()) await router.replace({ name: 'home' })
})
</script>

<template>
  <main class="login-page" :class="{ 'is-dark': isDark }">
    <header class="login-header">
      <span class="org-name">北京理工大学网络开拓者协会</span>
      <button
        type="button"
        class="theme-toggle"
        :aria-label="isDark ? '切换到浅色主题' : '切换到深色主题'"
        :aria-pressed="isDark"
        :title="isDark ? '切换到浅色主题' : '切换到深色主题'"
        @click="toggleTheme"
      >
        <Sun v-if="isDark" :size="17" aria-hidden="true" />
        <Moon v-else :size="17" aria-hidden="true" />
      </button>
    </header>

    <div class="login-grid">
      <aside class="login-figure" aria-hidden="true">
        <svg
          viewBox="0 0 640 640"
          class="radial-svg"
          fill="none"
          xmlns="http://www.w3.org/2000/svg"
        >
          <g class="figure-axes">
            <line x1="24" y1="320" x2="616" y2="320" />
            <line x1="320" y1="24" x2="320" y2="616" />
          </g>

          <g class="figure-rings">
            <circle :cx="CX" :cy="CY" :r="RINGS[0]" class="ring" />
            <circle :cx="CX" :cy="CY" :r="RINGS[1]" class="ring" />
            <circle :cx="CX" :cy="CY" :r="RINGS[2]" class="ring is-outer" />
          </g>

          <g class="figure-edges">
            <path
              v-for="edge in graphEdges"
              :key="edge.id"
              :d="edge.d"
              :class="['edge', { 'is-focus': edge.focus }]"
            />
          </g>

          <g class="figure-nodes">
            <circle
              v-for="pt in graphPoints"
              :key="pt.id"
              :cx="pt.x"
              :cy="pt.y"
              :r="pt.r"
              :class="['node', `is-${pt.kind}`, `tone-${pt.tone}`]"
            />
            <circle :cx="CX" :cy="CY" r="2.6" class="origin-core" />
          </g>
        </svg>
      </aside>

      <section class="login-editorial">
        <div class="editorial-main">
          <h1 class="login-title">开拓者公会</h1>
          <p class="login-english">Guild of Pioneers</p>
          <p class="login-lead">网协的工作协作空间</p>

          <NAlert v-if="error" type="error" class="login-alert">
            {{ error }}
          </NAlert>

          <div class="login-actions">
            <button
              type="button"
              class="login-button"
              :disabled="state === 'active'"
              @click="login(returnTo)"
            >
              使用网协通行证登录
            </button>
          </div>
        </div>
      </section>
    </div>
  </main>
</template>

<style scoped>
.login-page {
  --figure-bg: var(--guild-surface-subtle);
  --axis-stroke: #dfe5ed;
  --ring-stroke: #d7dfe9;
  --edge-stroke: #c4cfdd;
  --btn-bg: #1f5fbf;
  --btn-hover: #184e9e;
  --btn-active: #144185;

  min-height: 100dvh;
  display: grid;
  grid-template-rows: 52px minmax(0, 1fr);
  background: var(--guild-surface);
  color: var(--guild-text);
}

.login-page.is-dark {
  --figure-bg: #111820;
  --axis-stroke: #1d2834;
  --ring-stroke: #22303e;
  --edge-stroke: #324457;
  --btn-bg: #2563cc;
  --btn-hover: #2d6fdb;
  --btn-active: #1d52ad;
}

.login-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 16px 0 24px;
  border-bottom: 1px solid var(--guild-border);
  background: var(--guild-surface);
}

.org-name {
  font-size: 13px;
  font-weight: 500;
  color: var(--guild-muted);
}

.theme-toggle {
  width: 34px;
  height: 34px;
  padding: 0;
  border: 0;
  border-radius: 6px;
  background: transparent;
  color: var(--guild-muted);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition:
    background-color 0.14s ease,
    color 0.14s ease;
}

.theme-toggle:hover {
  background: var(--guild-surface-subtle);
  color: var(--guild-text);
}

.login-grid {
  display: grid;
  grid-template-columns: minmax(380px, 1.08fr) minmax(380px, 1fr);
  min-height: 0;
}

.login-editorial {
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  padding: clamp(32px, 5vw, 64px) clamp(24px, 6vw, 80px) 28px;
}

.editorial-main {
  margin: auto 0;
  max-width: 380px;
  padding-bottom: 4vh;
}

.login-title {
  margin: 0;
  font-size: clamp(34px, 3.6vw, 48px);
  line-height: 1.14;
  font-weight: 700;
  letter-spacing: -0.025em;
  color: var(--guild-text);
}

.login-english {
  margin: 8px 0 0;
  font-size: 14px;
  line-height: 20px;
  color: var(--guild-muted);
  letter-spacing: 0.01em;
}

.login-lead {
  margin: 24px 0 0;
  font-size: 15px;
  line-height: 1.65;
  color: var(--guild-text);
  opacity: 0.86;
}

.login-alert {
  margin-top: 20px;
}

.login-actions {
  margin-top: 32px;
}

.login-button {
  min-width: 224px;
  height: 44px;
  padding: 0 24px;
  border: 0;
  border-radius: 6px;
  background: var(--btn-bg);
  color: #ffffff;
  font: inherit;
  font-size: 14.5px;
  font-weight: 550;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: background-color 0.14s ease;
}

.login-button:hover:not(:disabled) {
  background: var(--btn-hover);
}

.login-button:active:not(:disabled) {
  background: var(--btn-active);
}

.login-button:disabled {
  opacity: 0.55;
  cursor: not-allowed;
}

.login-figure {
  display: grid;
  place-items: center;
  padding: clamp(24px, 4vw, 56px);
  border-right: 1px solid var(--guild-border);
  background: var(--figure-bg);
  overflow: hidden;
}

.radial-svg {
  width: min(100%, 560px);
  max-height: calc(100dvh - 120px);
  height: auto;
  display: block;
}

.figure-axes line {
  stroke: var(--axis-stroke);
  stroke-width: 1;
}

.ring {
  stroke: var(--ring-stroke);
  stroke-width: 1;
}

.ring.is-outer {
  stroke-dasharray: 3 6;
}

.edge {
  stroke: var(--edge-stroke);
  stroke-width: 1.25;
}

.edge.is-focus {
  stroke: var(--guild-primary);
  stroke-width: 1.6;
}

.node {
  fill: var(--figure-bg);
  stroke: var(--edge-stroke);
  stroke-width: 1.4;
}

.node.is-root {
  stroke: var(--guild-primary);
  stroke-width: 1.6;
}

.node.tone-focus {
  stroke: var(--guild-primary);
}

.node.is-leaf.tone-focus {
  fill: var(--guild-primary);
}

.node.tone-done {
  stroke: var(--guild-success);
}

.node.is-leaf.tone-done {
  fill: var(--guild-success);
}

.origin-core {
  fill: var(--guild-primary);
}

@media (max-width: 859px) {
  .login-grid {
    grid-template-columns: 1fr;
  }

  .login-figure {
    display: none;
  }

  .login-editorial {
    padding: 36px 24px 24px;
  }

  .editorial-main {
    max-width: none;
  }

  .login-button {
    width: 100%;
  }
}
</style>
