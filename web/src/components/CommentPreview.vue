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
      <NSpace
        v-if="comment.reactionSummary.length"
        :size="8"
        class="preview-reactions"
        ><NText
          v-for="reaction in comment.reactionSummary"
          :key="reaction.emoji"
          depth="3"
          >{{ reactionEmoji[reaction.emoji]?.symbol }}
          {{ reaction.count }}</NText
        ></NSpace
      >
    </div>
    <p class="preview-body">{{ comment.bodyPreview || '图片评论' }}</p>
  </div>
</template>
<style scoped>
.comment-preview {
  border-left: 2px solid var(--guild-border);
  padding-left: 10px;
  font-size: 12px;
}
.preview-byline {
  display: flex;
  align-items: center;
  gap: 6px 8px;
  flex-wrap: wrap;
}
.preview-body {
  margin: 4px 0 0;
  line-height: 1.65;
  overflow-wrap: anywhere;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
.preview-reactions {
  margin-left: auto;
}
</style>
