<script setup lang="ts">
import { NListItem, NSpace, NTag, NText, NThing } from 'naive-ui'
import NodeStatusTag from '@/components/NodeStatusTag.vue'
import CommentPreview from '@/components/CommentPreview.vue'
import UserIdentity from '@/components/UserIdentity.vue'
import type { ChildNode, UserSummary } from '@/lib/api'
import { formatDateTime } from '@/lib/utils'
defineProps<{ node: ChildNode; leaders?: UserSummary[] }>()
</script>
<template>
  <NListItem
    ><NThing>
      <template #header
        ><RouterLink
          :to="{ name: 'node', params: { id: node.id } }"
          class="node-link"
          >{{ node.title }}</RouterLink
        ></template
      >
      <template #header-extra
        ><NodeStatusTag :due-at="node.dueAt" :completed-at="node.completedAt"
      /></template>
      <template #description
        ><NSpace :size="8"
          ><NTag v-for="tag in node.tags" :key="tag" size="small">{{
            tag
          }}</NTag
          ><span v-if="node.dueAt"
            >预期完成 {{ formatDateTime(node.dueAt) }}</span
          ></NSpace
        ></template
      >
      <NSpace vertical :size="12"
        ><NSpace v-if="leaders" :size="12"
          ><UserIdentity
            v-for="leader in leaders"
            :key="leader.subject"
            :user="leader"
            :size="22"
          /><NText v-if="!leaders.length" depth="3">暂无负责人</NText></NSpace
        ><NText depth="3"
          >{{ node.childCount }} 个子节点 ·
          {{ node.commentCount }} 条评论</NText
        ><CommentPreview
          v-if="node.latestComment"
          :comment="node.latestComment"
      /></NSpace> </NThing
  ></NListItem>
</template>
<style scoped>
.node-link {
  color: inherit;
  text-decoration: none;
  overflow-wrap: anywhere;
}
.node-link:hover {
  color: var(--guild-primary);
}
.node-link:focus-visible {
  outline: 2px solid var(--guild-primary);
  outline-offset: 3px;
}
</style>
