<script setup lang="ts">
const emit = defineEmits<{
  "close": [];
  "deleted": [];
  "update:open": [open: boolean];
}>();
const props = defineProps<{ open: boolean; node: NodeDetail }>();
const deleting = ref(false);
const error = ref<string | null>(null);
watch(
  () => props.open,
  () => {
    error.value = null;
  },
);

async function confirm() {
  if (deleting.value) {
    return;
  }
  deleting.value = true;
  error.value = null;
  try {
    await deleteNode(props.node.id);
    emit("deleted");
    emit("close");
  } catch (err) {
    error.value = errorMessage(err, "删除节点失败");
  } finally {
    deleting.value = false;
  }
}

function close() {
  emit("update:open", false);
  emit("close");
}
</script>

<template>
  <ConfirmAction
    :busy="deleting"
    confirm-text="确认删除"
    :error
    :open
    title="删除节点"
    @close="close"
    @confirm="confirm"
  >
    <p>
      将删除「{{ node.title }}」及其所有下级节点，共
      {{ node.subtreeNodeCount }} 个节点。
    </p>
    <p>
      这些节点的全部进展记录和
      {{ node.subtreeAttachmentCount }} 张图片也会一并删除，操作不可撤销。
    </p>
  </ConfirmAction>
</template>
