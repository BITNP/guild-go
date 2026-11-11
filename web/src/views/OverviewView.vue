<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
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
  NSpace,
  NStatistic,
  NRadioButton,
  NRadioGroup,
  NSpin,
  NText,
} from 'naive-ui'
import AppShell from '@/components/AppShell.vue'
import NodeListItem from '@/components/NodeListItem.vue'
import ActivityPreview from '@/components/ActivityPreview.vue'
import SectionHeading from '@/components/SectionHeading.vue'
import NodeFormDialog from '@/components/NodeFormDialog.vue'
import OrderDialog from '@/components/OrderDialog.vue'
import { useAuth } from '@/composables/useAuth'
import { useNodeStore } from '@/composables/useNodeStore'
import { fetchOverview, type ActivityScope, type NodeDetail, type Overview } from '@/lib/api'
import { errorMessage, isOverdue } from '@/lib/utils'
const router = useRouter(),
  { isManager } = useAuth(),
  {
    nodes,
    error: nodesError,
    reload: reloadNodes,
  } = useNodeStore()
const overview = ref<Overview | null>(null),
  loading = ref(true),
  error = ref<string | null>(null),
  createOpen = ref(false),
  orderOpen = ref(false)
const activityScope = ref<ActivityScope>('all')
let loadVersion = 0
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
  const version = ++loadVersion
  loading.value = true
  error.value = null
  try {
    const result = await fetchOverview(activityScope.value)
    if (version === loadVersion) overview.value = result
  } catch (err) {
    if (version === loadVersion) error.value = errorMessage(err, '无法加载总览')
  } finally {
    if (version === loadVersion) loading.value = false
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
watch(activityScope, load)
</script>
<template>
  <AppShell rail-title="最近发布"
    ><section class="overview-page" :aria-busy="loading">
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
      <NSpace vertical :size="20"
        ><NAlert v-if="error || nodesError" type="error"
          >{{ error || nodesError }}
          <NButton text @click="refresh">重试</NButton></NAlert
        ><template v-if="overview && !error"
          ><NCard v-if="!nodesError" size="small"
            ><NGrid :cols="4" :x-gap="16"
              ><NGi
                v-for="stat in statistics"
                :key="stat.label"
                class="statistic-cell"
                ><NStatistic
                  :label="stat.label"
                  :value="stat.value"
                  :value-style="{
                    color:
                      stat.label === '已逾期' && stat.value
                        ? 'var(--guild-warning)'
                        : undefined,
                  }" /></NGi></NGrid
          ></NCard>
          <section class="root-section" aria-label="根节点">
            <SectionHeading title="根节点" :count="overview.roots.length" />
            <NCard size="small"
              ><NList v-if="overview.roots.length"
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
                "
            /></NCard></section></template
      ></NSpace>
    </section>
    <template #rail
      ><div class="recent-activities" :aria-busy="loading">
        <NRadioGroup v-model:value="activityScope" size="small" aria-label="动态范围">
          <NRadioButton value="all">全部动态</NRadioButton>
          <NRadioButton value="related">与我有关</NRadioButton>
        </NRadioGroup>
        <NText v-if="activityScope === 'related'" depth="3" class="activity-hint">
          我创建、负责或参与的节点，以及我的评论
        </NText>
        <div v-if="loading" class="activity-loading" role="status" aria-label="正在加载动态">
          <NSpin size="small" />
        </div>
        <NAlert
          v-else-if="error"
          type="error"
          >动态加载失败 <NButton text @click="load">重试</NButton></NAlert
        ><NList v-else-if="overview?.recentActivities.length"
          ><NListItem
            v-for="activity in overview.recentActivities"
            :key="`${activity.kind}-${activity.id}`"
            class="recent-activity"
            ><NSpace vertical :size="8"
              ><RouterLink
                :to="{ name: 'node', params: { id: activity.nodeId } }"
                class="recent-node"
                >{{
                  activity.rootTitle === activity.nodeTitle
                    ? activity.nodeTitle
                    : `${activity.rootTitle} / ${activity.nodeTitle}`
                }}</RouterLink
              ><ActivityPreview :activity="activity" /></NSpace></NListItem></NList
        ><NEmpty v-else-if="overview" size="small" :description="activityScope === 'related' ? '暂无与我有关的动态' : '暂无动态'" /></div></template
  ></AppShell>
  <NodeFormDialog
    v-model:open="createOpen"
    mode="create"
    @saved="created"
  /><OrderDialog
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
  padding: 20px;
}
.overview-heading {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
  flex-wrap: wrap;
  margin-bottom: 20px;
}
.overview-heading h1 {
  margin: 0;
  font-size: var(--guild-page-title-size);
  font-weight: 650;
}
.root-section {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.recent-activities {
  padding: 16px;
  font-size: 13px;
}
.recent-activity:first-child {
  padding-top: 0;
}
.recent-activities > :deep(.n-radio-group) {
  margin-bottom: 12px;
}
.activity-hint {
  display: block;
  margin-bottom: 12px;
  font-size: 12px;
}
.activity-loading {
  padding: 24px;
  text-align: center;
}
.recent-node {
  color: inherit;
  text-decoration: none;
  overflow-wrap: anywhere;
  font-weight: 600;
}
.recent-node:hover {
  color: var(--guild-primary);
}
.statistic-cell + .statistic-cell {
  padding-left: 16px;
  border-left: 1px solid var(--guild-border);
}
@media (max-width: 639px) {
  .overview-page {
    padding: 16px;
  }
  .statistic-cell + .statistic-cell {
    padding-left: 10px;
  }
  .statistic-cell :deep(.n-statistic .n-statistic__label) {
    font-size: 12px;
  }
  .statistic-cell :deep(.n-statistic-value__content) {
    font-size: 22px;
  }
}
</style>
