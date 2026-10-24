<script setup lang="ts">
import {
  NDescriptions,
  NDescriptionsItem,
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
    <section class="detail-group" aria-label="状态与时间">
      <h3>状态与时间</h3>
      <NDescriptions
        size="small"
        :column="1"
        label-placement="left"
        separator=""
        label-class="detail-label"
      >
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
          ><UserIdentity :user="node.createdBy" :size="22" /></NDescriptionsItem
      ></NDescriptions>
    </section>
    <section class="detail-group" aria-label="参与人员">
      <h3>参与人员</h3>
      <div class="people-group">
        <h4>直接负责人</h4>
        <NList :show-divider="false" class="people-list"
          ><NListItem v-for="person in node.directLeaders" :key="person.subject"
            ><UserIdentity :user="person" /></NListItem></NList
        ><NText
          v-if="node.inheritedLeaderCount"
          depth="3"
          class="inherited-note"
          >另有
          {{ node.inheritedLeaderCount }} 位上级节点负责人可管理此节点。</NText
        >
      </div>
      <div class="people-group">
        <h4>直接成员</h4>
        <NList :show-divider="false" class="people-list"
          ><NListItem v-for="person in node.directMembers" :key="person.subject"
            ><UserIdentity :user="person" /></NListItem></NList
        ><NText v-if="!node.directMembers.length" depth="3">暂无直接成员</NText>
      </div>
    </section>
    <section class="detail-group" aria-label="包含下级节点">
      <h3>包含下级节点</h3>
      <NDescriptions
        size="small"
        :column="1"
        label-placement="left"
        separator=""
        label-class="statistics-label"
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
  </section>
</template>
<style scoped>
.node-details {
  padding: 16px;
  font-size: 13px;
  overflow-wrap: anywhere;
}
.detail-group + .detail-group {
  margin-top: 16px;
  padding-top: 16px;
  border-top: 1px solid var(--guild-border);
}
.detail-group h3 {
  margin: 0 0 12px;
  font-size: 14px;
  line-height: 22px;
  font-weight: 600;
}
.detail-group :deep(.detail-label) {
  display: inline-block;
  width: 76px;
  color: var(--guild-muted);
}
.detail-group :deep(.statistics-label) {
  display: inline-block;
  width: 132px;
  color: var(--guild-muted);
}
.people-list :deep(.n-list-item) {
  padding: 4px 0;
}
.people-list {
  margin: -4px 0 6px;
  font-size: 14px;
}
.people-group h4 {
  margin: 0 0 8px;
  font-size: var(--guild-meta-size);
  font-weight: 500;
  color: var(--guild-muted);
}
.people-group + .people-group {
  margin-top: 16px;
}
.inherited-note {
  display: block;
  margin-top: 8px;
  font-size: var(--guild-meta-size);
  line-height: 1.6;
}
</style>
