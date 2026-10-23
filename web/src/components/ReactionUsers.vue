<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { Loader2, X } from '@lucide/vue'
import UserAvatar from '@/components/UserAvatar.vue'
import { fetchReactionUsers, type UserSummary } from '@/lib/api'
import { errorMessage, reactionEmoji } from '@/lib/utils'

const props = defineProps<{
  commentId: number
  emoji: string
}>()

const emit = defineEmits<{ close: [] }>()
const people = ref<UserSummary[]>([])
const loading = ref(true)
const error = ref<string | null>(null)

async function load() {
  loading.value = true
  error.value = null
  try {
    people.value = await fetchReactionUsers(props.commentId, props.emoji)
  } catch (err) {
    error.value = errorMessage(err, '无法加载人员列表')
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<template>
  <section class="mt-2 max-w-sm rounded-md border border-line bg-surface-2" :aria-label="`${reactionEmoji[emoji]?.label}的人员`" @keydown.esc.stop="emit('close')">
    <div class="flex items-center justify-between gap-2 px-3 py-1.5">
      <h4 class="flex items-center gap-1.5 text-xs font-medium text-ink-soft">
        <span aria-hidden="true">{{ reactionEmoji[emoji]?.symbol }}</span>
        {{ reactionEmoji[emoji]?.label }}
        <span v-if="!loading && !error" class="font-normal text-muted">{{ people.length }} 人</span>
      </h4>
      <button type="button" class="icon-btn" aria-label="收起人员列表" @click="emit('close')">
        <X class="h-3.5 w-3.5" />
      </button>
    </div>
    <p v-if="loading" class="flex items-center gap-2 px-3 pb-3 text-xs text-muted" role="status">
      <Loader2 class="h-3.5 w-3.5 animate-spin" /> 正在加载…
    </p>
    <div v-else-if="error" class="px-3 pb-3">
      <p role="alert" class="text-xs text-danger">{{ error }}</p>
      <button type="button" class="btn btn-xs mt-2" @click="load">重新加载</button>
    </div>
    <p v-else-if="people.length === 0" class="px-3 pb-3 text-xs text-muted">还没有人使用这个表情</p>
    <ul v-else class="grid max-h-60 gap-x-1 overflow-y-auto px-1 pb-1 sm:grid-cols-2">
      <li v-for="person in people" :key="person.subject" class="min-w-0">
        <RouterLink :to="{ name: 'user', params: { subject: person.subject } }" class="flex min-h-11 items-center gap-2 rounded px-2 py-1.5 text-[13px] text-ink-soft hover:bg-surface-3 hover:text-accent">
          <UserAvatar :name="person.userName" :avatar="person.avatar" size-class="h-6 w-6 text-xs" />
          <span class="min-w-0 break-words">{{ person.userName }}</span>
        </RouterLink>
      </li>
    </ul>
  </section>
</template>
