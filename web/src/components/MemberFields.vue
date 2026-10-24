<script setup lang="ts">
import { computed } from 'vue'
import { NFormItemGi, NGrid, NSelect } from 'naive-ui'
import type { UserSummary } from '@/lib/api'
const props = withDefaults(
  defineProps<{
    users: UserSummary[]
    loading?: boolean
    disabled?: boolean
    creating?: boolean
  }>(),
  { loading: false, disabled: false, creating: false },
)
const leaders = defineModel<string[]>('leaders', { required: true }),
  members = defineModel<string[]>('members', { required: true })
const options = computed(() =>
  props.users.map((user) => ({
    value: user.subject,
    label:
      user.userName +
      (user.positions.length
        ? ` · ${user.positions.map((position) => position.label).join('、')}`
        : ''),
  })),
)
const leaderOptions = computed(() =>
  options.value.map((option) => ({
    ...option,
    disabled: members.value.includes(option.value),
  })),
)
const memberOptions = computed(() =>
  options.value.map((option) => ({
    ...option,
    disabled: leaders.value.includes(option.value),
  })),
)
</script>
<template>
  <NGrid cols="1 s:2" responsive="screen" :x-gap="16"
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
        :disabled="disabled"
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
        :disabled="disabled"
        placeholder="搜索并选择成员" /></NFormItemGi
  ></NGrid>
</template>
