<script setup lang="ts">
const emit = defineEmits<{ close: []; saved: [node: NodeDetail] }>();
const props = defineProps<{ open: boolean; node: NodeDetail }>();
const { isAdmin, isManager } = useAuth();
const nodes = ref<NodeSummary[]>([]);
const target = ref<number | null>(null);
const loading = ref(false);
const saving = ref(false);
const error = ref<string | null>(null);
const retry = ref(0);

watch(
  [() => props.open, () => props.node.id, retry],
  async ([open], _, onCleanup) => {
    if (!open) {
      return;
    }
    let active = true;
    onCleanup(() => {
      active = false;
    });
    target.value = null;
    nodes.value = [];
    error.value = null;
    loading.value = true;
    try {
      const result = await fetchNodes();
      if (active) {
        nodes.value = result;
      }
    } catch (err) {
      if (active) {
        error.value = errorMessage(err, "无法加载目标节点");
      }
    } finally {
      if (active) {
        loading.value = false;
      }
    }
  },
  { immediate: true },
);

const options = computed(() =>
  buildMoveTargets(nodes.value, props.node, {
    canMoveAnyNode: isAdmin.value,
    canMoveToRoot: isAdmin.value || isManager.value,
  }),
);

async function save() {
  if (saving.value || loading.value || target.value == null) {
    return;
  }
  saving.value = true;
  error.value = null;
  try {
    const result = await moveNode(
      props.node.id,
      target.value === 0 ? null : target.value,
    );
    emit("saved", result);
    emit("close");
  } catch (err) {
    error.value = errorMessage(err, "移动节点失败");
  } finally {
    saving.value = false;
  }
}
</script>

<template>
  <AppModal :busy="saving" :open title="移动节点" @close="emit('close')">
    <NSpace :size="16" vertical>
      <NText>当前节点：{{ node.title }}</NText>
      <NText depth="3">
        当前位置：{{
          node.breadcrumb.map((item) => item.title).join(" / ") || "总览"
        }}
      </NText>
      <NFormItem label="移动到" :show-feedback="false">
        <NSelect
          v-model:value="target"
          aria-label="目标父节点"
          :disabled="loading || saving"
          filterable
          :loading
          :options
          placeholder="搜索并选择目标父节点"
        />
      </NFormItem>
      <NText depth="3">
        节点及所有下级节点将一同移动到目标列表末尾。成员和内容保留，继承权限将按新位置计算。
      </NText>
      <NEmpty
        v-if="!loading && !error && options.length === 0"
        description="暂无可移动的位置"
        size="small"
      />
      <NAlert v-if="error" role="alert" type="error">
        {{ error }}
        <NButton v-if="nodes.length === 0" text @click="retry++">
          重新加载
        </NButton>
      </NAlert>
      <NSpace justify="end">
        <NButton :disabled="saving" @click="emit('close')">取消</NButton>
        <NButton
          :disabled="loading || target == null"
          :loading="saving"
          type="primary"
          @click="save"
        >
          确认移动
        </NButton>
      </NSpace>
    </NSpace>
  </AppModal>
</template>
