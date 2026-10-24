<script setup lang="ts">
import {
  NDescriptions,
  NDescriptionsItem,
  NDivider,
  NH3,
  NList,
  NListItem,
  NText,
} from 'naive-ui'
import UserIdentity from '@/components/UserIdentity.vue'
import NodeStatusTag from '@/components/NodeStatusTag.vue'
import type { NodeDetail } from '@/lib/api'
import { formatDateTime, timeAgo } from '@/lib/utils'
defineProps<{ node: NodeDetail }>()
</script>
<template>
  <section class="node-details">
    <NDescriptions title="状态与时间" :column="1" label-placement="left">
      <NDescriptionsItem label="状态"
        ><NodeStatusTag :due-at="node.dueAt" :completed-at="node.completedAt"
      /></NDescriptionsItem>
      <NDescriptionsItem label="预期完成">{{
        node.dueAt ? formatDateTime(node.dueAt) : '未设置'
      }}</NDescriptionsItem>
      <NDescriptionsItem v-if="node.completedAt" label="实际完成">{{
        formatDateTime(node.completedAt)
      }}</NDescriptionsItem>
      <NDescriptionsItem label="创建时间">{{
        formatDateTime(node.createdAt)
      }}</NDescriptionsItem>
      <NDescriptionsItem label="最近更新"
        ><time :title="formatDateTime(node.updatedAt)">{{
          timeAgo(node.updatedAt)
        }}</time></NDescriptionsItem
      >
      <NDescriptionsItem label="创建者"
        ><UserIdentity :user="node.createdBy" :size="22"
      /></NDescriptionsItem> </NDescriptions
    ><NDivider /> <NH3>直接负责人</NH3
    ><NList
      ><NListItem v-for="person in node.directLeaders" :key="person.subject"
        ><UserIdentity :user="person" /></NListItem></NList
    ><NText v-if="node.inheritedLeaderCount" depth="3"
      >另有
      {{ node.inheritedLeaderCount }} 位上级节点负责人可管理此节点。</NText
    >
    <NH3>直接成员</NH3
    ><NList
      ><NListItem v-for="person in node.directMembers" :key="person.subject"
        ><UserIdentity :user="person" /></NListItem></NList
    ><NText v-if="!node.directMembers.length" depth="3">暂无直接成员</NText
    ><NDivider />
    <NDescriptions title="包含下级节点" :column="1" label-placement="left"
      ><NDescriptionsItem label="成员（含负责人）">{{
        node.effectiveMemberCount
      }}</NDescriptionsItem
      ><NDescriptionsItem label="节点">{{
        node.subtreeNodeCount
      }}</NDescriptionsItem
      ><NDescriptionsItem label="评论">{{
        node.subtreeCommentCount
      }}</NDescriptionsItem
      ><NDescriptionsItem label="图片">{{
        node.subtreeAttachmentCount
      }}</NDescriptionsItem></NDescriptions
    >
  </section>
</template>
<style scoped>
.node-details {
  padding: 16px;
  overflow-wrap: anywhere;
}
</style>
