<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { Loader2, UserPlus, X } from '@lucide/vue'
import BaseModal from '@/components/BaseModal.vue'
import UserAvatar from '@/components/UserAvatar.vue'
import UserPicker from '@/components/UserPicker.vue'
import {
  fetchNodeMembers,
  fetchUsers,
  saveNodeMembers,
  type EffectiveMembers,
  type NodeDetail,
  type UserSummary,
} from '@/lib/api'
import { errorMessage } from '@/lib/utils'

const props = defineProps<{
  open: boolean
  node: NodeDetail
}>()

const emit = defineEmits<{
  close: []
  saved: [node: NodeDetail]
}>()

const tab = ref<'direct' | 'inherited'>('direct')
const leaders = ref<string[]>([])
const members = ref<string[]>([])
const users = ref<UserSummary[]>([])
const effective = ref<EffectiveMembers | null>(null)
const loading = ref(false)
const loadingEffective = ref(false)
const saving = ref(false)
const error = ref<string | null>(null)
const pickerGroup = ref<'leader' | 'member' | null>(null)

const userMap = computed(() => new Map(users.value.map((user) => [user.subject, user])))
const pickerOpen = computed({
  get: () => pickerGroup.value !== null,
  set: (value: boolean) => { if (!value) pickerGroup.value = null },
})
const pickerSelected = computed({
  get: () => pickerGroup.value === 'leader' ? leaders.value : members.value,
  set: (value: string[]) => {
    if (pickerGroup.value === 'leader') leaders.value = value
    else if (pickerGroup.value === 'member') members.value = value
  },
})
const pickerExclude = computed(() => pickerGroup.value === 'leader' ? members.value : leaders.value)

function userFor(subject: string): UserSummary {
  return userMap.value.get(subject) ?? { subject, userName: subject, avatar: null, positions: [] }
}

watch(() => props.open, async (open) => {
  if (!open) return
  tab.value = 'direct'
  error.value = null
  leaders.value = props.node.directLeaders.map((user) => user.subject)
  members.value = props.node.directMembers.map((user) => user.subject)
  loading.value = users.value.length === 0
  if (loading.value) {
    try {
      users.value = await fetchUsers()
    } catch (err) {
      error.value = errorMessage(err, '无法加载用户列表')
    } finally {
      loading.value = false
    }
  }
  loadingEffective.value = true
  try {
    effective.value = await fetchNodeMembers(props.node.id)
  } catch (err) {
    if (!error.value) error.value = errorMessage(err, '无法加载继承成员')
  } finally {
    loadingEffective.value = false
  }
})

function remove(group: 'leader' | 'member', subject: string) {
  if (group === 'leader') leaders.value = leaders.value.filter((item) => item !== subject)
  else members.value = members.value.filter((item) => item !== subject)
}

async function save() {
  error.value = null
  if (leaders.value.length === 0) {
    error.value = '节点至少需要一个直接负责人'
    return
  }
  saving.value = true
  try {
    const updated = await saveNodeMembers(props.node.id, leaders.value, members.value)
    emit('saved', updated)
    emit('close')
  } catch (err) {
    error.value = errorMessage(err)
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <BaseModal :open="open" title="节点成员" wide @close="emit('close')">
    <div class="flex flex-col gap-3">
      <div class="flex items-center gap-1 rounded-md border border-line bg-surface-2 p-0.5 text-[11px]">
        <button
          type="button"
          class="flex-1 rounded px-2 py-1"
          :class="tab === 'direct' ? 'bg-surface text-ink shadow-sm' : 'text-muted'"
          @click="tab = 'direct'"
        >
          直接成员（可编辑）
        </button>
        <button
          type="button"
          class="flex-1 rounded px-2 py-1"
          :class="tab === 'inherited' ? 'bg-surface text-ink shadow-sm' : 'text-muted'"
          @click="tab = 'inherited'"
        >
          有效成员（含继承）
        </button>
      </div>

      <p class="rounded-md border border-line bg-surface-2 px-3 py-2 text-[11px] leading-5 text-muted">
        负责人管理权向下继承，成员身份向上继承。只保存直接关系，继承关系实时计算。
      </p>

      <template v-if="tab === 'direct'">
        <p v-if="loading" class="flex items-center gap-1 text-xs text-muted"><Loader2 class="h-3.5 w-3.5 animate-spin" /> 正在加载用户…</p>

        <section class="rounded-md border border-line">
          <header class="flex items-center justify-between border-b border-line px-2.5 py-1.5">
            <span class="text-[11px] font-semibold text-muted">直接负责人</span>
            <button type="button" class="btn btn-xs" @click="pickerGroup = 'leader'">
              <UserPlus class="h-3.5 w-3.5" /> 添加
            </button>
          </header>
          <div class="flex min-h-10 flex-wrap gap-1.5 p-2">
            <span v-if="leaders.length === 0" class="text-[11px] text-faint">至少需要保留一个负责人</span>
            <span
              v-for="subject in leaders"
              :key="subject"
              class="inline-flex items-center gap-1 rounded border border-line bg-surface-2 py-0.5 pl-0.5 pr-1 text-[11px]"
            >
              <UserAvatar :name="userFor(subject).userName" :avatar="userFor(subject).avatar" size-class="h-4 w-4 text-[8px]" />
              {{ userFor(subject).userName }}
              <button type="button" class="text-faint hover:text-danger" @click="remove('leader', subject)">
                <X class="h-3 w-3" />
              </button>
            </span>
          </div>
        </section>

        <section class="rounded-md border border-line">
          <header class="flex items-center justify-between border-b border-line px-2.5 py-1.5">
            <span class="text-[11px] font-semibold text-muted">直接成员</span>
            <button type="button" class="btn btn-xs" @click="pickerGroup = 'member'">
              <UserPlus class="h-3.5 w-3.5" /> 添加
            </button>
          </header>
          <div class="flex min-h-10 flex-wrap gap-1.5 p-2">
            <span v-if="members.length === 0" class="text-[11px] text-faint">暂无直接成员</span>
            <span
              v-for="subject in members"
              :key="subject"
              class="inline-flex items-center gap-1 rounded border border-line bg-surface-2 py-0.5 pl-0.5 pr-1 text-[11px]"
            >
              <UserAvatar :name="userFor(subject).userName" :avatar="userFor(subject).avatar" size-class="h-4 w-4 text-[8px]" />
              {{ userFor(subject).userName }}
              <button type="button" class="text-faint hover:text-danger" @click="remove('member', subject)">
                <X class="h-3 w-3" />
              </button>
            </span>
          </div>
        </section>
      </template>

      <template v-else>
        <p v-if="loadingEffective" class="flex items-center gap-1 text-xs text-muted"><Loader2 class="h-3.5 w-3.5 animate-spin" /> 正在计算继承关系…</p>
        <div v-else-if="effective" class="grid gap-3 sm:grid-cols-2">
          <section class="rounded-md border border-line">
            <header class="border-b border-line px-2.5 py-1.5 text-[11px] font-semibold text-muted">有效负责人</header>
            <div class="divide-y divide-line">
              <div v-for="entry in effective.leaders" :key="entry.user.subject" class="flex items-start gap-2 px-2.5 py-1.5">
                <UserAvatar :name="entry.user.userName" :avatar="entry.user.avatar" size-class="h-5 w-5 text-[8px]" />
                <div class="min-w-0">
                  <p class="truncate text-[12px] text-ink">{{ entry.user.userName }}</p>
                  <p class="text-[10px] text-faint">
                    {{ entry.direct ? '直接负责人' : `继承自 ${entry.sources.map((source) => source.title).join('、')}` }}
                  </p>
                </div>
              </div>
            </div>
          </section>
          <section class="rounded-md border border-line">
            <header class="border-b border-line px-2.5 py-1.5 text-[11px] font-semibold text-muted">有效成员</header>
            <div class="max-h-64 divide-y divide-line overflow-y-auto">
              <div v-for="entry in effective.members" :key="entry.user.subject" class="flex items-start gap-2 px-2.5 py-1.5">
                <UserAvatar :name="entry.user.userName" :avatar="entry.user.avatar" size-class="h-5 w-5 text-[8px]" />
                <div class="min-w-0">
                  <p class="truncate text-[12px] text-ink">{{ entry.user.userName }}</p>
                  <p class="text-[10px] text-faint">
                    {{ entry.direct ? '直接成员' : `继承自 ${entry.sources.map((source) => source.title).join('、')}` }}
                  </p>
                </div>
              </div>
            </div>
          </section>
        </div>
        <p class="text-[11px] text-faint">系统管理员可评论节点，但不计入有效成员。</p>
      </template>

      <p v-if="error" class="rounded-md border border-danger/20 bg-danger-soft px-3 py-1.5 text-[11px] text-danger">{{ error }}</p>
    </div>

    <template #footer>
      <button type="button" class="btn" :disabled="saving" @click="emit('close')">关闭</button>
      <button v-if="tab === 'direct'" type="button" class="btn btn-primary" :disabled="saving" @click="save">
        <Loader2 v-if="saving" class="h-3.5 w-3.5 animate-spin" />
        {{ saving ? '保存中…' : '保存成员' }}
      </button>
    </template>
  </BaseModal>

  <UserPicker
    v-model:selected="pickerSelected"
    :open="pickerOpen"
    :title="pickerGroup === 'leader' ? '选择直接负责人' : '选择直接成员'"
    :users="users"
    :exclude="pickerExclude"
    @close="pickerGroup = null"
  />
</template>