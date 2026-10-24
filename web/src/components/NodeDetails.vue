<script setup lang="ts">
import { NText } from 'naive-ui'
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
      <dl class="detail-fields">
        <dt>状态</dt>
        <dd>
          <NodeStatusTag
            :due-at="node.dueAt"
            :completed-at="node.completedAt"
          />
        </dd>
        <dt>预期完成</dt>
        <dd>{{ node.dueAt ? formatDateTime(node.dueAt) : '未设置' }}</dd>
        <template v-if="node.completedAt">
          <dt>实际完成</dt>
          <dd>{{ formatDateTime(node.completedAt) }}</dd>
        </template>
        <dt>创建时间</dt>
        <dd>{{ formatDateTime(node.createdAt) }}</dd>
        <dt>最近更新</dt>
        <dd>
          <time :title="formatDateTime(node.updatedAt)">{{
            timeAgo(node.updatedAt)
          }}</time>
        </dd>
        <dt>创建者</dt>
        <dd>
          <UserIdentity
            class="detail-person"
            :user="node.createdBy"
            :size="22"
          />
        </dd>
      </dl>
    </section>
    <section class="detail-group" aria-label="参与人员">
      <h3>参与人员</h3>
      <div class="people-group">
        <h4>直接负责人</h4>
        <ul class="people-list">
          <li v-for="person in node.directLeaders" :key="person.subject">
            <UserIdentity class="detail-person" :user="person" />
          </li>
        </ul>
        <NText v-if="node.inheritedLeaderCount" depth="3" class="inherited-note"
          >另有
          {{ node.inheritedLeaderCount }} 位上级节点负责人可管理此节点。</NText
        >
      </div>
      <div class="people-group">
        <h4>直接成员</h4>
        <ul class="people-list">
          <li v-for="person in node.directMembers" :key="person.subject">
            <UserIdentity class="detail-person" :user="person" />
          </li>
        </ul>
        <NText v-if="!node.directMembers.length" depth="3">暂无直接成员</NText>
      </div>
    </section>
    <section class="detail-group" aria-label="包含下级节点">
      <h3>包含下级节点</h3>
      <dl class="detail-fields">
        <dt>成员（含负责人）</dt>
        <dd>{{ node.effectiveMemberCount }}</dd>
        <dt>节点</dt>
        <dd>{{ node.subtreeNodeCount }}</dd>
        <dt>评论</dt>
        <dd>{{ node.subtreeCommentCount }}</dd>
        <dt>图片</dt>
        <dd>{{ node.subtreeAttachmentCount }}</dd>
      </dl>
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
.detail-fields {
  display: grid;
  grid-template-columns: max-content minmax(0, 1fr);
  align-items: center;
  gap: 8px 16px;
  margin: 0;
}
.detail-fields dt {
  color: var(--guild-muted);
}
.detail-fields dd {
  min-width: 0;
  margin: 0;
  text-align: right;
  font-variant-numeric: tabular-nums;
}
.people-list {
  display: grid;
  gap: 8px;
  margin: 0 0 6px;
  padding: 0;
  list-style: none;
  font-size: 14px;
}
.detail-person {
  max-width: 100%;
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
