<script setup lang="ts">
import { computed, defineAsyncComponent, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import {
  NAlert,
  NButton,
  NCard,
  NEmpty,
  NGrid,
  NGi,
  NH1,
  NList,
  NListItem,
  NSkeleton,
  NSpace,
  NStatistic,
} from 'naive-ui'
import AppShell from '@/components/AppShell.vue'
import NodeListItem from '@/components/NodeListItem.vue'
import CommentPreview from '@/components/CommentPreview.vue'
import { useAuth } from '@/composables/useAuth'
import { useNodeStore } from '@/composables/useNodeStore'
import { fetchOverview, type NodeDetail, type Overview } from '@/lib/api'
import { errorMessage, isOverdue } from '@/lib/utils'
const NodeFormDialog = defineAsyncComponent(
    () => import('@/components/NodeFormDialog.vue'),
  ),
  OrderDialog = defineAsyncComponent(
    () => import('@/components/OrderDialog.vue'),
  )
const router = useRouter(),
  { isManager } = useAuth(),
  {
    nodes,
    loading: nodesLoading,
    error: nodesError,
    reload: reloadNodes,
  } = useNodeStore()
const overview = ref<Overview | null>(null),
  loading = ref(true),
  error = ref<string | null>(null),
  createOpen = ref(false),
  orderOpen = ref(false)
const statistics = computed(() => [
  { label: '全部节点', value: nodes.value.length },
  {
    label: '我负责',
    value: nodes.value.filter((node) => node.myRelation === 'leader').length,
  },
  {
    label: '未完成',
    value: nodes.value.filter((node) => !node.completedAt).length,
  },
  {
    label: '已逾期',
    value: nodes.value.filter((node) => isOverdue(node.dueAt, node.completedAt))
      .length,
  },
])
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
async function created(node: NodeDetail) {
  createOpen.value = false
  await refresh()
  await router.push({ name: 'node', params: { id: node.id } })
}
async function ordered() {
  orderOpen.value = false
  await refresh()
}
onMounted(load)
</script>
<template>
  <AppShell rail-title="最近评论"
    ><section class="overview-page">
      <div class="overview-heading">
        <NH1>总览</NH1
        ><NSpace v-if="isManager"
          ><NButton
            v-if="(overview?.roots.length ?? 0) > 1"
            @click="orderOpen = true"
            >根节点排序</NButton
          ><NButton type="primary" @click="createOpen = true"
            >新建根节点</NButton
          ></NSpace
        >
      </div>
      <NSpace vertical :size="24"
        ><NAlert v-if="error || nodesError" type="error"
          >{{ error || nodesError }}
          <NButton text @click="refresh">重试</NButton></NAlert
        ><NSkeleton v-if="loading" text :repeat="5" /><template
          v-else-if="overview && !error"
          ><NCard v-if="!nodesLoading && !nodesError" size="small"
            ><NGrid cols="2 s:4" responsive="screen" :x-gap="16" :y-gap="16"
              ><NGi v-for="stat in statistics" :key="stat.label"
                ><NStatistic
                  :label="stat.label"
                  :value="stat.value" /></NGi></NGrid></NCard
          ><NCard title="根节点" size="small"
            ><NList v-if="overview.roots.length" hoverable
              ><NodeListItem
                v-for="root in overview.roots"
                :key="root.id"
                :node="root"
                :leaders="root.leaders" /></NList
            ><NEmpty
              v-else
              :description="
                isManager
                  ? '暂无根节点，可新建根节点开始安排工作。'
                  : '暂无根节点，请联系部门管理员创建。'
              " /></NCard></template
      ></NSpace>
    </section>
    <template #rail
      ><div class="recent-comments">
        <NSkeleton v-if="loading" text :repeat="5" /><NAlert
          v-else-if="error"
          type="error"
          >评论加载失败 <NButton text @click="load">重试</NButton></NAlert
        ><NList v-else-if="overview?.recentComments.length"
          ><NListItem
            v-for="comment in overview.recentComments"
            :key="comment.id"
            ><NSpace vertical :size="8"
              ><RouterLink
                :to="{ name: 'node', params: { id: comment.nodeId } }"
                class="recent-node"
                >{{
                  comment.rootTitle === comment.nodeTitle
                    ? comment.nodeTitle
                    : `${comment.rootTitle} / ${comment.nodeTitle}`
                }}</RouterLink
              ><CommentPreview :comment="comment" /></NSpace></NListItem></NList
        ><NEmpty v-else size="small" description="还没有评论" /></div></template
  ></AppShell>
  <NodeFormDialog
    v-if="createOpen"
    v-model:open="createOpen"
    mode="create"
    @saved="created"
  /><OrderDialog
    v-if="orderOpen"
    :open="orderOpen"
    :parent-id="null"
    parent-title=""
    :children="overview?.roots ?? []"
    @close="orderOpen = false"
    @saved="ordered"
  />
</template>
<style scoped>
.overview-page {
  max-width: 1080px;
  margin: auto;
  padding: 24px;
}
.overview-heading {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
  flex-wrap: wrap;
  margin-bottom: 24px;
}
.overview-heading h1 {
  margin: 0;
  font-size: 24px;
}
.recent-comments {
  padding: 16px;
}
.recent-node {
  color: inherit;
  text-decoration: none;
  overflow-wrap: anywhere;
  font-weight: 500;
}
.recent-node:hover {
  color: var(--guild-primary);
}
@media (max-width: 639px) {
  .overview-page {
    padding: 16px;
  }
}
</style>
