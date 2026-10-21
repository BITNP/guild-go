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
  MessageSquare,
  Pencil,
  Plus,
  Trash2,
  Users,
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
    <div class="flex flex-col gap-3 p-4 lg:p-5">
      <p v-if="loading" class="flex items-center gap-2 py-10 text-[13px] text-muted">
        <Loader2 class="h-4 w-4 animate-spin" /> 正在加载节点…
      </p>
      <div v-else-if="error" class="rounded-md border border-danger/20 bg-danger-soft px-3 py-3 text-[12px] text-danger">
        {{ error }}
        <RouterLink :to="{ name: 'home' }" class="ml-2 underline">返回总览</RouterLink>
      </div>

      <template v-else-if="detail">
        <header class="flex flex-col gap-2">
          <div class="flex flex-wrap items-center gap-1 text-[11px] text-muted">
            <RouterLink :to="{ name: 'home' }" class="hover:text-accent">总览</RouterLink>
            <template v-for="item in detail.breadcrumb" :key="item.id">
              <ChevronRight class="h-3 w-3 text-faint" />
              <RouterLink :to="{ name: 'node', params: { id: item.id } }" class="max-w-48 truncate hover:text-accent">
                {{ item.title }}
              </RouterLink>
            </template>
          </div>

          <div class="flex flex-wrap items-start justify-between gap-3">
            <div class="min-w-0">
              <div class="flex items-start gap-2">
                <button
                  type="button"
                  class="mt-0.5 flex h-5 w-5 shrink-0 items-center justify-center rounded-full border transition-colors disabled:cursor-default"
                  :class="detail.completedAt
                    ? 'border-ok bg-ok text-white'
                    : 'border-line-strong bg-surface text-transparent hover:border-ok hover:text-ok'"
                  :disabled="!canEdit || toggling"
                  :title="detail.completedAt ? '重新打开' : '标记完成'"
                  @click="toggleComplete"
                >
                  <Check class="h-3.5 w-3.5" />
                </button>
                <div class="min-w-0">
                  <h1
                    class="text-xl font-semibold leading-7 tracking-tight text-ink"
                    :class="detail.completedAt ? 'text-muted line-through' : ''"
                  >
                    {{ detail.title }}
                  </h1>
                  <div class="mt-1 flex flex-wrap items-center gap-1.5">
                    <span v-for="tag in detail.tags" :key="tag" class="tag"># {{ tag }}</span>
                    <span v-if="detail.completedAt" class="inline-flex items-center gap-0.5 rounded bg-ok-soft px-1.5 py-0.5 text-[10px] text-ok">
                      <Check class="h-2.5 w-2.5" /> 已完成 {{ formatDateTime(detail.completedAt) }}
                    </span>
                    <span v-else-if="overdue" class="inline-flex items-center gap-0.5 rounded bg-warn-soft px-1.5 py-0.5 text-[10px] text-warn">
                      <Clock class="h-2.5 w-2.5" /> 已逾期 · 预期 {{ formatDateTime(detail.dueAt) }}
                    </span>
                    <span v-else-if="detail.dueAt" class="inline-flex items-center gap-0.5 rounded bg-surface-2 px-1.5 py-0.5 text-[10px] text-muted">
                      <CalendarDays class="h-2.5 w-2.5" /> 预期 {{ formatDateTime(detail.dueAt) }}
                    </span>
                  </div>
                </div>
              </div>
            </div>

            <div class="flex flex-wrap items-center gap-1.5">
              <button v-if="canAddChild" type="button" class="btn" @click="createChildOpen = true">
                <Plus class="h-3.5 w-3.5" /> 子节点
              </button>
              <button v-if="canEdit" type="button" class="btn" @click="editOpen = true">
                <Pencil class="h-3.5 w-3.5" /> 编辑
              </button>
              <button v-if="canManageMembers" type="button" class="btn" @click="membersOpen = true">
                <Users class="h-3.5 w-3.5" /> 成员
              </button>
              <button v-if="canReorderChildren && detail.children.length > 1" type="button" class="btn" @click="orderOpen = true">
                <ArrowUpDown class="h-3.5 w-3.5" /> 排序
              </button>
              <button v-if="canDelete" type="button" class="btn text-danger hover:border-danger/40 hover:bg-danger-soft" @click="deleteOpen = true">
                <Trash2 class="h-3.5 w-3.5" /> 删除
              </button>
            </div>
          </div>
        </header>

        <p v-if="actionError" class="rounded-md border border-danger/20 bg-danger-soft px-3 py-1.5 text-[11px] text-danger">{{ actionError }}</p>

        <section class="panel px-3.5 py-3">
          <p v-if="detail.description" class="whitespace-pre-wrap break-words text-[13px] leading-6 text-ink-soft">
            {{ detail.description }}
          </p>
          <p v-else class="text-[12px] text-faint">暂无描述。</p>
          <div class="mt-2 flex flex-wrap items-center gap-x-4 gap-y-1 border-t border-line pt-2 text-[10px] text-faint">
            <span class="inline-flex items-center gap-1">
              <UserAvatar :name="detail.createdBy.userName" :avatar="detail.createdBy.avatar" size-class="h-4 w-4 text-[8px]" />
              由 {{ detail.createdBy.userName }} 创建
            </span>
            <span>创建 {{ formatDateTime(detail.createdAt) }}</span>
            <span>更新 {{ timeAgo(detail.updatedAt) }}</span>
            <span>排序 #{{ detail.sortOrder + 1 }}</span>
          </div>
        </section>

        <section class="panel overflow-hidden">
          <header class="flex items-center justify-between border-b border-line px-3 py-2">
            <div class="flex items-center gap-1.5">
              <ListTree class="h-3.5 w-3.5 text-muted" />
              <h2 class="text-[12px] font-semibold text-ink">子节点</h2>
              <span class="rounded bg-surface-3 px-1.5 py-0.5 text-[10px] text-muted">{{ detail.children.length }}</span>
            </div>
            <button v-if="canAddChild" type="button" class="btn btn-xs" @click="createChildOpen = true">
              <Plus class="h-3 w-3" /> 新建
            </button>
          </header>
          <ChildRow v-for="child in detail.children" :key="child.id" :child="child" />
          <p v-if="detail.children.length === 0" class="empty-state">还没有子节点。{{ canAddChild ? '点击“新建”继续拆解。' : '' }}</p>
        </section>

        <CommentSection
          :node-id="detail.id"
          :can-comment="canComment"
          @changed="onCommentChanged"
        />

        <div class="flex items-center gap-3 pb-2 text-[10px] text-faint">
          <span>{{ detail.effectiveMemberCount }} 位有效成员</span>
          <span>{{ detail.subtreeNodeCount }} 个节点（含子树）</span>
          <span>{{ detail.subtreeCommentCount }} 条评论</span>
        </div>
      </template>
    </div>

    <template #rail>
      <div v-if="detail" class="flex flex-col gap-3 p-2.5">
        <section class="panel px-3 py-2.5">
          <h2 class="rail-title">状态</h2>
          <dl class="mt-2 grid grid-cols-[auto_1fr] gap-x-3 gap-y-1.5 text-[11px]">
            <dt class="text-muted">完成</dt>
            <dd class="text-ink-soft">{{ detail.completedAt ? formatDateTime(detail.completedAt) : '未完成' }}</dd>
            <dt class="text-muted">预期</dt>
            <dd :class="overdue ? 'font-medium text-warn' : 'text-ink-soft'">{{ formatDateTime(detail.dueAt) }}</dd>
            <dt class="text-muted">创建</dt>
            <dd class="text-ink-soft">{{ formatDateTime(detail.createdAt) }}</dd>
            <dt class="text-muted">更新</dt>
            <dd class="text-ink-soft">{{ timeAgo(detail.updatedAt) }}</dd>
            <dt class="text-muted">标签</dt>
            <dd class="flex flex-wrap gap-1">
              <span v-for="tag in detail.tags" :key="tag" class="tag">{{ tag }}</span>
              <span v-if="detail.tags.length === 0" class="text-faint">—</span>
            </dd>
          </dl>
        </section>

        <section class="panel px-3 py-2.5">
          <div class="flex items-center justify-between">
            <h2 class="rail-title">负责人</h2>
            <span class="text-[10px] text-faint">{{ detail.directLeaders.length }} 直接 · {{ detail.inheritedLeaderCount }} 继承</span>
          </div>
          <div class="mt-1.5 flex flex-col gap-1">
            <RouterLink
              v-for="leader in detail.directLeaders"
              :key="leader.subject"
              :to="{ name: 'user', params: { subject: leader.subject } }"
              class="flex items-center gap-1.5 rounded px-1 py-0.5 hover:bg-surface-2"
            >
              <UserAvatar :name="leader.userName" :avatar="leader.avatar" size-class="h-5 w-5 text-[8px]" />
              <span class="truncate text-[12px] text-ink-soft">{{ leader.userName }}</span>
            </RouterLink>
            <p v-if="detail.directLeaders.length === 0" class="text-[11px] text-faint">暂无直接负责人</p>
          </div>
          <div class="mt-2 border-t border-line pt-2">
            <h3 class="text-[10px] font-medium text-muted">成员</h3>
            <div class="mt-1 flex flex-col gap-1">
              <RouterLink
                v-for="member in detail.directMembers"
                :key="member.subject"
                :to="{ name: 'user', params: { subject: member.subject } }"
                class="flex items-center gap-1.5 rounded px-1 py-0.5 hover:bg-surface-2"
              >
                <UserAvatar :name="member.userName" :avatar="member.avatar" size-class="h-5 w-5 text-[8px]" />
                <span class="truncate text-[12px] text-ink-soft">{{ member.userName }}</span>
              </RouterLink>
              <p v-if="detail.directMembers.length === 0" class="text-[11px] text-faint">暂无直接成员</p>
            </div>
          </div>
          <button v-if="canManageMembers" type="button" class="btn btn-xs mt-2 w-full" @click="membersOpen = true">
            <Users class="h-3 w-3" /> 管理成员与继承
          </button>
        </section>

        <section class="panel px-3 py-2.5">
          <h2 class="rail-title">数据</h2>
          <div class="mt-1.5 grid grid-cols-2 gap-1.5 text-[11px]">
            <div class="rounded border border-line bg-surface-2 px-2 py-1.5">
              <p class="text-[10px] text-muted">有效成员</p>
              <p class="font-semibold text-ink">{{ detail.effectiveMemberCount }}</p>
            </div>
            <div class="rounded border border-line bg-surface-2 px-2 py-1.5">
              <p class="text-[10px] text-muted">子树节点</p>
              <p class="font-semibold text-ink">{{ detail.subtreeNodeCount }}</p>
            </div>
            <div class="rounded border border-line bg-surface-2 px-2 py-1.5">
              <p class="text-[10px] text-muted">评论</p>
              <p class="font-semibold text-ink">{{ detail.subtreeCommentCount }}</p>
            </div>
            <div class="rounded border border-line bg-surface-2 px-2 py-1.5">
              <p class="text-[10px] text-muted">附件</p>
              <p class="font-semibold text-ink">{{ detail.subtreeAttachmentCount }}</p>
            </div>
          </div>
        </section>

        <section class="panel px-3 py-2.5">
          <h2 class="rail-title">快捷操作</h2>
          <div class="mt-1.5 flex flex-col gap-1">
            <button v-if="canAddChild" type="button" class="btn btn-xs justify-start" @click="createChildOpen = true">
              <Plus class="h-3 w-3" /> 新建子节点
            </button>
            <button v-if="canEdit" type="button" class="btn btn-xs justify-start" @click="editOpen = true">
              <Pencil class="h-3 w-3" /> 编辑节点
            </button>
            <button v-if="canManageMembers" type="button" class="btn btn-xs justify-start" @click="membersOpen = true">
              <Users class="h-3 w-3" /> 管理成员
            </button>
            <button v-if="canDelete" type="button" class="btn btn-xs justify-start text-danger" @click="deleteOpen = true">
              <Trash2 class="h-3 w-3" /> 删除节点
            </button>
            <p v-if="!canEdit && !canDelete" class="text-[11px] text-faint">
              你是只读访问。评论权限取决于有效成员身份。
            </p>
          </div>
        </section>

        <div class="flex items-center gap-2 px-1 pb-2 text-[10px] text-faint">
          <MessageSquare class="h-3 w-3" />
          <span>子节点最新评论会显示在父节点页。</span>
        </div>
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