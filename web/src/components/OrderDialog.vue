<script setup lang="ts">
import { ref, watch } from 'vue'
import { ArrowDown, ArrowUp, Loader2 } from '@lucide/vue'
import BaseModal from '@/components/BaseModal.vue'
import { orderNodes, type ChildNode } from '@/lib/api'
import { errorMessage } from '@/lib/utils'

const props = defineProps<{
  open: boolean
  parentId: number | null
  parentTitle: string
  children: ChildNode[]
}>()

const emit = defineEmits<{
  close: []
  saved: []
}>()

const items = ref<ChildNode[]>([])
const saving = ref(false)
const error = ref<string | null>(null)

watch(() => props.open, (open) => {
  if (!open) return
  items.value = [...props.children].sort((a, b) => a.sortOrder - b.sortOrder || a.id - b.id)
  error.value = null
})

function move(index: number, direction: -1 | 1) {
  const target = index + direction
  if (target < 0 || target >= items.value.length) return
  const next = [...items.value]
  const current = next[index]!
  next[index] = next[target]!
  next[target] = current
  items.value = next
}

async function save() {
  if (saving.value) return
  saving.value = true
  error.value = null
  try {
    await orderNodes(props.parentId, items.value.map((item) => item.id))
    emit('saved')
    emit('close')
  } catch (err) {
    error.value = errorMessage(err)
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <BaseModal :open="open" :busy="saving" :title="parentId == null ? '调整根节点顺序' : '调整子节点顺序'" @close="emit('close')">
    <div class="flex flex-col gap-2">
      <p v-if="parentTitle" class="text-xs text-muted">父节点：{{ parentTitle }}</p>
      <div class="overflow-hidden rounded-md border border-line">
        <div
          v-for="(item, index) in items"
          :key="item.id"
          class="flex items-center gap-2 border-b border-line px-2.5 py-1.5 last:border-b-0"
        >
          <span class="w-5 text-right text-xs text-faint">{{ index + 1 }}</span>
          <span class="min-w-0 flex-1 break-words text-[13px] text-ink">{{ item.title }}</span>
          <button type="button" class="icon-btn" :disabled="saving || index === 0" :aria-label="`上移 ${item.title}`" @click="move(index, -1)">
            <ArrowUp class="h-3.5 w-3.5" />
          </button>
          <button type="button" class="icon-btn" :disabled="saving || index === items.length - 1" :aria-label="`下移 ${item.title}`" @click="move(index, 1)">
            <ArrowDown class="h-3.5 w-3.5" />
          </button>
        </div>
      </div>
      <p v-if="items.length === 0" class="empty-state">暂无可排序的节点</p>
      <p v-if="error" role="alert" class="text-xs text-danger">{{ error }}</p>
    </div>

    <template #footer>
      <button type="button" class="btn" :disabled="saving" @click="emit('close')">取消</button>
      <button type="button" class="btn btn-primary" :disabled="saving || items.length < 2" @click="save">
        <Loader2 v-if="saving" class="h-3.5 w-3.5 animate-spin" />
        {{ saving ? '保存中…' : '保存顺序' }}
      </button>
    </template>
  </BaseModal>
</template>
