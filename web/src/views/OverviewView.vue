<script setup lang="ts">
import {
  NAlert,
  NButton,
  NCard,
  NEmpty,
  NGi,
  NGrid,
  NH1,
  NList,
  NSpace,
  NSpin,
  NStatistic,
} from "naive-ui";
import { computed, onMounted, ref, watch } from "vue";
import { useRouter } from "vue-router";

import AppShell from "@/components/AppShell.vue";
import NodeFormDialog from "@/components/NodeFormDialog.vue";
import NodeListItem from "@/components/NodeListItem.vue";
import OrderDialog from "@/components/OrderDialog.vue";
import RecentProgress from "@/components/RecentProgress.vue";
import SectionHeading from "@/components/SectionHeading.vue";
import { useAuth } from "@/composables/useAuth";
import { useNodeStore } from "@/composables/useNodeStore";
import type { ActivityScope, NodeDetail, Overview } from "@/lib/api";
import { fetchOverview } from "@/lib/api";
import { errorMessage, isOverdue } from "@/lib/utils";

const router = useRouter();
const { isManager } = useAuth();
const {
  nodes,
  loading: nodesLoading,
  error: nodesError,
  reload: reloadNodes,
} = useNodeStore();
const overview = ref<Overview | null>(null);
const loading = ref(true);
const error = ref<string | null>(null);
const createOpen = ref(false);
const orderOpen = ref(false);
const activityScope = ref<ActivityScope>("related");
const rootIds = computed(
  () => new Set(overview.value?.roots.map((root) => root.id)),
);
let loadVersion = 0;
const statisticsLoading = computed(
  () => nodesLoading.value && nodes.value.length === 0,
);
const statistics = computed(() => {
  if (statisticsLoading.value) {
    return [
      { label: "全部节点", value: "—", warning: false },
      { label: "我负责", value: "—", warning: false },
      { label: "未完成", value: "—", warning: false },
      { label: "已逾期", value: "—", warning: false },
    ];
  }
  const overdueCount = nodes.value.filter((node) =>
    isOverdue(node.dueAt, node.completedAt),
  ).length;

  return [
    { label: "全部节点", value: nodes.value.length, warning: false },
    {
      label: "我负责",
      value: nodes.value.filter((node) => node.myRelation === "leader").length,
      warning: false,
    },
    {
      label: "未完成",
      value: nodes.value.filter((node) => !node.completedAt).length,
      warning: false,
    },
    {
      label: "已逾期",
      value: overdueCount,
      warning: overdueCount > 0,
    },
  ];
});

async function load() {
  const version = ++loadVersion;
  loading.value = true;
  error.value = null;
  try {
    const result = await fetchOverview(activityScope.value);
    if (version === loadVersion) {
      overview.value = result;
    }
  } catch (err) {
    if (version === loadVersion) {
      error.value = errorMessage(err, "无法加载最近进展");
    }
  } finally {
    if (version === loadVersion) {
      loading.value = false;
    }
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
watch(activityScope, load);
</script>

<template>
  <AppShell>
    <section class="overview-page">
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
        <NAlert v-if="nodesError" type="error">
          {{ nodesError }}
          <NButton text @click="reloadNodes">重试</NButton>
        </NAlert>
        <NSpin v-else :show="statisticsLoading" size="small">
          <NCard size="small">
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
                    color: stat.warning ? 'var(--guild-warning)' : undefined,
                  }"
                />
              </NGi>
            </NGrid>
          </NCard>
        </NSpin>
        <RecentProgress
          v-model:scope="activityScope"
          :activities="overview?.recentActivities ?? []"
          :error
          :loading
          :root-ids
          @retry="load"
        />
        <section
          v-if="overview || (loading && !error)"
          aria-label="根节点"
          class="root-section"
        >
          <SectionHeading :count="overview?.roots.length" title="根节点" />
          <NCard :aria-busy="loading && !overview" size="small">
            <div
              v-if="loading && !overview"
              aria-label="正在加载根节点"
              class="roots-placeholder"
              role="status"
            >
              <NSpin size="small" />
            </div>
            <NList v-else-if="overview?.roots.length">
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
      </NSpace>
    </section>
  </AppShell>
  <NodeFormDialog v-model:open="createOpen" mode="create" @saved="created" />
  <OrderDialog
    v-model:open="orderOpen"
    :children="overview?.roots ?? []"
    :parent-id="null"
    parent-title=""
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
.roots-placeholder {
  padding: 20px 0;
  text-align: center;
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
