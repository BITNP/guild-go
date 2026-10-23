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
const { nodes, loading: nodesLoading, error: nodesError, reload: reloadNodes } = useNodeStore()

const overview = ref<Overview | null>(null)
const loading = ref(true)
const error = ref<string | null>(null)
const createOpen = ref(false)
const orderOpen = ref(false)

const stats = computed(() => {
  const list = nodes.value
  return {
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
  <AppShell rail-title="最近评论">
    <div class="mx-auto flex max-w-5xl flex-col gap-4 p-4 lg:p-6">
      <header class="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h1 class="text-lg font-semibold tracking-tight text-ink">总览</h1>
        </div>
        <div v-if="isManager" class="flex items-center gap-1.5">
          <button v-if="(overview?.roots.length ?? 0) > 1" type="button" class="btn" @click="orderOpen = true">
            <ArrowUpDown class="h-3.5 w-3.5" /> 根节点排序
          </button>
          <button type="button" class="btn btn-primary" @click="createOpen = true">
            <Plus class="h-3.5 w-3.5" /> 新建根节点
          </button>
        </div>
      </header>

      <dl v-if="overview && !loading && !error && !nodesLoading && !nodesError" class="panel grid grid-cols-2 overflow-hidden text-xs text-muted sm:grid-cols-4">
        <div class="border-b border-r border-line px-4 py-3 sm:border-b-0"><dt>全部节点</dt><dd class="mt-1 text-xl font-semibold tabular-nums text-ink">{{ stats.total }}</dd></div>
        <div class="border-b border-line px-4 py-3 sm:border-b-0 sm:border-r"><dt>我负责</dt><dd class="mt-1 text-xl font-semibold tabular-nums text-ink">{{ stats.mine }}</dd></div>
        <div class="border-r border-line px-4 py-3"><dt>未完成</dt><dd class="mt-1 text-xl font-semibold tabular-nums text-ink">{{ stats.open }}</dd></div>
        <div class="px-4 py-3"><dt>已逾期</dt><dd class="mt-1 text-xl font-semibold tabular-nums" :class="stats.overdue ? 'text-warn' : 'text-ink'">{{ stats.overdue }}</dd></div>
      </dl>

      <p v-if="loading" class="flex items-center gap-2 py-6 text-[13px] text-muted">
        <Loader2 class="h-4 w-4 animate-spin" /> 正在加载总览…
      </p>
      <p v-else-if="error" class="rounded-md border border-danger/20 bg-danger-soft px-3 py-2 text-[12px] text-danger" role="alert">{{ error }} <button type="button" class="ml-2 underline" @click="refresh">重试</button></p>

      <section v-else class="panel overflow-hidden">
        <header class="flex items-center justify-between border-b border-line px-3 py-2">
          <h2 class="text-[12px] font-semibold text-ink">根节点</h2>
          <span class="text-xs text-muted">{{ overview?.roots.length ?? 0 }} 个</span>
        </header>

        <p v-if="!overview || overview.roots.length === 0" class="empty-state">
          <ListTree class="h-5 w-5 text-faint" />
          <span>{{ isManager ? '暂无根节点，可新建根节点开始安排工作。' : '暂无根节点，请联系部门管理员创建。' }}</span>
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
              class="status-icon bg-ok-soft text-ok"
              title="已完成"
            >
              <Check class="h-3 w-3" />
            </span>
            <span
              v-else
              class="status-icon"
              :class="isOverdue(root.dueAt, root.completedAt) ? 'text-warn' : 'text-muted'"
              aria-label="未完成"
            ><span class="h-3.5 w-3.5 rounded-full border border-current" /></span>

            <div class="min-w-0 flex-1">
              <div class="flex flex-wrap items-center gap-x-2 gap-y-1">
                <span class="min-w-0 break-words text-sm font-semibold text-ink" :class="root.completedAt ? 'text-muted line-through' : ''">
                  {{ root.title }}
                </span>
                <span v-for="tag in root.tags.slice(0, 4)" :key="tag" class="tag"># {{ tag }}</span>
                <span
                  v-if="isOverdue(root.dueAt, root.completedAt)"
                  class="inline-flex items-center gap-0.5 rounded bg-warn-soft px-1 py-0.5 text-xs text-warn"
                >
                  <Clock class="h-2.5 w-2.5" /> 逾期
                </span>
                <span v-else-if="root.dueAt && !root.completedAt" class="inline-flex items-center gap-0.5 text-xs text-muted">
                  <Clock class="h-2.5 w-2.5" /> {{ formatDateTime(root.dueAt) }}
                </span>
              </div>

              <div class="mt-1 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-faint">
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
              </div>

              <div v-if="root.latestComment" class="mt-2 flex items-start gap-2 border-l-2 border-line pl-3">
                <UserAvatar
                  :name="root.latestComment.author.userName"
                  :avatar="root.latestComment.author.avatar"
                  size-class="h-4 w-4 text-[8px]"
                />
                <div class="min-w-0 flex-1">
                  <div class="flex items-center gap-1.5 text-xs">
                    <span class="min-w-0 truncate font-medium text-ink-soft">{{ root.latestComment.author.userName }}</span>
                    <span class="shrink-0 text-faint">{{ timeAgo(root.latestComment.createdAt) }}</span>
                  </div>
                  <p class="line-clamp-1 text-xs text-muted">{{ root.latestComment.bodyPreview || '图片评论' }}</p>
                </div>
                <div v-if="root.latestComment.reactionSummary.length" class="flex shrink-0 gap-1 text-xs text-muted">
                  <span v-for="reaction in root.latestComment.reactionSummary.slice(0, 3)" :key="reaction.emoji">
                    {{ reactionEmoji[reaction.emoji]?.symbol }}{{ reaction.count }}
                  </span>
                </div>
              </div>
            </div>
          </div>
        </RouterLink>
      </section>
    </div>

    <template #rail>
      <div class="flex flex-col divide-y divide-line px-4">
        <p v-if="loading" class="py-3 text-xs text-muted">加载中…</p>
        <p v-else-if="error" class="py-3 text-xs text-danger">评论加载失败</p>
        <p v-else-if="!overview || overview.recentComments.length === 0" class="py-3 text-xs text-muted">还没有评论。</p>
        <RouterLink
          v-for="comment in overview?.recentComments ?? []"
          :key="comment.id"
          :to="{ name: 'node', params: { id: comment.nodeId } }"
          class="block py-3 row-hover"
        >
          <div class="flex items-center gap-1.5">
            <UserAvatar :name="comment.author.userName" :avatar="comment.author.avatar" size-class="h-4 w-4 text-[8px]" />
            <span class="truncate text-xs font-medium text-ink-soft">{{ comment.author.userName }}</span>
            <span class="ml-auto shrink-0 text-xs text-faint">{{ timeAgo(comment.createdAt) }}</span>
          </div>
          <p class="mt-1 line-clamp-2 text-xs leading-5 text-muted">{{ comment.bodyPreview || '图片评论' }}</p>
          <div class="mt-1 flex items-center gap-1 text-xs text-faint">
            <span class="truncate">{{ comment.rootTitle === comment.nodeTitle ? comment.nodeTitle : `${comment.rootTitle} / ${comment.nodeTitle}` }}</span>
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
