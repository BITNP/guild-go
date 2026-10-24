<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import {
  NAlert,
  NButton,
  NCheckbox,
  NDatePicker,
  NForm,
  NFormItem,
  NFormItemGi,
  NGrid,
  NInput,
  NSpace,
} from 'naive-ui'
import AppModal from '@/components/AppModal.vue'
import MemberFields from '@/components/MemberFields.vue'
import { createNode, updateNode, type NodeDetail } from '@/lib/api'
import { errorMessage, splitTags } from '@/lib/utils'
const props = withDefaults(
  defineProps<{
    open: boolean
    mode: 'create' | 'edit'
    parentId?: number | null
    parentTitle?: string
    node?: NodeDetail | null
  }>(),
  { parentId: null, parentTitle: '', node: null },
)
const emit = defineEmits<{
  'update:open': [open: boolean]
  saved: [node: NodeDetail]
}>()
const title = ref(''),
  description = ref(''),
  due = ref<number | null>(null),
  tags = ref(''),
  completed = ref(false),
  leaders = ref<string[]>([]),
  members = ref<string[]>([])
const loadingChoices = ref(false),
  saving = ref(false),
  error = ref<string | null>(null)
const creating = computed(() => props.mode === 'create'),
  heading = computed(() =>
    creating.value
      ? props.parentId == null
        ? '新建根节点'
        : '新建子节点'
      : '编辑节点',
  )
watch(
  () => props.open,
  (open) => {
    if (!open) return
    const node = creating.value ? null : props.node
    title.value = node?.title ?? ''
    description.value = node?.description ?? ''
    due.value = node?.dueAt ? new Date(node.dueAt).getTime() : null
    tags.value = node?.tags.join('，') ?? ''
    completed.value = !!node?.completedAt
    leaders.value = []
    members.value = []
    error.value = null
    loadingChoices.value = false
  },
  { immediate: true },
)
async function submit() {
  if (saving.value || loadingChoices.value) return
  error.value = null
  const cleanTitle = title.value.trim()
  if (!cleanTitle || [...cleanTitle].length > 200) {
    error.value = cleanTitle ? '标题不能超过 200 字' : '标题不能为空'
    return
  }
  saving.value = true
  try {
    const data = {
      title: cleanTitle,
      description: description.value.trim(),
      dueAt: due.value === null ? null : new Date(due.value).toISOString(),
      tags: splitTags(tags.value),
    }
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
        })
    emit('saved', result)
    emit('update:open', false)
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
    :title="heading"
    :busy="saving"
    wide
    @close="emit('update:open', false)"
    ><NForm label-placement="top" :disabled="saving" @submit.prevent="submit">
      <p v-if="parentTitle" class="parent-context">父节点：{{ parentTitle }}</p>
      <section class="form-section" aria-label="节点内容">
        <h3>节点内容</h3>
        <NFormItem label="标题" required :show-feedback="false"
          ><NInput
            v-model:value="title"
            :input-props="{ 'aria-label': '标题' }"
            placeholder="例如：秋季招新"
            :maxlength="200"
            show-count
            autofocus /></NFormItem
        ><NFormItem label="描述" :show-feedback="false"
          ><NInput
            v-model:value="description"
            :input-props="{ 'aria-label': '描述' }"
            type="textarea"
            :autosize="{ minRows: 3, maxRows: 8 }"
            placeholder="说明需要完成的工作"
        /></NFormItem>
      </section>
      <section class="form-section" aria-label="工作安排">
        <h3>工作安排</h3>
        <NGrid cols="1 s:2" responsive="screen" :x-gap="16" :y-gap="16"
          ><NFormItemGi label="预期完成时间" :show-feedback="false"
            ><label class="date-input">
              <span class="visually-hidden">预期完成时间</span>
              <NDatePicker
                v-model:value="due"
                type="datetime"
                format="yyyy-MM-dd HH:mm"
                :time-picker-props="{ format: 'HH:mm' }"
                clearable
                class="date-input"
              /> </label></NFormItemGi
          ><NFormItemGi label="标签" :show-feedback="false"
            ><NInput
              v-model:value="tags"
              :input-props="{ 'aria-label': '标签' }"
              placeholder="用逗号分隔，例如：招新，宣传" /></NFormItemGi></NGrid
        ><NCheckbox v-if="!creating" v-model:checked="completed"
          >标记为已完成</NCheckbox
        >
      </section>
      <section v-if="creating" class="form-section" aria-label="参与人员">
        <h3>参与人员</h3>
        <MemberFields
          v-model:leaders="leaders"
          v-model:members="members"
          v-model:loading="loadingChoices"
          :open="open"
          :parent-id="parentId"
          :disabled="saving"
          creating
        />
      </section>
      <NAlert v-if="error" type="error" role="alert" class="form-error">{{
        error
      }}</NAlert
      ><NSpace justify="end" class="form-actions"
        ><NButton :disabled="saving" @click="emit('update:open', false)"
          >取消</NButton
        ><NButton
          type="primary"
          attr-type="submit"
          :loading="saving"
          :disabled="loadingChoices"
          >{{
          creating ? '创建节点' : '保存修改'
        }}</NButton></NSpace
      ></NForm
    ></AppModal
  >
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
