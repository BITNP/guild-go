<script setup lang="ts">
import { computed, defineAsyncComponent, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  NAlert,
  NBreadcrumb,
  NBreadcrumbItem,
  NButton,
  NCard,
  NEmpty,
  NH1,
  NList,
  NSkeleton,
  NSpace,
  NSpin,
  NTag,
  NText,
} from 'naive-ui'
import { CalendarDays, Check, Circle } from '@lucide/vue'
import AppShell from '@/components/AppShell.vue'
import NodeDetails from '@/components/NodeDetails.vue'
import NodeListItem from '@/components/NodeListItem.vue'
import NodeStatusTag from '@/components/NodeStatusTag.vue'
import CommentSection from '@/components/CommentSection.vue'
import { useNodeStore } from '@/composables/useNodeStore'
import { fetchNode, updateNode, type NodeDetail } from '@/lib/api'
import { errorMessage, formatDateTime } from '@/lib/utils'
const NodeFormDialog = defineAsyncComponent(
  () => import('@/components/NodeFormDialog.vue'),
)
const MembersDialog = defineAsyncComponent(
  () => import('@/components/MembersDialog.vue'),
)
const OrderDialog = defineAsyncComponent(
  () => import('@/components/OrderDialog.vue'),
)
const DeleteNodeDialog = defineAsyncComponent(
  () => import('@/components/DeleteNodeDialog.vue'),
)
const route = useRoute(),
  router = useRouter()
const { reload: reloadNodes } = useNodeStore()
const nodeId = computed(() => Number(route.params.id))
const detail = ref<NodeDetail | null>(null),
  loading = ref(true),
  error = ref<string | null>(null),
  actionError = ref<string | null>(null),
  toggling = ref(false)
const editOpen = ref(false),
  createOpen = ref(false),
  membersOpen = ref(false),
  orderOpen = ref(false),
  deleteOpen = ref(false)
async function load(silent = false) {
  const requested = nodeId.value
  if (!Number.isSafeInteger(requested) || requested <= 0) {
    detail.value = null
    error.value = '节点不存在'
    loading.value = false
    return
  }
  if (!silent) {
    loading.value = true
    detail.value = null
  }
  error.value = null
  actionError.value = null
  try {
    const node = await fetchNode(requested)
    if (requested === nodeId.value) detail.value = node
  } catch (err) {
    if (requested === nodeId.value) {
      error.value = errorMessage(err, '无法加载节点')
      detail.value = null
    }
  } finally {
    if (requested === nodeId.value) loading.value = false
  }
}
async function toggleComplete() {
  const node = detail.value
  if (!node || !node.permissions.canEdit || toggling.value) return
  toggling.value = true
  actionError.value = null
  try {
    const updated = await updateNode(node.id, {
      title: node.title,
      description: node.description,
      dueAt: node.dueAt,
      tags: node.tags,
      completed: !node.completedAt,
    })
    if (node.id === nodeId.value) detail.value = updated
    await reloadNodes()
  } catch (err) {
    if (node.id === nodeId.value)
      actionError.value = errorMessage(err, '更新完成状态失败')
  } finally {
    toggling.value = false
  }
}
function onSaved(node: NodeDetail) {
  if (node.id === nodeId.value) detail.value = node
  editOpen.value = false
  membersOpen.value = false
  void reloadNodes()
}
async function onCreated(node: NodeDetail) {
  createOpen.value = false
  await reloadNodes()
  await router.push({ name: 'node', params: { id: node.id } })
}
async function onOrdered() {
  orderOpen.value = false
  await Promise.all([load(), reloadNodes()])
}
async function onDeleted() {
  const parentId = detail.value?.parentId
  deleteOpen.value = false
  await reloadNodes()
  await router.push(
    parentId ? { name: 'node', params: { id: parentId } } : { name: 'home' },
  )
}
async function onCommentChanged() {
  await Promise.all([load(true), reloadNodes()])
}
watch(
  nodeId,
  () => {
    editOpen.value = false
    createOpen.value = false
    membersOpen.value = false
    orderOpen.value = false
    deleteOpen.value = false
    void load()
  },
  { immediate: true },
)
</script>
<template>
  <AppShell
    ><section class="node-page">
      <NSpin :show="loading"
        ><NSpace vertical :size="16">
          <NSkeleton v-if="loading" text :repeat="5" />
          <NAlert v-if="error" type="error"
            >{{ error }} <NButton text @click="load()">重试</NButton>
            <NButton text @click="router.push({ name: 'home' })"
              >返回总览</NButton
            ></NAlert
          >
          <template v-else-if="detail">
            <header class="node-header">
              <NBreadcrumb aria-label="节点路径" class="node-breadcrumb"
                ><NBreadcrumbItem
                  ><RouterLink :to="{ name: 'home' }"
                    >总览</RouterLink
                  ></NBreadcrumbItem
                ><NBreadcrumbItem
                  v-for="item in detail.breadcrumb"
                  :key="item.id"
                  ><RouterLink
                    :to="{ name: 'node', params: { id: item.id } }"
                    >{{ item.title }}</RouterLink
                  ></NBreadcrumbItem
                ></NBreadcrumb
              >
              <div class="node-heading">
                <div class="node-title">
                  <NButton
                    circle
                    quaternary
                    :type="detail.completedAt ? 'success' : 'default'"
                    :loading="toggling"
                    :disabled="!detail.permissions.canEdit"
                    :aria-label="
                      detail.completedAt ? '标记为未完成' : '标记为已完成'
                    "
                    :aria-pressed="!!detail.completedAt"
                    @click="toggleComplete"
                    ><Check v-if="detail.completedAt" :size="24" /><Circle
                      v-else
                      :size="26" /></NButton
                  ><NH1 :class="{ completed: !!detail.completedAt }">{{
                    detail.title
                  }}</NH1>
                </div>
                <NSpace
                  ><NButton
                    v-if="detail.permissions.canEdit"
                    @click="editOpen = true"
                    >编辑节点</NButton
                  ><NButton
                    v-if="detail.permissions.canManageMembers"
                    @click="membersOpen = true"
                    >管理成员</NButton
                  ><NButton
                    v-if="detail.permissions.canDelete"
                    type="error"
                    quaternary
                    @click="deleteOpen = true"
                    >删除节点</NButton
                  ></NSpace
                >
              </div>
              <NSpace :size="8" align="center" class="node-metadata"
                ><NodeStatusTag
                  :due-at="detail.dueAt"
                  :completed-at="detail.completedAt"
                /><NTag v-for="tag in detail.tags" :key="tag" size="small">{{
                  tag
                }}</NTag>
                <span v-if="detail.dueAt" class="due-date"
                  ><CalendarDays :size="13" aria-hidden="true" />预期完成
                  {{ formatDateTime(detail.dueAt) }}</span
                >
              </NSpace>
            </header>
            <NAlert v-if="actionError" type="error">{{ actionError }}</NAlert>
            <NCard v-if="detail.description" size="small" aria-label="节点说明"
              ><p class="node-description">{{ detail.description }}</p></NCard
            >
            <NCard size="small"
              ><template #header
                ><span
                  >子节点<NText depth="3" class="section-count">{{
                    detail.children.length
                  }}</NText></span
                ></template
              ><template #header-extra
                ><NSpace :size="8"
                  ><NButton
                    v-if="
                      detail.permissions.canReorderChildren &&
                      detail.children.length > 1
                    "
                    size="small"
                    @click="orderOpen = true"
                    >调整顺序</NButton
                  ><NButton
                    v-if="detail.permissions.canAddChild"
                    type="primary"
                    size="small"
                    @click="createOpen = true"
                    >新建子节点</NButton
                  ></NSpace
                ></template
              ><NList v-if="detail.children.length"
                ><NodeListItem
                  v-for="child in detail.children"
                  :key="child.id"
                  :node="child" /></NList
              ><NEmpty v-else description="暂无子节点" size="small"
            /></NCard>
            <CommentSection
              :key="detail.id"
              :node-id="detail.id"
              :can-comment="detail.permissions.canComment"
              @changed="onCommentChanged"
            />
          </template> </NSpace
      ></NSpin>
    </section>
    <template #rail
      ><NodeDetails v-if="detail && !error" :node="detail" /></template
  ></AppShell>
  <template v-if="detail">
    <NodeFormDialog
      v-if="editOpen"
      v-model:open="editOpen"
      mode="edit"
      :node="detail"
      @saved="onSaved"
    />
    <NodeFormDialog
      v-if="createOpen"
      v-model:open="createOpen"
      mode="create"
      :parent-id="detail.id"
      :parent-title="detail.title"
      @saved="onCreated"
    />
    <MembersDialog
      v-if="membersOpen"
      :open="membersOpen"
      :node="detail"
      @close="membersOpen = false"
      @saved="onSaved"
    />
    <OrderDialog
      v-if="orderOpen"
      :open="orderOpen"
      :parent-id="detail.id"
      :parent-title="detail.title"
      :children="detail.children"
      @close="orderOpen = false"
      @saved="onOrdered"
    />
    <DeleteNodeDialog
      v-if="deleteOpen"
      :open="deleteOpen"
      :node="detail"
      @close="deleteOpen = false"
      @deleted="onDeleted"
    />
  </template>
</template>
<style scoped>
.node-page {
  max-width: 1080px;
  margin: auto;
  padding: 20px;
}
.node-header {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.node-breadcrumb {
  font-size: 12px;
}
.node-metadata {
  padding-left: 46px;
}
.due-date {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  color: var(--guild-muted);
  font-size: 12px;
}
.section-count {
  margin-left: 6px;
  font-size: 13px;
  font-weight: 400;
}
.node-heading {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 16px;
  flex-wrap: wrap;
}
.node-title {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  flex: 1;
  min-width: 200px;
}
.node-title h1 {
  font-size: 26px;
  line-height: 34px;
  font-weight: 600;
  margin: 0;
  overflow-wrap: anywhere;
}
.node-title > .n-button {
  flex: none;
}
.completed {
  text-decoration: line-through;
  color: var(--guild-muted);
}
.node-description {
  margin: 0;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
  line-height: 1.8;
}
@media (max-width: 639px) {
  .node-page {
    padding: 16px;
  }
  .node-title h1 {
    font-size: 22px;
  }
}
</style>
