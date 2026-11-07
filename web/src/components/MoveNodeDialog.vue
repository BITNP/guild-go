<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { NAlert, NButton, NEmpty, NFormItem, NSelect, NSpace, NText } from 'naive-ui'
import AppModal from '@/components/AppModal.vue'
import { useAuth } from '@/composables/useAuth'
import { fetchNodes, moveNode, type NodeDetail, type NodeSummary } from '@/lib/api'
import { errorMessage } from '@/lib/utils'

const props = defineProps<{ open: boolean; node: NodeDetail }>()
const emit = defineEmits<{ close: []; saved: [node: NodeDetail] }>()
const { isAdmin, isManager } = useAuth()
const nodes = ref<NodeSummary[]>([])
const target = ref<number | null>(null)
const loading = ref(false)
const saving = ref(false)
const error = ref<string | null>(null)
const retry = ref(0)

watch(
  [() => props.open, () => props.node.id, retry],
  async ([open], _, onCleanup) => {
    if (!open) return
    let active = true
    onCleanup(() => { active = false })
    target.value = null
    nodes.value = []
    error.value = null
    loading.value = true
    try {
      const result = await fetchNodes()
      if (active) nodes.value = result
    } catch (err) {
      if (active) error.value = errorMessage(err, '无法加载目标节点')
    } finally {
      if (active) loading.value = false
    }
  },
  { immediate: true },
)

const options = computed(() => {
  const byId = new Map(nodes.value.map(node => [node.id, node]))
  const children = new Map<number, number[]>()
  for (const node of nodes.value) {
    if (node.parentId == null) continue
    const siblings = children.get(node.parentId) ?? []
    siblings.push(node.id)
    children.set(node.parentId, siblings)
  }
  const excluded = new Set<number>()
  const pending = [props.node.id]
  while (pending.length) {
    const id = pending.pop()!
    if (excluded.has(id)) continue
    excluded.add(id)
    pending.push(...(children.get(id) ?? []))
  }
  function path(node: NodeSummary) {
    const titles: string[] = []
    const seen = new Set<number>()
    let current: NodeSummary | undefined = node
    while (current && !seen.has(current.id)) {
      seen.add(current.id)
      titles.unshift(current.title)
      current = current.parentId == null ? undefined : byId.get(current.parentId)
    }
    return titles.join(' / ')
  }
  const result = nodes.value
    .filter(node => !excluded.has(node.id) && node.id !== props.node.parentId &&
      (isAdmin.value || node.myRelation === 'leader'))
    .map(node => ({ label: path(node), value: node.id }))
    .sort((a, b) => a.label.localeCompare(b.label, 'zh-CN') || a.value - b.value)
  if ((isAdmin.value || isManager.value) && props.node.parentId != null) {
    result.unshift({ label: '总览（移动为根节点）', value: 0 })
  }
  return result
})

async function save() {
  if (saving.value || loading.value || target.value == null) return
  saving.value = true
  error.value = null
  try {
    const result = await moveNode(props.node.id, target.value === 0 ? null : target.value)
    emit('saved', result)
    emit('close')
  } catch (err) {
    error.value = errorMessage(err, '移动节点失败')
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <AppModal :open="open" title="移动节点" :busy="saving" @close="emit('close')">
    <NSpace vertical :size="16">
      <NText>当前节点：{{ node.title }}</NText>
      <NText depth="3">当前位置：{{ node.breadcrumb.map(item => item.title).join(' / ') || '总览' }}</NText>
      <NFormItem label="移动到" :show-feedback="false">
        <NSelect
          v-model:value="target"
          aria-label="目标父节点"
          :options="options"
          :loading="loading"
          :disabled="loading || saving"
          filterable
          placeholder="搜索并选择目标父节点"
        />
      </NFormItem>
      <NText depth="3">节点及所有下级节点将一同移动到目标列表末尾。成员和内容保留，继承权限将按新位置计算。</NText>
      <NEmpty v-if="!loading && !error && !options.length" description="暂无可移动的位置" size="small" />
      <NAlert v-if="error" type="error" role="alert">
        {{ error }} <NButton v-if="!nodes.length" text @click="retry++">重新加载</NButton>
      </NAlert>
      <NSpace justify="end">
        <NButton :disabled="saving" @click="emit('close')">取消</NButton>
        <NButton type="primary" :loading="saving" :disabled="loading || target == null" @click="save">确认移动</NButton>
      </NSpace>
    </NSpace>
  </AppModal>
</template>
