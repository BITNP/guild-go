<script setup lang="ts">
import type { NodeDetail } from "@/api";

const emit = defineEmits<{
  "update:open": [open: boolean];
  "saved": [node: NodeDetail];
}>();
const props = withDefaults(
  defineProps<{
    open: boolean;
    mode: "create" | "edit";
    parentId?: number | null;
    parentTitle?: string;
    node?: NodeDetail | null;
  }>(),
  { parentId: null, parentTitle: "", node: null },
);
const title = ref("");
const description = ref("");
const due = ref<number | null>(null);
const tags = ref("");
const completed = ref(false);
const leaders = ref<string[]>([]);
const members = ref<string[]>([]);
const loadingChoices = ref(false);
const saving = ref(false);
const error = ref<string | null>(null);
const creating = computed(() => props.mode === "create");
const heading = computed(() =>
  creating.value
    ? props.parentId == null
      ? "新建根节点"
      : "新建子节点"
    : "编辑节点",
);
watch(
  () => props.open,
  (open) => {
    if (!open) {
      return;
    }
    const node = creating.value ? null : props.node;
    title.value = node?.title ?? "";
    description.value = node?.description ?? "";
    due.value = node?.dueAt ? new Date(node.dueAt).getTime() : null;
    tags.value = node?.tags.join("，") ?? "";
    completed.value = !!node?.completedAt;
    leaders.value = [];
    members.value = [];
    error.value = null;
    loadingChoices.value = false;
  },
  { immediate: true },
);

async function submit() {
  if (saving.value || loadingChoices.value) {
    return;
  }
  error.value = null;
  const cleanTitle = title.value.trim();
  if (!cleanTitle || [...cleanTitle].length > 200) {
    error.value = cleanTitle ? "标题不能超过 200 字" : "标题不能为空";

    return;
  }
  saving.value = true;
  try {
    const data = {
      title: cleanTitle,
      description: description.value.trim(),
      dueAt: due.value === null ? null : new Date(due.value).toISOString(),
      tags: splitTags(tags.value),
    };
    const result = creating.value
      ? await createNode({
          ...data,
          parentId: props.parentId,
          leaderSubjects: leaders.value,
          memberSubjects: members.value,
        })
      : await updateNode(props.node!.id, {
          ...data,
          completed: completed.value,
        });
    emit("saved", result);
    emit("update:open", false);
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
    :title="heading"
    wide
    @close="emit('update:open', false)"
  >
    <NForm :disabled="saving" label-placement="top" @submit.prevent="submit">
      <p v-if="parentTitle" class="parent-context">父节点：{{ parentTitle }}</p>
      <section aria-label="节点内容" class="form-section">
        <h3>节点内容</h3>
        <NFormItem label="标题" required :show-feedback="false">
          <NInput
            v-model:value="title"
            autofocus
            :input-props="{ 'aria-label': '标题' }"
            :maxlength="200"
            placeholder="例如：秋季招新"
            show-count
          />
        </NFormItem>
        <NFormItem label="描述" :show-feedback="false">
          <NInput
            v-model:value="description"
            :autosize="{ minRows: 3, maxRows: 8 }"
            :input-props="{ 'aria-label': '描述' }"
            placeholder="说明需要完成的工作"
            type="textarea"
          />
        </NFormItem>
      </section>
      <section aria-label="工作安排" class="form-section">
        <h3>工作安排</h3>
        <NGrid cols="1 s:2" responsive="screen" :x-gap="16" :y-gap="16">
          <NFormItemGi label="预期完成时间" :show-feedback="false">
            <label class="date-input">
              <span class="visually-hidden">预期完成时间</span>
              <NDatePicker
                v-model:value="due"
                class="date-input"
                clearable
                format="yyyy-MM-dd HH:mm"
                :time-picker-props="{ format: 'HH:mm' }"
                type="datetime"
              />
            </label>
          </NFormItemGi>
          <NFormItemGi label="标签" :show-feedback="false">
            <NInput
              v-model:value="tags"
              :input-props="{ 'aria-label': '标签' }"
              placeholder="用逗号分隔，例如：招新，宣传"
            />
          </NFormItemGi>
        </NGrid>
        <NCheckbox v-if="!creating" v-model:checked="completed">
          标记为已完成
        </NCheckbox>
      </section>
      <section v-if="creating" aria-label="参与人员" class="form-section">
        <h3>参与人员</h3>
        <MemberFields
          v-model:leaders="leaders"
          v-model:loading="loadingChoices"
          v-model:members="members"
          creating
          :disabled="saving"
          :open
          :parent-id
        />
      </section>
      <NAlert v-if="error" class="form-error" role="alert" type="error">
        {{ error }}
      </NAlert>
      <NSpace class="form-actions" justify="end">
        <NButton :disabled="saving" @click="emit('update:open', false)">
          取消
        </NButton>
        <NButton
          attr-type="submit"
          :disabled="loadingChoices"
          :loading="saving"
          type="primary"
        >
          {{ creating ? "创建节点" : "保存修改" }}
        </NButton>
      </NSpace>
    </NForm>
  </AppModal>
</template>

<style scoped>
.parent-context {
  margin: 0 0 20px;
  color: var(--guild-muted);
  font-size: 13px;
  overflow-wrap: anywhere;
}
.form-section {
  display: grid;
  gap: 16px;
}
.form-section + .form-section {
  margin-top: 20px;
  padding-top: 20px;
  border-top: 1px solid var(--guild-border);
}
.form-section h3 {
  margin: 0;
  font-size: 14px;
  font-weight: 650;
  line-height: 22px;
}
.form-actions {
  margin-top: 24px;
  padding-top: 16px;
  border-top: 1px solid var(--guild-border);
}
.date-input {
  width: 100%;
}
.visually-hidden {
  position: absolute;
  width: 1px;
  height: 1px;
  clip-path: inset(50%);
  overflow: hidden;
  white-space: nowrap;
}
.form-error {
  margin-top: 16px;
}
</style>
