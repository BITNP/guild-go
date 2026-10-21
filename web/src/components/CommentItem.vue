<script setup lang="ts">
import { computed, ref } from 'vue'
import { MessageSquare, SmilePlus, Trash2 } from '@lucide/vue'
import UserAvatar from '@/components/UserAvatar.vue'
import type { Comment } from '@/lib/api'
import { formatDateTime, linkify, reactionEmoji, reactionKeys, timeAgo } from '@/lib/utils'

const props = defineProps<{
  comment: Comment
}>()

const emit = defineEmits<{
  delete: [comment: Comment]
  reaction: [comment: Comment, emoji: string]
}>()

const picker = ref<HTMLDetailsElement | null>(null)
const segments = computed(() => linkify(props.comment.body))
const activeReactions = computed(() => props.comment.reactions.filter((reaction) => reaction.count > 0))
const primaryPosition = computed(() => props.comment.author.positions[0]?.label ?? '')

function react(emoji: string) {
  emit('reaction', props.comment, emoji)
  if (picker.value) picker.value.open = false
}
</script>

<template>
  <article class="group border-b border-line px-3 py-3 last:border-b-0">
    <div class="flex items-start gap-2.5">
      <RouterLink :to="{ name: 'user', params: { subject: comment.author.subject } }" class="mt-0.5">
        <UserAvatar :name="comment.author.userName" :avatar="comment.author.avatar" size-class="h-7 w-7 text-[10px]" />
      </RouterLink>

      <div class="min-w-0 flex-1">
        <div class="flex flex-wrap items-center gap-x-2 gap-y-0.5">
          <RouterLink
            :to="{ name: 'user', params: { subject: comment.author.subject } }"
            class="text-[12px] font-semibold text-ink hover:text-accent"
          >
            {{ comment.author.userName }}
          </RouterLink>
          <span v-if="primaryPosition" class="rounded bg-surface-3 px-1 py-0.5 text-[9px] text-muted">{{ primaryPosition }}</span>
          <time class="text-[10px] text-faint" :datetime="comment.createdAt" :title="formatDateTime(comment.createdAt)">
            {{ timeAgo(comment.createdAt) }}
          </time>
          <button
            v-if="comment.canDelete"
            type="button"
            class="ml-auto inline-flex items-center gap-1 rounded px-1 py-0.5 text-[10px] text-faint opacity-0 transition-opacity hover:bg-danger-soft hover:text-danger group-hover:opacity-100 focus:opacity-100"
            @click="emit('delete', comment)"
          >
            <Trash2 class="h-3 w-3" /> 删除
          </button>
        </div>

        <div v-if="comment.body" class="mt-1 whitespace-pre-wrap break-words text-[13px] leading-6 text-ink-soft">
          <template v-for="(segment, index) in segments" :key="index">
            <a
              v-if="segment.type === 'link'"
              :href="segment.href"
              target="_blank"
              rel="noopener noreferrer"
              class="link break-all"
            >{{ segment.value }}</a>
            <span v-else>{{ segment.value }}</span>
          </template>
        </div>

        <div v-if="comment.attachments.length" class="mt-2 flex flex-wrap gap-2">
          <a
            v-for="attachment in comment.attachments"
            :key="attachment.id"
            :href="attachment.url"
            target="_blank"
            rel="noopener noreferrer"
            class="block overflow-hidden rounded-md border border-line"
          >
            <img
              :src="attachment.url"
              :alt="`评论图片 ${attachment.id}`"
              class="h-20 w-20 object-cover transition-transform hover:scale-105"
              loading="lazy"
            >
          </a>
        </div>

        <div class="mt-2 flex flex-wrap items-center gap-1">
          <button
            v-for="reaction in activeReactions"
            :key="reaction.emoji"
            type="button"
            class="inline-flex h-6 items-center gap-1 rounded-full border px-1.5 text-[11px] transition-colors"
            :class="reaction.reactedByMe
              ? 'border-accent/30 bg-accent-soft text-accent'
              : 'border-line bg-surface-2 text-ink-soft hover:border-accent/30'"
            :title="reactionEmoji[reaction.emoji]?.label"
            @click="emit('reaction', comment, reaction.emoji)"
          >
            <span>{{ reactionEmoji[reaction.emoji]?.symbol ?? reaction.emoji }}</span>
            <span>{{ reaction.count }}</span>
          </button>

          <details ref="picker" class="relative">
            <summary
              class="flex h-6 w-6 cursor-pointer list-none items-center justify-center rounded-full border border-line bg-surface text-faint hover:border-accent/30 hover:text-accent"
              title="添加表情"
            >
              <SmilePlus class="h-3.5 w-3.5" />
            </summary>
            <div class="absolute bottom-7 left-0 z-20 flex gap-1 rounded-md border border-line bg-surface p-1.5 shadow-lg">
              <button
                v-for="emoji in reactionKeys"
                :key="emoji"
                type="button"
                class="flex h-7 w-7 items-center justify-center rounded text-sm hover:bg-surface-2"
                :title="reactionEmoji[emoji]?.label"
                @click="react(emoji)"
              >
                {{ reactionEmoji[emoji]?.symbol }}
              </button>
            </div>
          </details>

          <span v-if="comment.attachments.length" class="ml-1 inline-flex items-center gap-0.5 text-[10px] text-faint">
            <MessageSquare class="h-2.5 w-2.5" /> {{ comment.attachments.length }} 张图
          </span>
        </div>
      </div>
    </div>
  </article>
</template>