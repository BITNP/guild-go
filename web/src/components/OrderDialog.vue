<script setup lang="ts">
import { ArrowDown, ArrowUp, GripVertical } from "@lucide/vue";
import {
  NAlert,
  NButton,
  NEmpty,
  NList,
  NListItem,
  NSpace,
  NText,
} from "naive-ui";
import { ref, watch } from "vue";

import AppModal from "@/components/AppModal.vue";
import type { ChildNode } from "@/lib/api";
import { orderNodes } from "@/lib/api";
import { errorMessage } from "@/lib/utils";

const emit = defineEmits<{ saved: [] }>();
const props = defineProps<{
  parentId: number | null;
  parentTitle: string;
  children: ChildNode[];
}>();
const open = defineModel<boolean>("open", { required: true });
const items = ref<ChildNode[]>([]);
const saving = ref(false);
const error = ref<string | null>(null);
const draggingIndex = ref<number | null>(null);
const dragOverIndex = ref<number | null>(null);
watch(
  open,
  (isOpen) => {
    if (isOpen) {
      items.value = [...props.children].sort(
        (a, b) => a.sortOrder - b.sortOrder || a.id - b.id,
      );
      error.value = null;
      draggingIndex.value = null;
      dragOverIndex.value = null;
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

function onDragStart(index: number, event: DragEvent) {
  if (saving.value) {
    return;
  }
  draggingIndex.value = index;
  if (event.dataTransfer) {
    event.dataTransfer.effectAllowed = "move";
    event.dataTransfer.setData("text/plain", String(index));
  }
}

function onDragOver(index: number) {
  if (draggingIndex.value === null || draggingIndex.value === index) {
    dragOverIndex.value = null;

    return;
  }
  dragOverIndex.value = index;
}

function onDragLeave(index: number) {
  if (dragOverIndex.value === index) {
    dragOverIndex.value = null;
  }
}

function onDrop(index: number) {
  const from = draggingIndex.value;
  draggingIndex.value = null;
  dragOverIndex.value = null;
  if (from === null || from === index || saving.value) {
    return;
  }
  const next = [...items.value];
  const [moved] = next.splice(from, 1);
  if (!moved) {
    return;
  }
  next.splice(index, 0, moved);
  items.value = next;
}

function onDragEnd() {
  draggingIndex.value = null;
  dragOverIndex.value = null;
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
    open.value = false;
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
    @close="open = false"
  >
    <NSpace :size="16" vertical>
      <NText v-if="parentTitle" depth="3">父节点：{{ parentTitle }}</NText>
      <NList bordered class="order-list">
        <NListItem
          v-for="(item, index) in items"
          :key="item.id"
          class="order-row"
          :class="{
            'is-dragging': draggingIndex === index,
            'is-drag-over': dragOverIndex === index,
          }"
          :draggable="!saving"
          @dragend="onDragEnd"
          @dragleave="onDragLeave(index)"
          @dragover.prevent="onDragOver(index)"
          @dragstart="onDragStart(index, $event)"
          @drop.prevent="onDrop(index)"
        >
          <div class="order-item">
            <GripVertical aria-hidden="true" class="drag-handle" :size="16" />
            <span class="order-label">{{ index + 1 }}. {{ item.title }}</span>
          </div>
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
        <NButton :disabled="saving" @click="open = false">取消</NButton>
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
.order-list {
  max-height: min(60vh, 420px);
  overflow-y: auto;
}
.order-row {
  cursor: grab;
  transition:
    background-color 0.15s,
    opacity 0.15s;
}
.order-row.is-dragging {
  opacity: 0.45;
}
.order-row.is-drag-over {
  background: var(--guild-primary-soft);
}
.order-item {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}
.drag-handle {
  flex: none;
  color: var(--guild-muted);
}
.order-label {
  overflow-wrap: anywhere;
}
</style>
