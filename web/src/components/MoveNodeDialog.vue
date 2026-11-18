<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { NAlert, NButton, NFormItem, NSelect, NSpace, NText } from 'naive-ui'
import AppModal from '@/components/AppModal.vue'
import { useAuth } from '@/composables/useAuth'
import { useNodeStore } from '@/composables/useNodeStore'
import { moveNode, type NodeDetail } from '@/lib/api'
import { errorMessage } from '@/lib/utils'

const open = defineModel<boolean>('open', { required: true })
const props = defineProps<{ node: NodeDetail }>()
const emit = defineEmits<{ saved: [node: NodeDetail] }>()
const { isAdmin, isManager } = useAuth()
const { nodes, loading, error: loadError, reload, subtreeIds, ancestorPath } =
  useNodeStore()
const target = ref<number | null>(null)
const saving = ref(false)
const submitError = ref<string | null>(null)
const displayError = computed(() => submitError.value ?? loadError.value)

watch(
  [open, () => props.node.id],
  ([isOpen]) => {
    if (!isOpen) return
    target.value = null
    submitError.value = null
    void reload()
  },
  { immediate: true },
)

const options = computed(() => {
  const excluded = subtreeIds(props.node.id)
  const result = nodes.value
    .filter(
      (node) =>
        !excluded.has(node.id) &&
        node.id !== props.node.parentId &&
        (isAdmin.value || node.myRelation === 'leader'),
    )
    .map((node) => ({ label: ancestorPath(node.id), value: node.id }))
    .sort((a, b) => a.label.localeCompare(b.label, 'zh-CN') || a.value - b.value)
  if ((isAdmin.value || isManager.value) && props.node.parentId != null) {
    result.unshift({ label: '总览（移动为根节点）', value: 0 })
  }
  return result
})

async function save() {
  if (saving.value || loading.value || target.value == null) return
  saving.value = true
  submitError.value = null
  try {
    const result = await moveNode(props.node.id, target.value === 0 ? null : target.value)
    emit('saved', result)
    open.value = false
  } catch (err) {
    submitError.value = errorMessage(err, '移动节点失败')
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <AppModal :open="open" title="移动节点" :busy="saving" @close="open = false">
    <NSpace vertical :size="16">
      <NText>当前节点：{{ node.title }}</NText>
      <NText depth="3">当前位置：{{ node.breadcrumb.map(item => item.title).join(' / ') || '总览' }}</NText>
      <NFormItem label="移动到" :show-feedback="false">
        <NSelect
          v-model:value="target"
          aria-label="目标父节点"
          :options="options"
          :loading="loading"
          :disabled="loading || saving || (!displayError && !options.length)"
          filterable
          :placeholder="!loading && !displayError && !options.length ? '暂无可移动的位置' : '搜索并选择目标父节点'"
        />
      </NFormItem>
      <NText depth="3">节点及所有下级节点将一同移动到目标列表末尾。成员和内容保留，继承权限将按新位置计算。</NText>
      <NAlert v-if="displayError" type="error" role="alert">
        {{ displayError }} <NButton v-if="loadError && !nodes.length" text @click="reload">重新加载</NButton>
      </NAlert>
      <NSpace justify="end">
        <NButton :disabled="saving" @click="open = false">取消</NButton>
        <NButton type="primary" :loading="saving" :disabled="loading || target == null" @click="save">确认移动</NButton>
      </NSpace>
    </NSpace>
  </AppModal>
</template>
