<script setup lang="ts">
import { computed, ref, watch } from "vue";

import ConfirmAction from "@/components/ConfirmAction.vue";
import type { NodeDetail } from "@/lib/api";
import { deleteNode } from "@/lib/api";
import { errorMessage } from "@/lib/utils";

const emit = defineEmits<{ deleted: [] }>();
const props = defineProps<{ node: NodeDetail }>();
const open = defineModel<boolean>("open", { required: true });
const deleting = ref(false);
const error = ref<string | null>(null);
const scopeSummary = computed(() =>
  props.node.subtreeNodeCount > 1
    ? `将删除「${props.node.title}」及其所有下级节点，共 ${props.node.subtreeNodeCount} 个节点。`
    : `将删除「${props.node.title}」。`,
);
const impactSummary = computed(() => {
  const subject = props.node.subtreeNodeCount > 1 ? "这些节点" : "该节点";
  const attachments =
    props.node.subtreeAttachmentCount > 0
      ? `和 ${props.node.subtreeAttachmentCount} 张图片`
      : "";

  return `${subject}的全部进展记录${attachments}也会一并删除，操作不可撤销。`;
});
watch(open, () => {
  error.value = null;
});

async function confirm() {
  if (deleting.value) {
    return;
  }
  deleting.value = true;
  error.value = null;
  try {
    await deleteNode(props.node.id);
    emit("deleted");
    open.value = false;
  } catch (err) {
    error.value = errorMessage(err, "删除节点失败");
  } finally {
    deleting.value = false;
  }
}
</script>

<template>
  <ConfirmAction
    :busy="deleting"
    confirm-text="确认删除"
    :error
    :open
    title="删除节点"
    @close="open = false"
    @confirm="confirm"
  >
    <p>{{ scopeSummary }}</p>
    <p>{{ impactSummary }}</p>
  </ConfirmAction>
</template>
