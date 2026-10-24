<script setup lang="ts">
import { ref, watch } from 'vue'
import {
  NAlert,
  NButton,
  NEmpty,
  NList,
  NListItem,
  NSpace,
  NText,
} from 'naive-ui'
import { ArrowDown, ArrowUp } from '@lucide/vue'
import AppModal from '@/components/AppModal.vue'
import { orderNodes, type ChildNode } from '@/lib/api'
import { errorMessage } from '@/lib/utils'
const props = defineProps<{
    open: boolean
    parentId: number | null
    parentTitle: string
    children: ChildNode[]
  }>(),
  emit = defineEmits<{ close: []; saved: [] }>()
const items = ref<ChildNode[]>([]),
  saving = ref(false),
  error = ref<string | null>(null)
watch(
  () => props.open,
  (open) => {
    if (open) {
      items.value = [...props.children].sort(
        (a, b) => a.sortOrder - b.sortOrder || a.id - b.id,
      )
      error.value = null
    }
  },
  { immediate: true },
)
function move(index: number, direction: -1 | 1) {
  const target = index + direction
  if (target < 0 || target >= items.value.length) return
  const next = [...items.value]
  ;[next[index], next[target]] = [next[target]!, next[index]!]
  items.value = next
}
async function save() {
  if (saving.value) return
  saving.value = true
  error.value = null
  try {
    await orderNodes(
      props.parentId,
      items.value.map((item) => item.id),
    )
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
  <AppModal
    :open="open"
    :title="parentId == null ? '调整根节点顺序' : '调整子节点顺序'"
    :busy="saving"
    @close="emit('close')"
    ><NSpace vertical :size="16"
      ><NText v-if="parentTitle" depth="3">父节点：{{ parentTitle }}</NText
      ><NList bordered
        ><NListItem v-for="(item, index) in items" :key="item.id"
          ><span class="order-label">{{ index + 1 }}. {{ item.title }}</span
          ><template #suffix
            ><NSpace :wrap="false" :size="4"
              ><NButton
                quaternary
                :disabled="saving || index === 0"
                :aria-label="`上移 ${item.title}`"
                @click="move(index, -1)"
                ><ArrowUp :size="16" /></NButton
              ><NButton
                quaternary
                :disabled="saving || index === items.length - 1"
                :aria-label="`下移 ${item.title}`"
                @click="move(index, 1)"
                ><ArrowDown
                  :size="16" /></NButton></NSpace></template></NListItem></NList
      ><NEmpty
        v-if="!items.length"
        description="暂无可排序的节点"
        size="small"
      /><NAlert v-if="error" type="error">{{ error }}</NAlert
      ><NSpace justify="end"
        ><NButton :disabled="saving" @click="emit('close')">取消</NButton
        ><NButton
          type="primary"
          :loading="saving"
          :disabled="items.length < 2"
          @click="save"
          >保存顺序</NButton
        ></NSpace
      ></NSpace
    ></AppModal
  >
</template>
<style scoped>
.order-label {
  overflow-wrap: anywhere;
}
</style>
