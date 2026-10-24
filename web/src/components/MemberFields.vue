<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import {
  NAlert,
  NButton,
  NFormItemGi,
  NGrid,
  NSelect,
  type SelectGroupOption,
  type SelectOption,
} from 'naive-ui'
import { useAuth } from '@/composables/useAuth'
import {
  fetchNode,
  fetchNodeMembers,
  fetchUsers,
  type EffectiveMembers,
  type UserSummary,
} from '@/lib/api'
import { errorMessage } from '@/lib/utils'
const props = withDefaults(
  defineProps<{
    open: boolean
    parentId?: number | null
    rootId?: number | null
    disabled?: boolean
    creating?: boolean
  }>(),
  { parentId: null, rootId: null, disabled: false, creating: false },
)
const leaders = defineModel<string[]>('leaders', { required: true }),
  members = defineModel<string[]>('members', { required: true }),
  loading = defineModel<boolean>('loading', { required: true })
const { user: currentUser } = useAuth()
const users = ref<UserSummary[]>([]),
  parentParticipants = ref(new Set<string>()),
  rootParticipants = ref(new Set<string>()),
  error = ref<string | null>(null),
  retry = ref(0)

function participantSubjects(group: EffectiveMembers | null) {
  return new Set(
    [...(group?.leaders ?? []), ...(group?.members ?? [])].map(
      (entry) => entry.user.subject,
    ),
  )
}

watch(
  [() => props.open, () => props.parentId, () => props.rootId, retry],
  async ([open, parentId, currentRootId], _, onCleanup) => {
    if (!open) return
    let active = true
    onCleanup(() => {
      active = false
      loading.value = false
    })
    loading.value = true
    error.value = null
    users.value = []
    parentParticipants.value = new Set()
    rootParticipants.value = new Set()
    try {
      const [allUsers, parent, parentGroup] = await Promise.all([
        fetchUsers(),
        parentId == null ? null : fetchNode(parentId),
        parentId == null ? null : fetchNodeMembers(parentId),
      ])
      if (!active) return
      users.value = allUsers
      parentParticipants.value = participantSubjects(parentGroup)
      if (!members.value.length && parent) {
        const selectedLeaders = leaders.value.length
          ? leaders.value
          : props.creating && currentUser.value
            ? [currentUser.value.subject]
            : []
        members.value = parent.directMembers
          .filter((user) => !selectedLeaders.includes(user.subject))
          .map((user) => user.subject)
      }
      const rootId = parent?.breadcrumb[0]?.id ?? parent?.id ?? currentRootId
      const rootGroup =
        rootId == null || rootId === parentId
          ? null
          : await fetchNodeMembers(rootId)
      if (!active) return
      rootParticipants.value = participantSubjects(rootGroup)
    } catch (err) {
      if (active) error.value = errorMessage(err, '无法加载可选人员')
    } finally {
      if (active) loading.value = false
    }
  },
  { immediate: true },
)

function optionsFor(excluded: string[]): Array<SelectOption | SelectGroupOption> {
  const options = users.value.map((user) => ({
    value: user.subject,
    disabled: excluded.includes(user.subject),
    label:
      user.userName +
      (user.positions.length
        ? ` · ${user.positions.map((position) => position.label).join('、')}`
        : ''),
  }))
  const parent = options.filter((option) =>
    parentParticipants.value.has(option.value),
  )
  const root = options.filter(
    (option) =>
      !parentParticipants.value.has(option.value) &&
      rootParticipants.value.has(option.value),
  )
  if (!parent.length && !root.length) return options
  const others = options.filter(
    (option) =>
      !parentParticipants.value.has(option.value) &&
      !rootParticipants.value.has(option.value),
  )
  return [
    { type: 'group' as const, label: '父节点参与者', children: parent },
    { type: 'group' as const, label: '根节点参与者', children: root },
    { type: 'group' as const, label: '其他用户', children: others },
  ].filter((group) => group.children.length)
}
const leaderOptions = computed(() => optionsFor(members.value))
const memberOptions = computed(() => optionsFor(leaders.value))
</script>
<template>
  <NAlert v-if="error" type="error" class="member-load-error">
    {{ error }} <NButton text @click="retry++">重新加载</NButton>
  </NAlert>
  <NGrid cols="1 s:2" responsive="screen" :x-gap="16" :y-gap="16"
    ><NFormItemGi
      label="直接负责人"
      :required="!creating"
      :show-feedback="false"
      ><NSelect
        v-model:value="leaders"
        aria-label="直接负责人"
        :options="leaderOptions"
        multiple
        filterable
        :loading="loading"
        :disabled="disabled || loading"
        :placeholder="
          creating ? '留空则由你负责' : '搜索并选择负责人'
        " /></NFormItemGi
    ><NFormItemGi label="直接成员" :show-feedback="false"
      ><NSelect
        v-model:value="members"
        aria-label="直接成员"
        :options="memberOptions"
        multiple
        filterable
        clearable
        :loading="loading"
        :disabled="disabled || loading"
        :placeholder="
          parentId == null ? '搜索并选择成员' : '留空使用父节点的直接成员'
        " /></NFormItemGi
  ></NGrid>
</template>
<style scoped>
.member-load-error {
  margin-bottom: 16px;
}
</style>
