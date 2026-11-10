<script setup lang="ts">
import type { ChildNode } from "@/api";

const emit = defineEmits<{ close: []; saved: [] }>();
const props = defineProps<{
  open: boolean;
  parentId: number | null;
  parentTitle: string;
  children: ChildNode[];
}>();
const items = ref<ChildNode[]>([]);
const saving = ref(false);
const error = ref<string | null>(null);
watch(
  () => props.open,
  (open) => {
    if (open) {
      items.value = [...props.children].sort(
        (a, b) => a.sortOrder - b.sortOrder || a.id - b.id,
      );
      error.value = null;
    }
  },
  { immediate: true },
);

function move(index: number, direction: -1 | 1) {
  const target = index + direction;
  if (target < 0 || target >= items.value.length) {
    return;
  }
  const next = [...items.value];
  [next[index], next[target]] = [next[target], next[index]];
  items.value = next;
}

async function save() {
  if (saving.value) {
    return;
  }
  saving.value = true;
  error.value = null;
  try {
    await orderNodes(
      props.parentId,
      items.value.map((item) => item.id),
    );
    emit("saved");
    emit("close");
  } catch (err) {
    error.value = errorMessage(err);
  } finally {
    saving.value = false;
  }
}
</script>

<template>
  <AppModal
    :busy="saving"
    :open
    :title="parentId == null ? '调整根节点顺序' : '调整子节点顺序'"
    @close="emit('close')"
  >
    <NSpace :size="16" vertical>
      <NText v-if="parentTitle" depth="3">父节点：{{ parentTitle }}</NText>
      <NList bordered>
        <NListItem v-for="(item, index) in items" :key="item.id">
          <span class="order-label">{{ index + 1 }}. {{ item.title }}</span>
          <template #suffix>
            <NSpace :size="4" :wrap="false">
              <NButton
                :aria-label="`上移 ${item.title}`"
                :disabled="saving || index === 0"
                quaternary
                @click="move(index, -1)"
              >
                <ArrowUp :size="16" />
              </NButton>
              <NButton
                :aria-label="`下移 ${item.title}`"
                :disabled="saving || index === items.length - 1"
                quaternary
                @click="move(index, 1)"
              >
                <ArrowDown :size="16" />
              </NButton>
            </NSpace>
          </template>
        </NListItem>
      </NList>
      <NEmpty
        v-if="items.length === 0"
        description="暂无可排序的节点"
        size="small"
      />
      <NAlert v-if="error" type="error">{{ error }}</NAlert>
      <NSpace justify="end">
        <NButton :disabled="saving" @click="emit('close')">取消</NButton>
        <NButton
          :disabled="items.length < 2"
          :loading="saving"
          type="primary"
          @click="save"
        >
          保存顺序
        </NButton>
      </NSpace>
    </NSpace>
  </AppModal>
</template>

<style scoped>
.order-label {
  overflow-wrap: anywhere;
}
</style>
