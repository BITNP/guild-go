<script setup lang="ts">
import type { TreeOption } from "naive-ui";

const emit = defineEmits<{ navigate: [] }>();
const route = useRoute();
const router = useRouter();
const { isManager } = useAuth();
const { nodes, roots, childrenOf, byId, loading, error, reload } =
  useNodeStore();
const search = ref("");
const relation = ref("all");
const tag = ref<string | null>(null);
const expanded = ref<number[]>([]);
const initialized = ref(false);
const createOpen = ref(false);
const filtering = computed(
  () => !!(search.value.trim() || relation.value !== "all" || tag.value),
);
const tags = computed(() =>
  [...new Set(nodes.value.flatMap((node) => node.tags))]
    .sort((a, b) => a.localeCompare(b, "zh-CN"))
    .map((value) => ({ label: value, value })),
);
const visible = computed(() => {
  if (!filtering.value) {
    return null;
  }
  const result = new Set<number>();
  const query = search.value.trim().toLowerCase();
  for (const node of nodes.value) {
    if (relation.value === "leader" && node.myRelation !== "leader") {
      continue;
    }
    if (relation.value === "member" && !node.myRelation) {
      continue;
    }
    if (tag.value && !node.tags.includes(tag.value)) {
      continue;
    }
    if (
      query &&
      !node.title.toLowerCase().includes(query) &&
      !node.tags.some((value) => value.toLowerCase().includes(query))
    ) {
      continue;
    }
    let current: NodeSummary | undefined = node;
    while (current && !result.has(current.id)) {
      result.add(current.id);
      current =
        current.parentId == null ? undefined : byId.value.get(current.parentId);
    }
  }

  return result;
});

function treeOption(node: NodeSummary): TreeOption[] {
  if (visible.value && !visible.value.has(node.id)) {
    return [];
  }
  const children = childrenOf(node.id).flatMap(treeOption);

  return [
    {
      key: node.id,
      label: node.title,
      isRoot: node.parentId == null,
      children: children.length > 0 ? children : undefined,
      isLeaf: children.length === 0,
      prefix: () =>
        h(node.completedAt ? Check : Circle, {
          "size": 14,
          "color": node.completedAt
            ? "var(--guild-success)"
            : isOverdue(node.dueAt, node.completedAt)
              ? "var(--guild-warning)"
              : "var(--guild-muted)",
          "aria-label": node.completedAt ? "已完成" : "未完成",
        }),
      suffix: node.myRelation
        ? () =>
            h(NText, { class: "tree-metadata tree-relation" }, () =>
              node.myRelation === "leader" ? "负责" : "参与",
            )
        : undefined,
    },
  ];
}

const data = computed(() => roots.value.flatMap(treeOption));
const selected = computed(() =>
  route.name === "node" ? [Number(route.params.id)] : [],
);
const expandedKeys = computed(() =>
  filtering.value ? nodes.value.map((node) => node.id) : expanded.value,
);
watch(
  nodes,
  (list) => {
    if (!initialized.value && list.length > 0) {
      expanded.value = list
        .filter((node) => node.childCount)
        .map((node) => node.id);
      initialized.value = true;
    }
  },
  { immediate: true },
);

function navigate(keys: (string | number)[]) {
  if (keys.length === 0) {
    return;
  }
  void router.push({ name: "node", params: { id: keys[0] } });
  emit("navigate");
}

async function created(node: NodeDetail) {
  createOpen.value = false;
  await reload();
  navigate([node.id]);
}
</script>

<template>
  <section class="node-navigation">
    <div class="node-filters">
      <NInputGroup>
        <NInput
          v-model:value="search"
          clearable
          :input-props="{ 'aria-label': '搜索节点或标签' }"
          placeholder="搜索节点或标签"
        />
        <NButton
          v-if="isManager"
          aria-label="新建根节点"
          @click="createOpen = true"
        >
          <Plus :size="16" />
        </NButton>
      </NInputGroup>
      <NRadioGroup
        v-model:value="relation"
        aria-label="按参与关系筛选"
        class="relation-filter"
        size="small"
      >
        <NRadioButton class="relation-option" value="all">全部</NRadioButton>
        <NRadioButton class="relation-option" value="leader">
          我负责
        </NRadioButton>
        <NRadioButton class="relation-option" value="member">
          我参与
        </NRadioButton>
      </NRadioGroup>
      <div class="tree-controls">
        <NSelect
          v-model:value="tag"
          aria-label="按标签筛选"
          class="tag-filter"
          clearable
          :options="tags"
          placeholder="全部标签"
          size="small"
        />
        <NTooltip>
          <template #trigger>
            <NButton
              aria-label="展开全部"
              class="tree-action"
              :disabled="filtering"
              size="small"
              @click="
                expanded = nodes
                  .filter((node) => node.childCount)
                  .map((node) => node.id)
              "
            >
              <ChevronsUpDown aria-hidden="true" :size="16" />
            </NButton>
          </template>
          展开全部
        </NTooltip>
        <NTooltip>
          <template #trigger>
            <NButton
              aria-label="收起全部"
              class="tree-action"
              :disabled="filtering"
              size="small"
              @click="expanded = []"
            >
              <ChevronsDownUp aria-hidden="true" :size="16" />
            </NButton>
          </template>
          收起全部
        </NTooltip>
      </div>
    </div>
    <NScrollbar class="tree-scroll">
      <div class="tree-content">
        <NSpin :delay="200" :show="loading && nodes.length === 0">
          <NAlert v-if="error" type="error">
            {{ error }}
            <NButton text @click="reload">重试</NButton>
          </NAlert>
          <NTree
            v-else-if="data.length > 0"
            block-line
            :data
            ellipsis
            :expanded-keys
            :indent="18"
            :render-label="
              ({ option }) =>
                h(
                  NEllipsis,
                  {
                    tooltip: true,
                    lineClamp: 2,
                    class: 'tree-label',
                    style: { fontWeight: option.isRoot ? 600 : undefined },
                  },
                  () => String(option.label),
                )
            "
            :selected-keys="selected"
            show-line
            @update:expanded-keys="(keys) => (expanded = keys.map(Number))"
            @update:selected-keys="navigate"
          />
          <NEmpty
            v-else
            :description="filtering ? '没有匹配的节点' : '还没有节点'"
            size="small"
          />
        </NSpin>
      </div>
    </NScrollbar>
    <NText class="node-count" depth="3">
      {{ nodes.length }} 个节点 · {{ roots.length }} 个根节点
    </NText>
    <NodeFormDialog v-model:open="createOpen" mode="create" @saved="created" />
  </section>
</template>

<style scoped>
.node-navigation {
  height: 100%;
  min-height: 320px;
  display: flex;
  flex-direction: column;
}
.node-filters {
  padding: 12px;
  display: flex;
  flex-direction: column;
  gap: 8px;
  border-bottom: 1px solid var(--guild-border);
  background: var(--guild-surface-subtle);
}
.tree-scroll {
  flex: 1;
  min-height: 0;
}
.relation-filter {
  display: flex;
  width: 100%;
}
.relation-option {
  flex: 1;
  min-width: 0;
  padding: 0 8px;
  text-align: center;
}
.tree-controls {
  display: flex;
  align-items: center;
  gap: 8px;
}
.tag-filter {
  flex: 1;
  min-width: 0;
}
.tree-action {
  width: 28px;
  padding: 0;
  flex: none;
}
.tree-content {
  padding: 8px;
}
.node-count {
  padding: 12px;
  font-size: 12px;
  border-top: 1px solid var(--guild-border);
}
.node-navigation :deep(.tree-metadata) {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding-left: 8px;
  font-size: 11px;
  line-height: 16px;
  white-space: nowrap;
}
.node-navigation :deep(.tree-label) {
  display: -webkit-box;
  white-space: normal;
  overflow-wrap: anywhere;
}
.node-navigation :deep(.tree-relation) {
  color: var(--guild-primary);
}
</style>
