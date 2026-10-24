<script setup lang="ts">
import { NSpace, NText } from 'naive-ui'
import UserIdentity from '@/components/UserIdentity.vue'
import type { CommentPreview } from '@/lib/api'
import { reactionEmoji, timeAgo } from '@/lib/utils'
defineProps<{
  comment: Omit<CommentPreview, 'attachmentCount'> & {
    attachmentCount?: number
  }
}>()
</script>
<template>
  <div class="comment-preview">
    <div class="preview-byline">
      <UserIdentity :user="comment.author" :size="22" /><NText depth="3">{{
        timeAgo(comment.createdAt)
      }}</NText
      ><NText v-if="comment.attachmentCount" depth="3"
        >{{ comment.attachmentCount }} 张图片</NText
      >
    </div>
    <p class="preview-body">{{ comment.bodyPreview || '图片评论' }}</p>
    <NSpace v-if="comment.reactionSummary.length" :size="8"
      ><NText
        v-for="reaction in comment.reactionSummary"
        :key="reaction.emoji"
        depth="3"
        >{{ reactionEmoji[reaction.emoji]?.symbol }} {{ reaction.count }}</NText
      ></NSpace
    >
  </div>
</template>
<style scoped>
.comment-preview {
  border-left: 2px solid var(--guild-border);
  padding-left: 12px;
  font-size: 13px;
}
.preview-byline {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}
.preview-body {
  margin: 6px 0;
  overflow-wrap: anywhere;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
</style>
