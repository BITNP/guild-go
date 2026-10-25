<script setup lang="ts">
import { computed } from 'vue'
import { NButton, NListItem, NText } from 'naive-ui'
import { Trash2 } from '@lucide/vue'
import PreviewImage from '@/components/PreviewImage.vue'
import ImageGallery from '@/components/ImageGallery.vue'
import UserIdentity from '@/components/UserIdentity.vue'
import CommentReactions from '@/components/CommentReactions.vue'
import type { Comment } from '@/lib/api'
import { formatDateTime, linkify, timeAgo } from '@/lib/utils'
const props = defineProps<{ comment: Comment; busy: boolean }>(),
  emit = defineEmits<{
    delete: [comment: Comment]
    reaction: [comment: Comment, emoji: string]
  }>()
const segments = computed(() => linkify(props.comment.body))
</script>
<template>
  <NListItem class="comment-row"
    ><article class="comment">
      <div class="comment-header">
        <UserIdentity :user="comment.author" :size="32" show-position /><NText
          depth="3"
          ><time
            :datetime="comment.createdAt"
            :title="formatDateTime(comment.createdAt)"
            >{{ timeAgo(comment.createdAt) }}</time
          ></NText
        ><NButton
          v-if="comment.canDelete"
          size="small"
          quaternary
          class="delete-comment"
          aria-label="删除进展"
          :disabled="busy"
          @click="emit('delete', comment)"
          ><template #icon><Trash2 :size="14" /></template>删除</NButton
        >
      </div>
      <div class="comment-content">
        <div v-if="comment.body" class="comment-body">
          <template v-for="(segment, index) in segments" :key="index"
            ><a
              v-if="segment.type === 'link'"
              :href="segment.href"
              target="_blank"
              rel="noopener noreferrer"
              >{{ segment.value }}</a
            ><span v-else>{{ segment.value }}</span></template
          >
        </div>
        <ImageGallery v-if="comment.attachments.length"
          ><div class="comment-images">
            <PreviewImage
              v-for="(image, index) in comment.attachments"
              :key="image.id"
              :src="image.url"
              :alt="`进展图片 ${index + 1}`"
              :width="88"
              :height="88"
              lazy
              class="comment-image"
            /></div></ImageGallery
        ><CommentReactions
          :comment-id="comment.id"
          :reactions="comment.reactions"
          :busy="busy"
          @reaction="(emoji) => emit('reaction', comment, emoji)"
        />
      </div></article
  ></NListItem>
</template>
<style scoped>
.comment-row {
  padding: 8px 0;
}
.comment-row:first-child {
  padding-top: 0;
}
.comment-row:last-child {
  padding-bottom: 0;
}
.comment {
  min-width: 0;
}
.comment-header {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  font-size: var(--guild-meta-size);
}
.comment-header :deep(.user-name) {
  color: var(--guild-text);
  font-size: 14px;
  font-weight: 600;
}
.comment-content {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin: 6px 0 0 40px;
}
.delete-comment {
  margin-left: auto;
  color: var(--guild-muted);
}
.delete-comment:hover {
  color: var(--guild-danger);
}
.comment-body {
  white-space: pre-wrap;
  overflow-wrap: anywhere;
  line-height: 1.8;
}
.comment-body a {
  color: var(--guild-primary);
  text-decoration: underline;
  text-underline-offset: 3px;
}
.comment-images {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
.comment-image {
  border-radius: 8px;
  overflow: hidden;
}
</style>
