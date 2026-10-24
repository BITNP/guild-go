<script setup lang="ts">
import { computed, h, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  NAlert,
  NButton,
  NEmpty,
  NEllipsis,
  NInput,
  NInputGroup,
  NRadioButton,
  NRadioGroup,
  NScrollbar,
  NSelect,
  NSpin,
  NText,
  NTooltip,
  NTree,
  type TreeOption,
} from 'naive-ui'
import {
  Check,
  ChevronsDownUp,
  ChevronsUpDown,
  Circle,
  MessageSquare,
  Plus,
} from '@lucide/vue'
import { useAuth } from '@/composables/useAuth'
import { useNodeStore } from '@/composables/useNodeStore'
import NodeFormDialog from '@/components/NodeFormDialog.vue'
import type { NodeDetail, NodeSummary } from '@/lib/api'
import { isOverdue } from '@/lib/utils'
const emit = defineEmits<{ navigate: [] }>()
const route = useRoute(),
  router = useRouter(),
  { isManager } = useAuth()
const { nodes, roots, childrenOf, byId, loading, error, reload } =
  useNodeStore()
const search = ref(''),
  relation = ref('all'),
  tag = ref<string | null>(null),
  expanded = ref<number[]>([]),
  initialized = ref(false),
  createOpen = ref(false)
const filtering = computed(
  () => !!(search.value.trim() || relation.value !== 'all' || tag.value),
)
const tags = computed(() =>
  [...new Set(nodes.value.flatMap((node) => node.tags))]
    .sort((a, b) => a.localeCompare(b, 'zh-CN'))
    .map((value) => ({ label: value, value })),
)
const visible = computed(() => {
  if (!filtering.value) return null
  const result = new Set<number>(),
    query = search.value.trim().toLowerCase()
  for (const node of nodes.value) {
    if (relation.value === 'leader' && node.myRelation !== 'leader') continue
    if (relation.value === 'member' && !node.myRelation) continue
    if (tag.value && !node.tags.includes(tag.value)) continue
    if (
      query &&
      !node.title.toLowerCase().includes(query) &&
      !node.tags.some((value) => value.toLowerCase().includes(query))
    )
      continue
    let current: NodeSummary | undefined = node
    while (current && !result.has(current.id)) {
      result.add(current.id)
      current =
        current.parentId == null ? undefined : byId.value.get(current.parentId)
    }
  }
  return result
})
function treeOption(node: NodeSummary): TreeOption[] {
  if (visible.value && !visible.value.has(node.id)) return []
  const children = childrenOf(node.id).flatMap(treeOption)
  return [
    {
      key: node.id,
      label: node.title,
      isRoot: node.parentId == null,
      children: children.length ? children : undefined,
      isLeaf: !children.length,
      prefix: () =>
        h(node.completedAt ? Check : Circle, {
          size: 14,
          color: node.completedAt
            ? '#157347'
            : isOverdue(node.dueAt, node.completedAt)
              ? '#966200'
              : '#687587',
          'aria-label': node.completedAt ? '已完成' : '未完成',
        }),
      suffix:
        node.myRelation || node.commentCount
          ? () =>
              h(
                'span',
                {
                  class: 'tree-metadata',
                },
                [
                  node.commentCount
                    ? h(
                        'span',
                        {
                          class: 'tree-comment-count',
                          title: `${node.commentCount} 条评论`,
                        },
                        [
                          h(MessageSquare, { size: 12, 'aria-hidden': true }),
                          String(node.commentCount),
                        ],
                      )
                    : null,
                  node.myRelation
                    ? h(
                        NText,
                        {
                          class: 'tree-relation',
                        },
                        () => (node.myRelation === 'leader' ? '负责' : '参与'),
                      )
                    : null,
                ],
              )
          : undefined,
    },
  ]
}
const data = computed(() => roots.value.flatMap(treeOption))
const selected = computed(() =>
  route.name === 'node' ? [Number(route.params.id)] : [],
)
const expandedKeys = computed(() =>
  filtering.value ? nodes.value.map((node) => node.id) : expanded.value,
)
watch(
  nodes,
  (list) => {
    if (!initialized.value && list.length) {
      expanded.value = list
        .filter((node) => node.childCount)
        .map((node) => node.id)
      initialized.value = true
    }
  },
  { immediate: true },
)
function navigate(keys: Array<string | number>) {
  if (!keys.length) return
  void router.push({ name: 'node', params: { id: keys[0] } })
  emit('navigate')
}
async function created(node: NodeDetail) {
  createOpen.value = false
  await reload()
  navigate([node.id])
}
</script>
<template>
  <section class="node-navigation">
    <div class="node-filters">
      <NInputGroup
        ><NInput
          v-model:value="search"
          clearable
          placeholder="搜索节点或标签"
          :input-props="{ 'aria-label': '搜索节点或标签' }" /><NButton
          v-if="isManager"
          aria-label="新建根节点"
          @click="createOpen = true"
          ><Plus :size="16" /></NButton></NInputGroup
      ><NRadioGroup
        v-model:value="relation"
        size="small"
        aria-label="按参与关系筛选"
        class="relation-filter"
        ><NRadioButton class="relation-option" value="all">全部</NRadioButton
        ><NRadioButton class="relation-option" value="leader"
          >我负责</NRadioButton
        ><NRadioButton class="relation-option" value="member"
          >我参与</NRadioButton
        ></NRadioGroup
      >
      <div class="tree-controls">
        <NSelect
          v-model:value="tag"
          size="small"
          clearable
          :options="tags"
          placeholder="全部标签"
          aria-label="按标签筛选"
          class="tag-filter"
        /><NTooltip
          ><template #trigger
            ><NButton
              size="small"
              aria-label="展开全部"
              class="tree-action"
              :disabled="filtering"
              @click="
                expanded = nodes
                  .filter((node) => node.childCount)
                  .map((node) => node.id)
              "
              ><ChevronsUpDown
                :size="16"
                aria-hidden="true" /></NButton></template
          >展开全部</NTooltip
        ><NTooltip
          ><template #trigger
            ><NButton
              size="small"
              aria-label="收起全部"
              class="tree-action"
              :disabled="filtering"
              @click="expanded = []"
              ><ChevronsDownUp
                :size="16"
                aria-hidden="true" /></NButton></template
          >收起全部</NTooltip
        >
      </div>
    </div>
    <NScrollbar class="tree-scroll"
      ><div class="tree-content">
        <NSpin :show="loading && !nodes.length" :delay="200"
          ><NAlert v-if="error" type="error"
            >{{ error }} <NButton text @click="reload">重试</NButton></NAlert
          ><NTree
            v-else-if="data.length"
            block-line
            show-line
            ellipsis
            :indent="18"
            :data="data"
            :selected-keys="selected"
            :expanded-keys="expandedKeys"
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
            @update:selected-keys="navigate"
            @update:expanded-keys="
              (keys) => (expanded = keys.map(Number))
            " /><NEmpty
            v-else
            size="small"
            :description="filtering ? '没有匹配的节点' : '还没有节点'"
        /></NSpin></div
    ></NScrollbar>
    <NText depth="3" class="node-count"
      >{{ nodes.length }} 个节点 · {{ roots.length }} 个根节点</NText
    ><NodeFormDialog v-model:open="createOpen" mode="create" @saved="created" />
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
.node-navigation :deep(.tree-comment-count) {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  color: var(--guild-muted);
}
</style>
