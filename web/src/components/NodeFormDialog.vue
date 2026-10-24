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
import {
  createNode,
  fetchUsers,
  updateNode,
  type NodeDetail,
  type UserSummary,
} from '@/lib/api'
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
  members = ref<string[]>([]),
  users = ref<UserSummary[]>([])
const loadingUsers = ref(false),
  saving = ref(false),
  error = ref<string | null>(null),
  usersError = ref<string | null>(null)
const creating = computed(() => props.mode === 'create'),
  heading = computed(() =>
    creating.value
      ? props.parentId == null
        ? '新建根节点'
        : '新建子节点'
      : '编辑节点',
  )
async function loadUsers() {
  loadingUsers.value = true
  usersError.value = null
  try {
    users.value = await fetchUsers()
  } catch (err) {
    usersError.value = errorMessage(err, '无法加载用户列表')
  } finally {
    loadingUsers.value = false
  }
}
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
    if (creating.value) void loadUsers()
  },
  { immediate: true },
)
async function submit() {
  if (saving.value) return
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
    ><NForm label-placement="top" :disabled="saving" @submit.prevent="submit"
      ><NFormItem v-if="parentTitle" label="父节点">{{ parentTitle }}</NFormItem
      ><NFormItem label="标题" required
        ><NInput
          v-model:value="title"
          :input-props="{ 'aria-label': '标题' }"
          placeholder="例如：秋季招新"
          :maxlength="200"
          show-count
          autofocus /></NFormItem
      ><NFormItem label="描述"
        ><NInput
          v-model:value="description"
          :input-props="{ 'aria-label': '描述' }"
          type="textarea"
          :autosize="{ minRows: 3, maxRows: 8 }"
          placeholder="说明需要完成的工作" /></NFormItem
      ><NGrid cols="1 s:2" responsive="screen" :x-gap="16"
        ><NFormItemGi label="预期完成时间"
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
        ><NFormItemGi label="标签"
          ><NInput
            v-model:value="tags"
            :input-props="{ 'aria-label': '标签' }"
            placeholder="用逗号分隔，例如：招新，宣传" /></NFormItemGi></NGrid
      ><NFormItem v-if="!creating"
        ><NCheckbox v-model:checked="completed"
          >标记为已完成</NCheckbox
        ></NFormItem
      ><template v-else
        ><NAlert v-if="usersError" type="error" class="form-error"
          >{{ usersError }}
          <NButton text @click="loadUsers">重新加载</NButton></NAlert
        ><MemberFields
          v-model:leaders="leaders"
          v-model:members="members"
          :users="users"
          :loading="loadingUsers"
          :disabled="saving"
          creating /></template
      ><NAlert v-if="error" type="error" role="alert" class="form-error">{{
        error
      }}</NAlert
      ><NSpace justify="end"
        ><NButton :disabled="saving" @click="emit('update:open', false)"
          >取消</NButton
        ><NButton type="primary" attr-type="submit" :loading="saving">{{
          creating ? '创建节点' : '保存修改'
        }}</NButton></NSpace
      ></NForm
    ></AppModal
  >
</template>
<style scoped>
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
  margin-bottom: 16px;
}
</style>
