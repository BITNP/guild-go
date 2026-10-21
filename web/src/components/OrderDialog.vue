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
  <BaseModal :open="open" :title="parentId == null ? '调整根节点顺序' : `调整「${parentTitle}」的子节点顺序`" @close="emit('close')">
    <div class="flex flex-col gap-2">
      <p class="text-[11px] text-muted">顺序只影响同一父节点下的直接子节点；点击箭头调整，保存后立即生效。</p>
      <div class="overflow-hidden rounded-md border border-line">
        <div
          v-for="(item, index) in items"
          :key="item.id"
          class="flex items-center gap-2 border-b border-line px-2.5 py-1.5 last:border-b-0"
        >
          <span class="w-5 text-right text-[10px] text-faint">{{ index + 1 }}</span>
          <span class="min-w-0 flex-1 truncate text-[12px] text-ink">{{ item.title }}</span>
          <button type="button" class="icon-btn" :disabled="index === 0" aria-label="上移" @click="move(index, -1)">
            <ArrowUp class="h-3.5 w-3.5" />
          </button>
          <button type="button" class="icon-btn" :disabled="index === items.length - 1" aria-label="下移" @click="move(index, 1)">
            <ArrowDown class="h-3.5 w-3.5" />
          </button>
        </div>
      </div>
      <p v-if="error" class="text-[11px] text-danger">{{ error }}</p>
    </div>

    <template #footer>
      <button type="button" class="btn" :disabled="saving" @click="emit('close')">取消</button>
      <button type="button" class="btn btn-primary" :disabled="saving" @click="save">
        <Loader2 v-if="saving" class="h-3.5 w-3.5 animate-spin" />
        {{ saving ? '保存中…' : '保存顺序' }}
      </button>
    </template>
  </BaseModal>
</template>