<script setup lang="ts">
import type { Comment } from "@/api";

const emit = defineEmits<{
  delete: [comment: Comment];
  reaction: [comment: Comment, emoji: string];
}>();
const props = defineProps<{ comment: Comment; busy: boolean }>();
const segments = computed(() => linkify(props.comment.body));
</script>

<template>
  <NListItem class="comment-row">
    <article class="comment">
      <div class="comment-header">
        <UserIdentity show-position :size="32" :user="comment.author" />
        <NText depth="3">
          <time
            :datetime="comment.createdAt"
            :title="formatDateTime(comment.createdAt)"
          >
            {{ timeAgo(comment.createdAt) }}
          </time>
        </NText>
        <NButton
          v-if="comment.canDelete"
          aria-label="删除进展"
          class="delete-comment"
          :disabled="busy"
          quaternary
          size="small"
          @click="emit('delete', comment)"
        >
          <template #icon><Trash2 :size="14" /></template>
          删除
        </NButton>
      </div>
      <div class="comment-content">
        <div v-if="comment.body" class="comment-body">
          <template v-for="(segment, index) in segments" :key="index">
            <a
              v-if="segment.type === 'link'"
              :href="segment.href"
              rel="noopener noreferrer"
              target="_blank"
            >
              {{ segment.value }}
            </a>
            <span v-else>{{ segment.value }}</span>
          </template>
        </div>
        <ImageGallery v-if="comment.attachments.length > 0">
          <div class="comment-images">
            <PreviewImage
              v-for="(image, index) in comment.attachments"
              :key="image.id"
              :alt="`进展图片 ${index + 1}`"
              class="comment-image"
              :height="88"
              lazy
              :src="image.url"
              :width="88"
            />
          </div>
        </ImageGallery>
        <CommentReactions
          :busy
          :comment-id="comment.id"
          :reactions="comment.reactions"
          @reaction="(emoji) => emit('reaction', comment, emoji)"
        />
      </div>
    </article>
  </NListItem>
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
