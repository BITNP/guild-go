<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import {
  ArrowUpDown,
  Check,
  Clock,
  ListTree,
  Loader2,
  MessageSquare,
  Plus,
} from '@lucide/vue'
import AppShell from '@/components/AppShell.vue'
import NodeFormDialog from '@/components/NodeFormDialog.vue'
import OrderDialog from '@/components/OrderDialog.vue'
import UserAvatar from '@/components/UserAvatar.vue'
import { useAuth } from '@/composables/useAuth'
import { useNodeStore } from '@/composables/useNodeStore'
import { fetchOverview, type NodeDetail, type Overview } from '@/lib/api'
import {
  errorMessage,
  formatDateTime,
  isOverdue,
  reactionEmoji,
  timeAgo,
} from '@/lib/utils'

const router = useRouter()
const { isManager } = useAuth()
const { nodes, reload: reloadNodes } = useNodeStore()

const overview = ref<Overview | null>(null)
const loading = ref(true)
const error = ref<string | null>(null)
const createOpen = ref(false)
const orderOpen = ref(false)

const stats = computed(() => {
  const list = nodes.value
  return {
    roots: list.filter((node) => node.parentId == null).length,
    total: list.length,
    mine: list.filter((node) => node.myRelation === 'leader').length,
    open: list.filter((node) => !node.completedAt).length,
    overdue: list.filter((node) => isOverdue(node.dueAt, node.completedAt)).length,
  }
})

async function load() {
  loading.value = true
  error.value = null
  try {
    overview.value = await fetchOverview()
  } catch (err) {
    error.value = errorMessage(err, '无法加载总览')
  } finally {
    loading.value = false
  }
}

async function refresh() {
  await Promise.all([load(), reloadNodes()])
}

async function onCreated(node: NodeDetail) {
  createOpen.value = false
  await refresh()
  await router.push({ name: 'node', params: { id: node.id } })
}

async function onOrdered() {
  orderOpen.value = false
  await refresh()
}

onMounted(load)
</script>

<template>
  <AppShell>
    <div class="flex flex-col gap-4 p-4 lg:p-5">
      <header class="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h1 class="text-lg font-semibold tracking-tight text-ink">总览</h1>
          <p class="mt-0.5 text-[12px] text-muted">以树状结构组织公会工作，点开任意节点查看子节点最新讨论。</p>
        </div>
        <div v-if="isManager" class="flex items-center gap-1.5">
          <button type="button" class="btn" @click="orderOpen = true">
            <ArrowUpDown class="h-3.5 w-3.5" /> 根节点排序
          </button>
          <button type="button" class="btn btn-primary" @click="createOpen = true">
            <Plus class="h-3.5 w-3.5" /> 新建根节点
          </button>
        </div>
      </header>

      <div class="grid grid-cols-2 gap-2 sm:grid-cols-3 lg:grid-cols-5">
        <div class="panel px-3 py-2">
          <p class="text-[10px] text-muted">根节点</p>
          <p class="mt-0.5 text-lg font-semibold text-ink">{{ stats.roots }}</p>
        </div>
        <div class="panel px-3 py-2">
          <p class="text-[10px] text-muted">全部节点</p>
          <p class="mt-0.5 text-lg font-semibold text-ink">{{ stats.total }}</p>
        </div>
        <div class="panel px-3 py-2">
          <p class="text-[10px] text-muted">我负责</p>
          <p class="mt-0.5 text-lg font-semibold text-accent">{{ stats.mine }}</p>
        </div>
        <div class="panel px-3 py-2">
          <p class="text-[10px] text-muted">未完成</p>
          <p class="mt-0.5 text-lg font-semibold text-ink">{{ stats.open }}</p>
        </div>
        <div class="panel px-3 py-2">
          <p class="text-[10px] text-muted">已逾期</p>
          <p class="mt-0.5 text-lg font-semibold" :class="stats.overdue ? 'text-warn' : 'text-ink'">{{ stats.overdue }}</p>
        </div>
      </div>

      <p v-if="loading" class="flex items-center gap-2 py-6 text-[13px] text-muted">
        <Loader2 class="h-4 w-4 animate-spin" /> 正在加载总览…
      </p>
      <p v-else-if="error" class="rounded-md border border-danger/20 bg-danger-soft px-3 py-2 text-[12px] text-danger">{{ error }}</p>

      <section v-else class="panel overflow-hidden">
        <header class="flex items-center justify-between border-b border-line px-3 py-2">
          <h2 class="text-[12px] font-semibold text-ink">根节点</h2>
          <span class="text-[10px] text-faint">按手动顺序排列 · 点击进入</span>
        </header>

        <p v-if="!overview || overview.roots.length === 0" class="empty-state">
          <ListTree class="h-5 w-5 text-faint" />
          <span>还没有根节点。{{ isManager ? '点击右上角新建第一棵树。' : '请联系 manager 创建。' }}</span>
        </p>

        <RouterLink
          v-for="root in overview?.roots ?? []"
          :key="root.id"
          :to="{ name: 'node', params: { id: root.id } }"
          class="block border-b border-line px-3 py-2.5 last:border-b-0 row-hover"
        >
          <div class="flex items-start gap-2.5">
            <span
              v-if="root.completedAt"
              class="mt-1 flex h-4 w-4 shrink-0 items-center justify-center rounded-full bg-ok-soft text-ok"
              title="已完成"
            >
              <Check class="h-3 w-3" />
            </span>
            <span
              v-else
              class="status-dot mt-2 shrink-0"
              :class="isOverdue(root.dueAt, root.completedAt) ? 'bg-warn' : 'bg-line-strong'"
            />

            <div class="min-w-0 flex-1">
              <div class="flex flex-wrap items-center gap-x-2 gap-y-1">
                <span class="truncate text-[13.5px] font-semibold text-ink" :class="root.completedAt ? 'text-muted line-through' : ''">
                  {{ root.title }}
                </span>
                <span v-for="tag in root.tags.slice(0, 4)" :key="tag" class="tag"># {{ tag }}</span>
                <span
                  v-if="isOverdue(root.dueAt, root.completedAt)"
                  class="inline-flex items-center gap-0.5 rounded bg-warn-soft px-1 py-0.5 text-[10px] text-warn"
                >
                  <Clock class="h-2.5 w-2.5" /> 逾期
                </span>
                <span v-else-if="root.dueAt && !root.completedAt" class="inline-flex items-center gap-0.5 text-[10px] text-muted">
                  <Clock class="h-2.5 w-2.5" /> {{ formatDateTime(root.dueAt) }}
                </span>
              </div>

              <div class="mt-1 flex flex-wrap items-center gap-x-3 gap-y-1 text-[10px] text-faint">
                <span v-if="root.leaders.length" class="flex items-center gap-1">
                  <span class="flex -space-x-1">
                    <UserAvatar
                      v-for="leader in root.leaders.slice(0, 4)"
                      :key="leader.subject"
                      :name="leader.userName"
                      :avatar="leader.avatar"
                      size-class="h-4 w-4 text-[8px] border-white"
                    />
                  </span>
                  {{ root.leaders.map((leader) => leader.userName).join('、') }}
                </span>
                <span v-else>暂无负责人</span>
                <span>{{ root.childCount }} 个子节点</span>
                <span class="inline-flex items-center gap-0.5"><MessageSquare class="h-2.5 w-2.5" /> {{ root.commentCount }}</span>
                <span v-if="root.latestCommentAt">最近讨论 {{ timeAgo(root.latestCommentAt) }}</span>
              </div>

              <div v-if="root.latestComment" class="mt-1.5 flex items-start gap-1.5 rounded border border-line/70 bg-surface-2/70 px-2 py-1.5">
                <UserAvatar
                  :name="root.latestComment.author.userName"
                  :avatar="root.latestComment.author.avatar"
                  size-class="h-4 w-4 text-[8px]"
                />
                <div class="min-w-0 flex-1">
                  <div class="flex items-center gap-1.5 text-[10px]">
                    <span class="font-medium text-ink-soft">{{ root.latestComment.author.userName }}</span>
                    <span class="text-faint">{{ timeAgo(root.latestComment.createdAt) }}</span>
                  </div>
                  <p class="line-clamp-1 text-[11px] text-muted">{{ root.latestComment.bodyPreview || '（图片）' }}</p>
                </div>
                <div v-if="root.latestComment.reactionSummary.length" class="flex shrink-0 gap-1 text-[10px] text-muted">
                  <span v-for="reaction in root.latestComment.reactionSummary.slice(0, 3)" :key="reaction.emoji">
                    {{ reactionEmoji[reaction.emoji]?.symbol }}{{ reaction.count }}
                  </span>
                </div>
              </div>
              <p v-else class="mt-1.5 text-[11px] text-faint">暂无评论</p>
            </div>
          </div>
        </RouterLink>
      </section>
    </div>

    <template #rail>
      <div class="flex flex-col gap-2 p-2.5">
        <div class="flex items-center justify-between">
          <h2 class="rail-title">最近讨论</h2>
          <span class="text-[10px] text-faint">全站</span>
        </div>
        <p v-if="loading" class="py-3 text-[11px] text-muted">加载中…</p>
        <p v-else-if="!overview || overview.recentComments.length === 0" class="py-3 text-[11px] text-muted">还没有评论。</p>
        <RouterLink
          v-for="comment in overview?.recentComments ?? []"
          :key="comment.id"
          :to="{ name: 'node', params: { id: comment.nodeId } }"
          class="rounded-md border border-line bg-surface px-2.5 py-2 row-hover"
        >
          <div class="flex items-center gap-1.5">
            <UserAvatar :name="comment.author.userName" :avatar="comment.author.avatar" size-class="h-4 w-4 text-[8px]" />
            <span class="truncate text-[11px] font-medium text-ink-soft">{{ comment.author.userName }}</span>
            <span class="ml-auto shrink-0 text-[10px] text-faint">{{ timeAgo(comment.createdAt) }}</span>
          </div>
          <p class="mt-1 line-clamp-2 text-[11px] leading-5 text-muted">{{ comment.bodyPreview || '（图片评论）' }}</p>
          <div class="mt-1 flex items-center gap-1 text-[10px] text-faint">
            <span class="truncate">{{ comment.rootTitle }} / {{ comment.nodeTitle }}</span>
            <span v-if="comment.reactionSummary.length" class="ml-auto flex shrink-0 gap-1">
              <span v-for="reaction in comment.reactionSummary.slice(0, 2)" :key="reaction.emoji">
                {{ reactionEmoji[reaction.emoji]?.symbol }}{{ reaction.count }}
              </span>
            </span>
          </div>
        </RouterLink>
      </div>
    </template>
  </AppShell>

  <NodeFormDialog
    v-model:open="createOpen"
    mode="create"
    :parent-id="null"
    @saved="onCreated"
  />
  <OrderDialog
    :open="orderOpen"
    :parent-id="null"
    parent-title=""
    :children="overview?.roots ?? []"
    @close="orderOpen = false"
    @saved="onOrdered"
  />
</template>