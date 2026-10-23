<script setup lang="ts">
import { computed, defineAsyncComponent, ref, useId } from 'vue'
import { ChevronDown, SmilePlus, Trash2 } from '@lucide/vue'
import ReactionUsers from '@/components/ReactionUsers.vue'
import UserAvatar from '@/components/UserAvatar.vue'
import type { Comment } from '@/lib/api'
import { formatDateTime, linkify, reactionEmoji, reactionKeys, timeAgo } from '@/lib/utils'

const ImageViewer = defineAsyncComponent(() => import('@/components/ImageViewer.vue'))

const props = defineProps<{
  comment: Comment
}>()

const emit = defineEmits<{
  delete: [comment: Comment]
  reaction: [comment: Comment, emoji: string]
}>()

const picker = ref<HTMLDetailsElement | null>(null)
const peopleId = useId()
const selectedReaction = ref<string | null>(null)
let peopleTrigger: HTMLButtonElement | null = null
const imageIndex = ref<number | null>(null)
const images = computed(() => props.comment.attachments.map((attachment, index) => ({
  src: attachment.url,
  alt: `评论图片 ${index + 1}`,
})))
const segments = computed(() => linkify(props.comment.body))
const activeReactions = computed(() => props.comment.reactions.filter((reaction) => reaction.count > 0))
const primaryPosition = computed(() => props.comment.author.positions[0]?.label ?? '')

function react(emoji: string) {
  selectedReaction.value = null
  emit('reaction', props.comment, emoji)
  if (picker.value) picker.value.open = false
}

function togglePeople(emoji: string, event: MouseEvent) {
  peopleTrigger = event.currentTarget as HTMLButtonElement
  selectedReaction.value = selectedReaction.value === emoji ? null : emoji
}

function closePeople() {
  selectedReaction.value = null
  peopleTrigger?.focus()
}
</script>

<template>
  <article class="group border-b border-line px-3 py-3 last:border-b-0">
    <div class="flex items-start gap-2.5">
      <RouterLink :to="{ name: 'user', params: { subject: comment.author.subject } }" class="mt-0.5">
        <UserAvatar :name="comment.author.userName" :avatar="comment.author.avatar" size-class="h-7 w-7 text-xs" />
      </RouterLink>

      <div class="min-w-0 flex-1">
        <div class="flex flex-wrap items-center gap-x-2 gap-y-0.5">
          <RouterLink
            :to="{ name: 'user', params: { subject: comment.author.subject } }"
            class="min-w-0 break-words text-[13px] font-semibold text-ink hover:text-accent"
          >
            {{ comment.author.userName }}
          </RouterLink>
          <span v-if="primaryPosition" class="rounded bg-surface-3 px-1 py-0.5 text-xs text-muted">{{ primaryPosition }}</span>
          <time class="text-xs text-faint" :datetime="comment.createdAt" :title="formatDateTime(comment.createdAt)">
            {{ timeAgo(comment.createdAt) }}
          </time>
          <button
            v-if="comment.canDelete"
            type="button"
            class="btn btn-ghost btn-xs ml-auto text-muted hover:bg-danger-soft hover:text-danger"
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
          <button
            v-for="(attachment, index) in comment.attachments"
            :key="attachment.id"
            type="button"
            :aria-label="`查看评论图片 ${index + 1}`"
            aria-haspopup="dialog"
            class="block cursor-zoom-in overflow-hidden rounded-md border border-line hover:border-accent"
            @click="imageIndex = index"
          >
            <img
              :src="attachment.url"
              alt=""
              class="h-20 w-20 object-cover"
              loading="lazy"
            >
          </button>
        </div>

        <div class="relative mt-2 flex flex-wrap items-center gap-1">
          <div
            v-for="reaction in activeReactions"
            :key="reaction.emoji"
            class="inline-flex items-center rounded-full border text-xs"
            :class="reaction.reactedByMe
              ? 'border-accent/30 bg-accent-soft text-accent'
              : 'border-line bg-surface-2 text-ink-soft'"
          >
            <button
              type="button"
              class="flex h-8 min-w-8 items-center justify-center rounded-l-full px-2 hover:bg-accent/10"
              :title="`${reaction.reactedByMe ? '取消' : ''}${reactionEmoji[reaction.emoji]?.label}`"
              :aria-label="reactionEmoji[reaction.emoji]?.label"
              :aria-pressed="reaction.reactedByMe"
              @click="react(reaction.emoji)"
            >
              {{ reactionEmoji[reaction.emoji]?.symbol ?? reaction.emoji }}
            </button>
            <button
              type="button"
              class="flex h-8 min-w-9 items-center justify-center gap-0.5 rounded-r-full border-l border-current/15 pr-2 pl-1.5 hover:bg-accent/10"
              :class="selectedReaction === reaction.emoji && 'bg-accent/10'"
              :title="`查看${reactionEmoji[reaction.emoji]?.label}的人员`"
              :aria-label="`查看${reactionEmoji[reaction.emoji]?.label}的人员，${reaction.count} 人`"
              :aria-expanded="selectedReaction === reaction.emoji"
              :aria-controls="peopleId"
              @click="togglePeople(reaction.emoji, $event)"
              @keydown.esc.stop="selectedReaction && closePeople()"
            >
              <span>{{ reaction.count }}</span>
              <ChevronDown class="h-3 w-3" :class="selectedReaction === reaction.emoji && 'rotate-180'" />
            </button>
          </div>

          <details ref="picker" @keydown.esc.stop="picker && (picker.open = false)" @focusout="!picker?.contains($event.relatedTarget as Node) && picker && (picker.open = false)">
            <summary
              class="flex h-8 w-8 cursor-pointer list-none items-center justify-center rounded-full border border-line bg-surface text-faint hover:border-accent/30 hover:text-accent"
              title="添加表情"
              aria-label="添加表情"
            >
              <SmilePlus class="h-3.5 w-3.5" />
            </summary>
            <div class="absolute bottom-8 left-0 z-20 grid w-40 grid-cols-4 gap-1 rounded-md border border-line bg-surface p-1.5 shadow-md">
              <button
                v-for="emoji in reactionKeys"
                :key="emoji"
                type="button"
                class="flex h-7 w-7 items-center justify-center rounded text-sm hover:bg-surface-2"
                :title="reactionEmoji[emoji]?.label"
                :aria-label="reactionEmoji[emoji]?.label"
                @click="react(emoji)"
              >
                {{ reactionEmoji[emoji]?.symbol }}
              </button>
            </div>
          </details>
        </div>
        <ReactionUsers v-if="selectedReaction" :id="peopleId" :key="selectedReaction" :comment-id="comment.id" :emoji="selectedReaction" @close="closePeople" />
      </div>
    </div>
    <ImageViewer v-if="imageIndex !== null" :images="images" :index="imageIndex" @close="imageIndex = null" />
  </article>
</template>
