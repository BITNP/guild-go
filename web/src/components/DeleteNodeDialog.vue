<script setup lang="ts">
import { ref, watch } from "vue";

import ConfirmAction from "@/components/ConfirmAction.vue";
import type { NodeDetail } from "@/lib/api";
import { deleteNode } from "@/lib/api";
import { errorMessage } from "@/lib/utils";

const emit = defineEmits<{ deleted: [] }>();
const props = defineProps<{ node: NodeDetail }>();
const open = defineModel<boolean>("open", { required: true });
const deleting = ref(false);
const error = ref<string | null>(null);
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
