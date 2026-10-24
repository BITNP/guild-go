<script setup lang="ts">
import { ref, watch } from 'vue'
import ConfirmAction from '@/components/ConfirmAction.vue'
import { deleteNode, type NodeDetail } from '@/lib/api'
import { errorMessage } from '@/lib/utils'
const props = defineProps<{ open: boolean; node: NodeDetail }>(),
  emit = defineEmits<{ close: []; deleted: [] }>()
const deleting = ref(false),
  error = ref<string | null>(null)
watch(
  () => props.open,
  () => {
    error.value = null
  },
)
async function confirm() {
  if (deleting.value) return
  deleting.value = true
  error.value = null
  try {
    await deleteNode(props.node.id)
    emit('deleted')
    emit('close')
  } catch (err) {
    error.value = errorMessage(err, '删除节点失败')
  } finally {
    deleting.value = false
  }
}
</script>
<template>
  <ConfirmAction
    :open="open"
    title="删除节点"
    confirm-text="确认删除"
    :busy="deleting"
    :error="error"
    @confirm="confirm"
    @close="emit('close')"
    ><p>将删除「{{ node.title }}」及其所有下级节点。</p>
    <p>
      此操作不可撤销，包含 {{ node.subtreeNodeCount }} 个节点、{{
        node.subtreeCommentCount
      }}
      条评论和 {{ node.subtreeAttachmentCount }} 张图片。
    </p></ConfirmAction
  >
</template>
