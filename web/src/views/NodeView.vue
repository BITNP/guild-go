<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  ArrowUpDown,
  CalendarDays,
  Check,
  ChevronRight,
  Clock,
  ListTree,
  Loader2,
  Plus,
} from '@lucide/vue'
import AppShell from '@/components/AppShell.vue'
import ChildRow from '@/components/ChildRow.vue'
import CommentSection from '@/components/CommentSection.vue'
import DeleteNodeDialog from '@/components/DeleteNodeDialog.vue'
import MembersDialog from '@/components/MembersDialog.vue'
import NodeFormDialog from '@/components/NodeFormDialog.vue'
import OrderDialog from '@/components/OrderDialog.vue'
import UserAvatar from '@/components/UserAvatar.vue'
import { useNodeStore } from '@/composables/useNodeStore'
import {
  fetchNode,
  updateNode,
  type NodeDetail,
} from '@/lib/api'
import {
  errorMessage,
  formatDateTime,
  isOverdue,
  timeAgo,
} from '@/lib/utils'

const route = useRoute()
const router = useRouter()
const { reload: reloadNodes } = useNodeStore()

const detail = ref<NodeDetail | null>(null)
const loading = ref(true)
const error = ref<string | null>(null)
const actionError = ref<string | null>(null)
const toggling = ref(false)
const editOpen = ref(false)
const createChildOpen = ref(false)
const membersOpen = ref(false)
const orderOpen = ref(false)
const deleteOpen = ref(false)

const nodeId = computed(() => Number(route.params.id))
const canEdit = computed(() => detail.value?.permissions.canEdit ?? false)
const canDelete = computed(() => detail.value?.permissions.canDelete ?? false)
const canComment = computed(() => detail.value?.permissions.canComment ?? false)
const canManageMembers = computed(() => detail.value?.permissions.canManageMembers ?? false)
const canAddChild = computed(() => detail.value?.permissions.canAddChild ?? false)
const canReorderChildren = computed(() => detail.value?.permissions.canReorderChildren ?? false)
const overdue = computed(() => detail.value ? isOverdue(detail.value.dueAt, detail.value.completedAt) : false)

async function load(silent = false) {
  if (!Number.isFinite(nodeId.value) || nodeId.value <= 0) {
    error.value = '节点不存在'
    loading.value = false
    return
  }
  if (!silent) loading.value = true
  error.value = null
  actionError.value = null
  try {
    detail.value = await fetchNode(nodeId.value)
  } catch (err) {
    detail.value = null
    error.value = errorMessage(err, '无法加载节点')
  } finally {
    if (!silent) loading.value = false
  }
}

async function toggleComplete() {
  if (!detail.value || toggling.value) return
  toggling.value = true
  actionError.value = null
  try {
    const node = detail.value
    const updated = await updateNode(node.id, {
      title: node.title,
      description: node.description,
      dueAt: node.dueAt,
      tags: node.tags,
      completed: node.completedAt == null,
    })
    detail.value = updated
    await reloadNodes()
  } catch (err) {
    actionError.value = errorMessage(err, '更新完成状态失败')
  } finally {
    toggling.value = false
  }
}

function onEditSaved(node: NodeDetail) {
  detail.value = node
  editOpen.value = false
  void reloadNodes()
}

async function onChildCreated(node: NodeDetail) {
  createChildOpen.value = false
  await reloadNodes()
  await router.push({ name: 'node', params: { id: node.id } })
}

function onMembersSaved(node: NodeDetail) {
  detail.value = node
  membersOpen.value = false
  void reloadNodes()
}

async function onOrdered() {
  orderOpen.value = false
  await Promise.all([load(), reloadNodes()])
}

async function onDeleted() {
  deleteOpen.value = false
  const parentId = detail.value?.parentId ?? null
  await reloadNodes()
  if (parentId) await router.push({ name: 'node', params: { id: parentId } })
  else await router.push({ name: 'home' })
}

async function onCommentChanged() {
  await Promise.all([load(true), reloadNodes()])
}

onMounted(() => load())
watch(nodeId, () => load())
</script>

<template>
  <AppShell>
    <div class="mx-auto flex max-w-5xl flex-col gap-4 p-4 lg:p-6">
      <p v-if="loading" class="flex items-center gap-2 py-10 text-[13px] text-muted">
        <Loader2 class="h-4 w-4 animate-spin" /> 正在加载节点…
      </p>
      <div v-else-if="error" class="rounded-md border border-danger/20 bg-danger-soft px-3 py-3 text-[12px] text-danger">
        {{ error }}
        <button type="button" class="ml-2 underline" @click="load()">重试</button>
        <RouterLink :to="{ name: 'home' }" class="ml-2 underline">返回总览</RouterLink>
      </div>

      <template v-else-if="detail">
        <nav aria-label="节点路径" class="flex flex-wrap items-center gap-1 text-xs text-muted">
          <RouterLink :to="{ name: 'home' }" class="hover:text-accent">总览</RouterLink>
          <template v-for="item in detail.breadcrumb" :key="item.id">
            <ChevronRight class="h-3 w-3 text-faint" />
            <RouterLink :to="{ name: 'node', params: { id: item.id } }" class="max-w-48 truncate hover:text-accent">
              {{ item.title }}
            </RouterLink>
          </template>
        </nav>

        <header class="flex flex-wrap items-start justify-between gap-3">
          <div class="grid min-w-0 flex-1 basis-56 grid-cols-[2rem_minmax(0,1fr)] items-start gap-x-2 gap-y-1">
            <button
              type="button"
              class="flex h-8 w-8 items-center justify-center rounded-full transition-colors disabled:cursor-default"
              :class="detail.completedAt ? 'text-ok' : 'text-muted enabled:hover:text-ok'"
              :disabled="!canEdit || toggling"
              :aria-label="detail.completedAt ? '标记为未完成' : '标记为已完成'"
              :aria-pressed="detail.completedAt != null"
              :aria-busy="toggling"
              :title="detail.completedAt ? '标记为未完成' : '标记为已完成'"
              @click="toggleComplete"
            >
              <Loader2 v-if="toggling" class="h-5 w-5 animate-spin" />
              <span v-else class="flex h-5 w-5 items-center justify-center rounded-full border"
                :class="detail.completedAt ? 'border-ok bg-ok text-white' : 'border-current bg-surface'">
                <Check v-if="detail.completedAt" class="h-3.5 w-3.5" />
              </span>
            </button>
            <h1 class="min-w-0 text-xl font-semibold leading-8 text-ink"
              :class="detail.completedAt ? 'text-muted line-through' : ''">
              {{ detail.title }}
            </h1>
            <div v-if="detail.tags.length || detail.completedAt || detail.dueAt" class="col-start-2 flex flex-wrap items-center gap-1.5">
              <span v-for="tag in detail.tags" :key="tag" class="tag">{{ tag }}</span>
              <span v-if="detail.completedAt" class="tag border-ok/20 bg-ok-soft text-ok">
                已完成 · {{ formatDateTime(detail.completedAt) }}
              </span>
              <span v-else-if="overdue" class="tag border-warn/20 bg-warn-soft text-warn">
                <Clock class="mr-1 h-3 w-3 shrink-0" /> 已逾期 · {{ formatDateTime(detail.dueAt) }}
              </span>
              <span v-else-if="detail.dueAt" class="tag">
                <CalendarDays class="mr-1 h-3 w-3 shrink-0" /> 预期完成 {{ formatDateTime(detail.dueAt) }}
              </span>
            </div>
          </div>
          <div v-if="canEdit || canManageMembers || canDelete" class="flex flex-wrap items-center gap-1.5">
            <button v-if="canEdit" type="button" class="btn" @click="editOpen = true">编辑节点</button>
            <button v-if="canManageMembers" type="button" class="btn" @click="membersOpen = true">管理成员</button>
            <button v-if="canDelete" type="button" class="btn btn-ghost text-danger hover:bg-danger-soft" @click="deleteOpen = true">删除节点</button>
          </div>
        </header>

        <p v-if="actionError" class="rounded-md border border-danger/20 bg-danger-soft px-3 py-1.5 text-xs text-danger">{{ actionError }}</p>

        <section v-if="detail.description" class="panel px-3.5 py-3" aria-label="节点说明">
          <p class="whitespace-pre-wrap break-words text-sm leading-6 text-ink-soft">{{ detail.description }}</p>
        </section>

        <section class="panel overflow-hidden">
          <header class="flex flex-wrap items-center justify-between gap-2 border-b border-line px-3 py-2">
            <div class="flex items-center gap-1.5">
              <ListTree class="h-3.5 w-3.5 text-muted" />
              <h2 class="text-[12px] font-semibold text-ink">子节点</h2>
              <span class="rounded bg-surface-3 px-1.5 py-0.5 text-xs text-muted">{{ detail.children.length }}</span>
            </div>
            <div class="flex items-center gap-2">
              <button v-if="canReorderChildren && detail.children.length > 1" type="button" class="btn btn-ghost btn-xs" @click="orderOpen = true">
                <ArrowUpDown class="h-3.5 w-3.5" /> 调整顺序
              </button>
              <button v-if="canAddChild" type="button" class="btn btn-primary btn-xs" @click="createChildOpen = true">
                <Plus class="h-3.5 w-3.5" /> 新建子节点
              </button>
            </div>
          </header>
          <ChildRow v-for="child in detail.children" :key="child.id" :child="child" />
          <p v-if="detail.children.length === 0" class="empty-state">暂无子节点</p>
        </section>

        <CommentSection
          :key="detail.id"
          :node-id="detail.id"
          :can-comment="canComment"
          @changed="onCommentChanged"
        />

      </template>
    </div>

    <template #rail>
      <div v-if="detail && !loading && !error" class="divide-y divide-line px-4">
        <section class="py-4">
          <h3 class="rail-title">状态与时间</h3>
          <dl class="mt-3 grid grid-cols-[auto_minmax(0,1fr)] gap-x-4 gap-y-2 text-xs">
            <dt class="text-muted">状态</dt>
            <dd :class="detail.completedAt ? 'text-ok' : overdue ? 'text-warn' : 'text-ink-soft'">{{ detail.completedAt ? '已完成' : overdue ? '已逾期' : '未完成' }}</dd>
            <dt class="text-muted">预期完成</dt>
            <dd :class="overdue ? 'text-warn' : 'text-ink-soft'">{{ detail.dueAt ? formatDateTime(detail.dueAt) : '未设置' }}</dd>
            <template v-if="detail.completedAt">
              <dt class="text-muted">实际完成</dt>
              <dd>{{ formatDateTime(detail.completedAt) }}</dd>
            </template>
            <dt class="text-muted">创建时间</dt>
            <dd>{{ formatDateTime(detail.createdAt) }}</dd>
            <dt class="text-muted">最近更新</dt>
            <dd :title="formatDateTime(detail.updatedAt)">{{ timeAgo(detail.updatedAt) }}</dd>
            <dt class="text-muted">创建者</dt>
            <dd>
              <RouterLink :to="{ name: 'user', params: { subject: detail.createdBy.subject } }" class="hover:text-accent">{{ detail.createdBy.userName }}</RouterLink>
            </dd>
          </dl>
        </section>

        <section class="py-4">
          <h3 class="rail-title">直接负责人</h3>
          <div class="mt-2 flex flex-col gap-1">
            <RouterLink v-for="leader in detail.directLeaders" :key="leader.subject"
              :to="{ name: 'user', params: { subject: leader.subject } }"
              class="flex min-w-0 items-center gap-2 rounded py-1 hover:text-accent">
              <UserAvatar :name="leader.userName" :avatar="leader.avatar" size-class="h-6 w-6 text-xs" />
              <span class="min-w-0 break-words text-[13px]">{{ leader.userName }}</span>
            </RouterLink>
          </div>
          <p v-if="detail.inheritedLeaderCount" class="mt-2 text-xs leading-5 text-muted">另有 {{ detail.inheritedLeaderCount }} 位上级节点负责人可管理此节点。</p>
          <h3 class="rail-title mt-4">直接成员</h3>
          <div class="mt-2 flex flex-col gap-1">
            <RouterLink v-for="member in detail.directMembers" :key="member.subject"
              :to="{ name: 'user', params: { subject: member.subject } }"
              class="flex min-w-0 items-center gap-2 rounded py-1 hover:text-accent">
              <UserAvatar :name="member.userName" :avatar="member.avatar" size-class="h-6 w-6 text-xs" />
              <span class="min-w-0 break-words text-[13px]">{{ member.userName }}</span>
            </RouterLink>
            <p v-if="!detail.directMembers.length" class="text-xs text-muted">暂无直接成员</p>
          </div>
        </section>

        <section class="py-4">
          <h3 class="rail-title">包含下级节点</h3>
          <dl class="mt-3 grid grid-cols-[1fr_auto] gap-x-4 gap-y-2 text-xs">
            <dt class="text-muted">成员（含负责人）</dt><dd class="tabular-nums">{{ detail.effectiveMemberCount }}</dd>
            <dt class="text-muted">节点</dt><dd class="tabular-nums">{{ detail.subtreeNodeCount }}</dd>
            <dt class="text-muted">评论</dt><dd class="tabular-nums">{{ detail.subtreeCommentCount }}</dd>
            <dt class="text-muted">图片</dt><dd class="tabular-nums">{{ detail.subtreeAttachmentCount }}</dd>
          </dl>
        </section>
      </div>
    </template>
  </AppShell>

  <template v-if="detail">
    <NodeFormDialog
      v-model:open="editOpen"
      mode="edit"
      :node="detail"
      @saved="onEditSaved"
    />
    <NodeFormDialog
      v-model:open="createChildOpen"
      mode="create"
      :parent-id="detail.id"
      :parent-title="detail.title"
      @saved="onChildCreated"
    />
    <MembersDialog
      :open="membersOpen"
      :node="detail"
      @close="membersOpen = false"
      @saved="onMembersSaved"
    />
    <OrderDialog
      :open="orderOpen"
      :parent-id="detail.id"
      :parent-title="detail.title"
      :children="detail.children"
      @close="orderOpen = false"
      @saved="onOrdered"
    />
    <DeleteNodeDialog
      :open="deleteOpen"
      :node="detail"
      @close="deleteOpen = false"
      @deleted="onDeleted"
    />
  </template>
</template>
