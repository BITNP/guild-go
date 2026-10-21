<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { Loader2, UserPlus, X } from '@lucide/vue'
import BaseModal from '@/components/BaseModal.vue'
import UserAvatar from '@/components/UserAvatar.vue'
import UserPicker from '@/components/UserPicker.vue'
import {
  createNode,
  fetchUsers,
  updateNode,
  type NodeDetail,
  type UserSummary,
} from '@/lib/api'
import {
  errorMessage,
  fromDatetimeLocal,
  splitTags,
  toDatetimeLocal,
} from '@/lib/utils'

const props = withDefaults(defineProps<{
  open: boolean
  mode: 'create' | 'edit'
  parentId?: number | null
  parentTitle?: string
  node?: NodeDetail | null
}>(), {
  parentId: null,
  parentTitle: '',
  node: null,
})

const emit = defineEmits<{
  'update:open': [open: boolean]
  saved: [node: NodeDetail]
}>()

const title = ref('')
const description = ref('')
const dueLocal = ref('')
const tagsText = ref('')
const completed = ref(false)
const leaders = ref<string[]>([])
const members = ref<string[]>([])
const users = ref<UserSummary[]>([])
const usersLoading = ref(false)
const usersError = ref<string | null>(null)
const saving = ref(false)
const formError = ref<string | null>(null)
const pickerGroup = ref<'leader' | 'member' | null>(null)

const isCreate = computed(() => props.mode === 'create')
const dialogTitle = computed(() => {
  if (!isCreate.value) return '编辑节点'
  return props.parentId == null ? '新建根节点' : '新建子节点'
})
const contextText = computed(() => {
  if (!isCreate.value) return props.node?.breadcrumb.map((item) => item.title).join(' / ') ?? ''
  if (props.parentId == null) return '根节点只有 manager 可以创建'
  return props.parentTitle ? `父节点：${props.parentTitle}` : '创建为当前节点的子节点'
})
const pickerOpen = computed({
  get: () => pickerGroup.value !== null,
  set: (open: boolean) => { if (!open) pickerGroup.value = null },
})
const pickerTitle = computed(() => pickerGroup.value === 'leader' ? '选择负责人' : '选择成员')
const pickerSelected = computed({
  get: () => pickerGroup.value === 'leader' ? leaders.value : members.value,
  set: (value: string[]) => {
    if (pickerGroup.value === 'leader') leaders.value = value
    else if (pickerGroup.value === 'member') members.value = value
  },
})
const pickerExclude = computed(() => pickerGroup.value === 'leader' ? members.value : leaders.value)

const userMap = computed(() => new Map(users.value.map((user) => [user.subject, user])))

function userFor(subject: string): UserSummary {
  return userMap.value.get(subject) ?? {
    subject,
    userName: subject,
    avatar: null,
    positions: [],
  }
}

function reset() {
  formError.value = null
  usersError.value = null
  pickerGroup.value = null
  if (isCreate.value) {
    title.value = ''
    description.value = ''
    dueLocal.value = ''
    tagsText.value = ''
    completed.value = false
    leaders.value = []
    members.value = []
  } else if (props.node) {
    title.value = props.node.title
    description.value = props.node.description
    dueLocal.value = toDatetimeLocal(props.node.dueAt)
    tagsText.value = props.node.tags.join(', ')
    completed.value = props.node.completedAt != null
    leaders.value = props.node.directLeaders.map((user) => user.subject)
    members.value = props.node.directMembers.map((user) => user.subject)
  }
}

watch(() => props.open, async (open) => {
  if (!open) return
  reset()
  if (isCreate.value && users.value.length === 0) {
    usersLoading.value = true
    try {
      users.value = await fetchUsers()
    } catch (err) {
      usersError.value = errorMessage(err, '无法加载用户列表')
    } finally {
      usersLoading.value = false
    }
  }
})

function close() {
  if (saving.value) return
  emit('update:open', false)
}

function removeSubject(group: 'leader' | 'member', subject: string) {
  if (group === 'leader') leaders.value = leaders.value.filter((item) => item !== subject)
  else members.value = members.value.filter((item) => item !== subject)
}

async function submit() {
  formError.value = null
  const cleanTitle = title.value.trim()
  if (!cleanTitle) {
    formError.value = '标题不能为空'
    return
  }
  if (cleanTitle.length > 200) {
    formError.value = '标题不能超过 200 字'
    return
  }
  saving.value = true
  try {
    const dueAt = fromDatetimeLocal(dueLocal.value)
    if (isCreate.value) {
      const created = await createNode({
        parentId: props.parentId,
        title: cleanTitle,
        description: description.value.trim(),
        dueAt,
        tags: splitTags(tagsText.value),
        leaderSubjects: leaders.value,
        memberSubjects: members.value,
      })
      emit('saved', created)
    } else if (props.node) {
      const updated = await updateNode(props.node.id, {
        title: cleanTitle,
        description: description.value.trim(),
        dueAt,
        tags: splitTags(tagsText.value),
        completed: completed.value,
      })
      emit('saved', updated)
    }
    emit('update:open', false)
  } catch (err) {
    formError.value = errorMessage(err)
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <BaseModal :open="open" :title="dialogTitle" wide @close="close">
    <form class="flex flex-col gap-4" @submit.prevent="submit">
      <p class="rounded-md border border-line bg-surface-2 px-3 py-2 text-xs text-muted">
        {{ contextText }}
      </p>

      <label class="flex flex-col gap-1">
        <span class="text-[11px] font-medium text-muted">标题</span>
        <input v-model="title" class="input" maxlength="200" placeholder="例如：秋季招新" autofocus>
      </label>

      <label class="flex flex-col gap-1">
        <span class="text-[11px] font-medium text-muted">描述</span>
        <textarea v-model="description" class="textarea" rows="5" placeholder="目标、背景、验收标准…" />
      </label>

      <div class="grid gap-3 sm:grid-cols-2">
        <label class="flex flex-col gap-1">
          <span class="text-[11px] font-medium text-muted">预期完成时间</span>
          <input v-model="dueLocal" type="datetime-local" class="input">
        </label>
        <label class="flex flex-col gap-1">
          <span class="text-[11px] font-medium text-muted">标签</span>
          <input v-model="tagsText" class="input" placeholder="用逗号分隔，最多 10 个">
        </label>
      </div>

      <label v-if="!isCreate" class="flex items-center gap-2 rounded-md border border-line bg-surface-2 px-3 py-2 text-[13px]">
        <input v-model="completed" type="checkbox" class="h-4 w-4 accent-[var(--color-ok)]">
        <span>标记为已完成</span>
        <span class="ml-auto text-[11px] text-muted">完成时自动记录实际完成时间</span>
      </label>

      <template v-if="isCreate">
        <div class="grid gap-3 sm:grid-cols-2">
          <section class="rounded-md border border-line">
            <header class="flex items-center justify-between border-b border-line px-2.5 py-1.5">
              <span class="text-[11px] font-semibold text-muted">负责人</span>
              <button type="button" class="btn btn-xs" @click="pickerGroup = 'leader'">
                <UserPlus class="h-3.5 w-3.5" /> 选择
              </button>
            </header>
            <div class="flex min-h-10 flex-wrap items-center gap-1.5 p-2">
              <span v-if="leaders.length === 0" class="text-[11px] text-faint">未指定时创建者成为负责人</span>
              <span
                v-for="subject in leaders"
                :key="subject"
                class="inline-flex items-center gap-1 rounded border border-line bg-surface-2 py-0.5 pl-0.5 pr-1 text-[11px]"
              >
                <UserAvatar :name="userFor(subject).userName" :avatar="userFor(subject).avatar" size-class="h-4 w-4 text-[8px]" />
                {{ userFor(subject).userName }}
                <button type="button" class="text-faint hover:text-danger" @click="removeSubject('leader', subject)">
                  <X class="h-3 w-3" />
                </button>
              </span>
            </div>
          </section>

          <section class="rounded-md border border-line">
            <header class="flex items-center justify-between border-b border-line px-2.5 py-1.5">
              <span class="text-[11px] font-semibold text-muted">成员</span>
              <button type="button" class="btn btn-xs" @click="pickerGroup = 'member'">
                <UserPlus class="h-3.5 w-3.5" /> 选择
              </button>
            </header>
            <div class="flex min-h-10 flex-wrap items-center gap-1.5 p-2">
              <span v-if="members.length === 0" class="text-[11px] text-faint">成员可查看并评论，不能编辑</span>
              <span
                v-for="subject in members"
                :key="subject"
                class="inline-flex items-center gap-1 rounded border border-line bg-surface-2 py-0.5 pl-0.5 pr-1 text-[11px]"
              >
                <UserAvatar :name="userFor(subject).userName" :avatar="userFor(subject).avatar" size-class="h-4 w-4 text-[8px]" />
                {{ userFor(subject).userName }}
                <button type="button" class="text-faint hover:text-danger" @click="removeSubject('member', subject)">
                  <X class="h-3 w-3" />
                </button>
              </span>
            </div>
          </section>
        </div>
        <p v-if="usersLoading" class="flex items-center gap-1 text-[11px] text-muted">
          <Loader2 class="h-3 w-3 animate-spin" /> 正在加载用户…
        </p>
        <p v-else-if="usersError" class="text-[11px] text-danger">{{ usersError }}</p>
        <p v-else class="text-[11px] text-faint">
          成员/负责人继承规则：子节点成员自动成为祖先成员；祖先负责人自动管理后代。
        </p>
      </template>

      <p v-else class="text-[11px] text-faint">
        负责人和成员在节点页的“成员”对话框中维护。
      </p>

      <p v-if="formError" class="rounded-md border border-danger/20 bg-danger-soft px-3 py-2 text-xs text-danger" role="alert">
        {{ formError }}
      </p>
    </form>

    <template #footer>
      <button type="button" class="btn" :disabled="saving" @click="close">取消</button>
      <button type="button" class="btn btn-primary" :disabled="saving" @click="submit">
        <Loader2 v-if="saving" class="h-3.5 w-3.5 animate-spin" />
        {{ saving ? '保存中…' : (isCreate ? '创建' : '保存') }}
      </button>
    </template>
  </BaseModal>

  <UserPicker
    v-model:selected="pickerSelected"
    :open="pickerOpen"
    :title="pickerTitle"
    :users="users"
    :exclude="pickerExclude"
    @close="pickerGroup = null"
  />
</template>