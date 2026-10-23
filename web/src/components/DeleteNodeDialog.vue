<script setup lang="ts">
import { ref, watch } from 'vue'
import { AlertTriangle, Loader2 } from '@lucide/vue'
import BaseModal from '@/components/BaseModal.vue'
import { deleteNode, type NodeDetail } from '@/lib/api'
import { errorMessage } from '@/lib/utils'

const props = defineProps<{
  open: boolean
  node: NodeDetail
}>()

const emit = defineEmits<{
  close: []
  deleted: []
}>()

const deleting = ref(false)
const error = ref<string | null>(null)

watch(() => props.open, () => { error.value = null })

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
  <BaseModal :open="open" title="删除节点" :busy="deleting" @close="emit('close')">
    <div class="flex flex-col gap-3">
      <div class="flex gap-2 rounded-md border border-danger/20 bg-danger-soft px-3 py-2.5 text-danger">
        <AlertTriangle class="mt-0.5 h-4 w-4 shrink-0" />
        <div class="text-[12px] leading-5">
          <p class="font-semibold">将删除「{{ node.title }}」及其所有下级节点</p>
          <p>此操作不可撤销，包含 {{ node.subtreeNodeCount }} 个节点、{{ node.subtreeCommentCount }} 条评论、{{ node.subtreeAttachmentCount }} 张图片。</p>
        </div>
      </div>
      <p v-if="error" role="alert" class="text-xs text-danger">{{ error }}</p>
    </div>

    <template #footer>
      <button type="button" class="btn" :disabled="deleting" @click="emit('close')">取消</button>
      <button type="button" class="btn btn-danger" :disabled="deleting" @click="confirm">
        <Loader2 v-if="deleting" class="h-3.5 w-3.5 animate-spin" />
        {{ deleting ? '删除中…' : '确认删除' }}
      </button>
    </template>
  </BaseModal>
</template>
