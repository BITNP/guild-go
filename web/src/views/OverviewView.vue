<script setup lang="ts">
import type { NodeDetail, Overview } from "@/api";

const router = useRouter();
const { isManager } = useAuth();
const { nodes, error: nodesError, reload: reloadNodes } = useNodeStore();
const overview = ref<Overview | null>(null);
const loading = ref(true);
const error = ref<string | null>(null);
const createOpen = ref(false);
const orderOpen = ref(false);
const statistics = computed(() => [
  { label: "全部节点", value: nodes.value.length },
  {
    label: "我负责",
    value: nodes.value.filter((node) => node.myRelation === "leader").length,
  },
  {
    label: "未完成",
    value: nodes.value.filter((node) => !node.completedAt).length,
  },
  {
    label: "已逾期",
    value: nodes.value.filter((node) => isOverdue(node.dueAt, node.completedAt))
      .length,
  },
]);

async function load() {
  loading.value = true;
  error.value = null;
  try {
    overview.value = await fetchOverview();
  } catch (err) {
    error.value = errorMessage(err, "无法加载总览");
  } finally {
    loading.value = false;
  }
}

async function refresh() {
  await Promise.all([load(), reloadNodes()]);
}

async function created(node: NodeDetail) {
  createOpen.value = false;
  await refresh();
  await router.push({ name: "node", params: { id: node.id } });
}

async function ordered() {
  orderOpen.value = false;
  await refresh();
}

onMounted(load);
</script>

<template>
  <AppShell rail-title="最近发布">
    <section :aria-busy="loading" class="overview-page">
      <div class="overview-heading">
        <NH1>总览</NH1>
        <NSpace v-if="isManager">
          <NButton
            v-if="(overview?.roots.length ?? 0) > 1"
            @click="orderOpen = true"
          >
            根节点排序
          </NButton>
          <NButton type="primary" @click="createOpen = true">
            新建根节点
          </NButton>
        </NSpace>
      </div>
      <NSpace :size="20" vertical>
        <NAlert v-if="error || nodesError" type="error">
          {{ error || nodesError }}
          <NButton text @click="refresh">重试</NButton>
        </NAlert>
        <template v-if="overview && !error">
          <NCard v-if="!nodesError" size="small">
            <NGrid :cols="4" :x-gap="16">
              <NGi
                v-for="stat in statistics"
                :key="stat.label"
                class="statistic-cell"
              >
                <NStatistic
                  :label="stat.label"
                  :value="stat.value"
                  :value-style="{
                    color:
                      stat.label === '已逾期' && stat.value
                        ? 'var(--guild-warning)'
                        : undefined,
                  }"
                />
              </NGi>
            </NGrid>
          </NCard>
          <section aria-label="根节点" class="root-section">
            <SectionHeading :count="overview.roots.length" title="根节点" />
            <NCard size="small">
              <NList v-if="overview.roots.length > 0">
                <NodeListItem
                  v-for="root in overview.roots"
                  :key="root.id"
                  :leaders="root.leaders"
                  :node="root"
                />
              </NList>
              <NEmpty
                v-else
                :description="
                  isManager
                    ? '暂无根节点，可新建根节点开始安排工作。'
                    : '暂无根节点，请联系部门管理员创建。'
                "
              />
            </NCard>
          </section>
        </template>
      </NSpace>
    </section>
    <template #rail>
      <div class="recent-comments">
        <NAlert v-if="error" type="error">
          进展加载失败
          <NButton text @click="load">重试</NButton>
        </NAlert>
        <NList v-else-if="overview?.recentComments.length">
          <NListItem
            v-for="comment in overview.recentComments"
            :key="comment.id"
            class="recent-comment"
          >
            <NSpace :size="8" vertical>
              <RouterLink
                class="recent-node"
                :to="{ name: 'node', params: { id: comment.nodeId } }"
              >
                {{
                  comment.rootTitle === comment.nodeTitle
                    ? comment.nodeTitle
                    : `${comment.rootTitle} / ${comment.nodeTitle}`
                }}
              </RouterLink>
              <CommentPreview :comment />
            </NSpace>
          </NListItem>
        </NList>
        <NEmpty
          v-else-if="overview"
          description="还没有人发布进展"
          size="small"
        />
      </div>
    </template>
  </AppShell>
  <NodeFormDialog v-model:open="createOpen" mode="create" @saved="created" />
  <OrderDialog
    :children="overview?.roots ?? []"
    :open="orderOpen"
    :parent-id="null"
    parent-title=""
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
.recent-comments {
  padding: 16px;
  font-size: 13px;
}
.recent-comment:first-child {
  padding-top: 0;
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
