<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  Check,
  ChevronDown,
  ChevronRight,
  ChevronsDownUp,
  ChevronsUpDown,
  Clock,
  MessageSquare,
  Plus,
  Search,
} from '@lucide/vue'
import NodeFormDialog from '@/components/NodeFormDialog.vue'
import { useAuth } from '@/composables/useAuth'
import { useNodeStore } from '@/composables/useNodeStore'
import type { NodeDetail, NodeSummary } from '@/lib/api'
import { isOverdue } from '@/lib/utils'

const emit = defineEmits<{
  navigate: []
}>()

const route = useRoute()
const router = useRouter()
const { isManager } = useAuth()
const { nodes, loading, error, roots, childrenOf, byId, reload } = useNodeStore()

const search = ref('')
const relation = ref<'all' | 'leader' | 'member'>('all')
const relationOptions: Array<{ value: 'all' | 'leader' | 'member'; label: string }> = [
  { value: 'all', label: '全部' },
  { value: 'leader', label: '我负责' },
  { value: 'member', label: '我参与' },
]
const tag = ref('')
const expanded = ref<Set<number>>(new Set())
const createOpen = ref(false)
const initialized = ref(false)

const allTags = computed(() => {
  const values = new Set<string>()
  for (const node of nodes.value) {
    for (const value of node.tags) values.add(value)
  }
  return [...values].sort((a, b) => a.localeCompare(b, 'zh-CN'))
})

const filterActive = computed(() => search.value.trim() !== '' || relation.value !== 'all' || tag.value !== '')

const visibleIDs = computed(() => {
  if (!filterActive.value) return null
  const query = search.value.trim().toLowerCase()
  const matched = new Set<number>()
  for (const node of nodes.value) {
    if (relation.value === 'leader' && node.myRelation !== 'leader') continue
    if (relation.value === 'member' && !node.myRelation) continue
    if (tag.value && !node.tags.includes(tag.value)) continue
    if (query && !node.title.toLowerCase().includes(query)
      && !node.tags.some((value) => value.toLowerCase().includes(query))) continue
    matched.add(node.id)
  }
  const result = new Set<number>()
  for (const id of matched) {
    let current = byId.value.get(id)
    const guard = new Set<number>()
    while (current && !guard.has(current.id)) {
      guard.add(current.id)
      result.add(current.id)
      current = current.parentId == null ? undefined : byId.value.get(current.parentId)
    }
  }
  return result
})

interface Row {
  node: NodeSummary
  depth: number
}

const rows = computed<Row[]>(() => {
  const result: Row[] = []
  const walk = (node: NodeSummary, depth: number) => {
    if (visibleIDs.value && !visibleIDs.value.has(node.id)) return
    result.push({ node, depth })
    const open = filterActive.value || expanded.value.has(node.id)
    if (!open) return
    for (const child of childrenOf(node.id)) walk(child, depth + 1)
  }
  for (const root of roots.value) walk(root, 0)
  return result
})

const activeId = computed(() => Number(route.params.id) || null)

watch(() => nodes.value.length, (length) => {
  if (initialized.value || length === 0) return
  const next = new Set<number>()
  for (const node of nodes.value) {
    if (node.childCount > 0) next.add(node.id)
  }
  expanded.value = next
  initialized.value = true
}, { immediate: true })

function setRelation(value: 'all' | 'leader' | 'member') {
  relation.value = value
}

function onChevron(row: Row) {
  if (row.node.childCount > 0) toggle(row.node.id)
}

function toggle(id: number) {
  const next = new Set(expanded.value)
  if (next.has(id)) next.delete(id)
  else next.add(id)
  expanded.value = next
}

function expandAll() {
  expanded.value = new Set(nodes.value.filter((node) => node.childCount > 0).map((node) => node.id))
}

function collapseAll() {
  expanded.value = new Set()
}

function openNode(id: number) {
  router.push({ name: 'node', params: { id } })
  emit('navigate')
}

async function onCreated(node: NodeDetail) {
  createOpen.value = false
  await reload()
  router.push({ name: 'node', params: { id: node.id } })
  emit('navigate')
}

function relationLabel(node: NodeSummary): string {
  if (node.myRelation === 'leader') return '负责'
  if (node.myRelation === 'member') return '参与'
  return ''
}
</script>

<template>
  <div class="flex h-full min-h-0 flex-col">
    <div class="flex flex-col gap-2 border-b border-line p-2.5">
      <div class="flex items-center gap-1.5">
        <label class="relative min-w-0 flex-1">
          <Search class="pointer-events-none absolute left-2 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-faint" />
          <input v-model="search" class="input h-7 pl-7 text-xs" placeholder="搜索节点或标签">
        </label>
        <button
          v-if="isManager"
          type="button"
          class="icon-btn border border-line bg-surface"
          title="新建根节点"
          aria-label="新建根节点"
          @click="createOpen = true"
        >
          <Plus class="h-4 w-4" />
        </button>
      </div>

      <div class="flex items-center gap-1">
        <div class="flex flex-1 rounded-md border border-line bg-surface-2 p-0.5 text-[11px]">
          <button
            v-for="item in relationOptions"
            :key="item.value"
            type="button"
            class="flex-1 rounded px-1.5 py-1 transition-colors"
            :class="relation === item.value ? 'bg-surface text-ink shadow-sm' : 'text-muted hover:text-ink'"
            @click="setRelation(item.value)"
          >
            {{ item.label }}
          </button>
        </div>
        <button class="icon-btn" title="展开全部" aria-label="展开全部" @click="expandAll">
          <ChevronsUpDown class="h-3.5 w-3.5" />
        </button>
        <button class="icon-btn" title="收起全部" aria-label="收起全部" @click="collapseAll">
          <ChevronsDownUp class="h-3.5 w-3.5" />
        </button>
      </div>

      <select v-model="tag" class="select h-7 w-full text-xs" aria-label="按标签筛选">
        <option value="">全部标签</option>
        <option v-for="item in allTags" :key="item" :value="item"># {{ item }}</option>
      </select>
    </div>

    <div class="min-h-0 flex-1 overflow-y-auto px-1.5 py-1.5">
      <p v-if="loading && nodes.length === 0" class="px-2 py-3 text-xs text-muted">正在加载节点树…</p>
      <p v-else-if="error" class="px-2 py-3 text-xs text-danger">{{ error }}</p>
      <p v-else-if="rows.length === 0" class="px-2 py-6 text-center text-xs text-muted">
        {{ filterActive ? '没有匹配的节点' : '还没有节点' }}
      </p>

      <button
        v-for="row in rows"
        :key="row.node.id"
        type="button"
        class="group flex w-full items-center gap-1 rounded-md py-1.5 pr-1.5 text-left transition-colors"
        :class="activeId === row.node.id ? 'bg-accent-soft text-accent' : 'text-ink-soft hover:bg-surface-2'"
        :style="{ paddingLeft: `${6 + row.depth * 13}px` }"
        @click="openNode(row.node.id)"
      >
        <span
          class="flex h-4 w-4 shrink-0 items-center justify-center rounded text-faint hover:bg-surface-3 hover:text-ink"
          @click.stop="onChevron(row)"
        >
          <ChevronDown v-if="row.node.childCount > 0 && (filterActive || expanded.has(row.node.id))" class="h-3.5 w-3.5" />
          <ChevronRight v-else-if="row.node.childCount > 0" class="h-3.5 w-3.5" />
          <span v-else class="h-1 w-1 rounded-full bg-line-strong" />
        </span>

        <span
          v-if="row.node.completedAt"
          class="flex h-3.5 w-3.5 shrink-0 items-center justify-center rounded-full bg-ok-soft text-ok"
          title="已完成"
        >
          <Check class="h-2.5 w-2.5" />
        </span>
        <span
          v-else
          class="status-dot shrink-0"
          :class="isOverdue(row.node.dueAt, row.node.completedAt) ? 'bg-warn' : 'bg-line-strong'"
        />

        <span class="min-w-0 flex-1 truncate text-[12.5px]" :class="row.node.completedAt ? 'text-muted' : ''">
          {{ row.node.title }}
        </span>

        <span
          v-if="relationLabel(row.node)"
          class="shrink-0 rounded px-1 text-[9px] font-medium"
          :class="row.node.myRelation === 'leader' ? 'bg-accent/10 text-accent' : 'bg-surface-3 text-muted'"
        >
          {{ relationLabel(row.node) }}
        </span>

        <span v-if="isOverdue(row.node.dueAt, row.node.completedAt)" class="shrink-0 text-warn" title="已逾期">
          <Clock class="h-3 w-3" />
        </span>

        <span v-if="row.node.commentCount > 0" class="flex shrink-0 items-center gap-0.5 text-[10px] text-faint">
          <MessageSquare class="h-2.5 w-2.5" />{{ row.node.commentCount }}
        </span>
      </button>
    </div>

    <div class="flex items-center justify-between border-t border-line px-3 py-1.5 text-[10px] text-faint">
      <span>{{ nodes.length }} 个节点</span>
      <span>{{ roots.length }} 棵树</span>
    </div>

    <NodeFormDialog
      v-model:open="createOpen"
      mode="create"
      :parent-id="null"
      @saved="onCreated"
    />
  </div>
</template>