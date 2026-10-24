<script setup lang="ts">
import { NAvatar, NTag } from 'naive-ui'
import type { UserSummary } from '@/lib/api'
withDefaults(
  defineProps<{
    user: UserSummary
    size?: number
    showName?: boolean
    showPosition?: boolean
  }>(),
  { size: 26, showName: true, showPosition: false },
)
</script>
<template>
  <span class="user-identity">
    <RouterLink
      :to="{ name: 'user', params: { subject: user.subject } }"
      class="user-link"
      :aria-label="user.userName"
    >
      <NAvatar
        round
        :size="size"
        :src="user.avatar ?? undefined"
        :img-props="{ alt: '' }"
        >{{ user.userName.slice(0, 1) }}</NAvatar
      >
      <span v-if="showName" class="user-name">{{ user.userName }}</span>
    </RouterLink>
    <NTag
      v-if="showPosition && user.positions[0]"
      size="small"
      :bordered="false"
      >{{ user.positions[0].label }}</NTag
    >
  </span>
</template>
<style scoped>
.user-identity,
.user-link {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}
.user-identity {
  flex-wrap: wrap;
}
.user-link {
  color: inherit;
  text-decoration: none;
}
.user-link:hover {
  color: var(--guild-primary);
}
.user-name {
  overflow-wrap: anywhere;
}
.user-link:focus-visible {
  outline: 2px solid var(--guild-primary);
  outline-offset: 3px;
}
</style>
