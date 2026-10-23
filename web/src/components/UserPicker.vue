<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { Check, Search } from '@lucide/vue'
import BaseModal from '@/components/BaseModal.vue'
import UserAvatar from '@/components/UserAvatar.vue'
import type { UserSummary } from '@/lib/api'

const props = defineProps<{
  open: boolean
  title: string
  users: UserSummary[]
  selected: string[]
  exclude?: string[]
}>()

const emit = defineEmits<{
  'update:selected': [selected: string[]]
  close: []
}>()

const query = ref('')
watch(() => props.open, (open) => {
  if (open) query.value = ''
})

const filtered = computed(() => {
  const value = query.value.trim().toLowerCase()
  return props.users
    .filter((user) => {
      if (!value) return true
      return user.userName.toLowerCase().includes(value)
        || user.subject.toLowerCase().includes(value)
        || user.positions.some((position) => position.label.includes(value))
    })
    .sort((a, b) => a.userName.localeCompare(b.userName, 'zh-CN'))
})

function selectedSet(): Set<string> {
  return new Set(props.selected)
}

function isExcluded(subject: string): boolean {
  return props.exclude?.includes(subject) ?? false
}

function toggle(subject: string) {
  if (isExcluded(subject)) return
  const next = selectedSet()
  if (next.has(subject)) next.delete(subject)
  else next.add(subject)
  emit('update:selected', [...next])
}
</script>

<template>
  <BaseModal
    :open="open"
    :title="title"
    @close="emit('close')"
  >
    <div class="flex flex-col gap-3">
      <label class="relative block">
        <span class="sr-only">搜索用户</span>
        <Search class="pointer-events-none absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-faint" />
        <input v-model="query" class="input pl-8" placeholder="搜索姓名或职务" autofocus>
      </label>

      <div v-if="selected.length" class="flex flex-wrap gap-1.5 rounded-md border border-line bg-surface-2 p-2">
        <span
          v-for="subject in selected"
          :key="subject"
          class="inline-flex items-center gap-1 rounded border border-line bg-surface px-1.5 py-0.5 text-xs"
        >
          <UserAvatar
            :name="users.find((user) => user.subject === subject)?.userName ?? subject"
            :avatar="users.find((user) => user.subject === subject)?.avatar ?? null"
            size-class="h-4 w-4 text-[8px]"
          />
          {{ users.find((user) => user.subject === subject)?.userName ?? subject }}
        </span>
      </div>

      <div class="max-h-[50vh] overflow-y-auto rounded-md border border-line">
        <button
          v-for="user in filtered"
          :key="user.subject"
          type="button"
          class="flex w-full items-center gap-2.5 border-b border-line px-2.5 py-2 text-left last:border-b-0 row-hover disabled:cursor-not-allowed disabled:opacity-45"
          :disabled="isExcluded(user.subject)"
          :aria-pressed="selected.includes(user.subject)"
          @click="toggle(user.subject)"
        >
          <UserAvatar :name="user.userName" :avatar="user.avatar" size-class="h-7 w-7 text-xs" />
          <span class="min-w-0 flex-1">
            <span class="block truncate text-[13px] font-medium text-ink">{{ user.userName }}</span>
            <span v-if="user.positions.length" class="block truncate text-xs text-muted">
              {{ user.positions.map((position) => position.label).join(' · ') }}
            </span>
          </span>
          <span
            v-if="isExcluded(user.subject)"
            class="text-xs text-faint"
          >已选为另一身份</span>
          <span
            v-else
            class="flex h-5 w-5 items-center justify-center rounded border"
            :class="selected.includes(user.subject)
              ? 'border-accent bg-accent text-white'
              : 'border-line text-transparent'"
          >
            <Check class="h-3 w-3" />
          </span>
        </button>
        <p v-if="filtered.length === 0" class="px-3 py-6 text-center text-xs text-muted">没有匹配的用户</p>
      </div>
    </div>
    <template #footer>
      <span class="mr-auto text-xs text-muted">已选 {{ selected.length }} 人</span>
      <button type="button" class="btn btn-primary" @click="emit('close')">完成选择</button>
    </template>
  </BaseModal>
</template>
