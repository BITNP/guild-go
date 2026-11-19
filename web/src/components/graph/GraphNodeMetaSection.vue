<script setup lang="ts">
import { CalendarClock } from "@lucide/vue";

import type { NodeDetail } from "@/lib/api";
import { formatDateTime, timeAgo } from "@/lib/utils";

defineProps<{ detail: NodeDetail }>();
</script>

<template>
  <section aria-labelledby="panel-meta-title" class="panel-section">
    <h3 id="panel-meta-title">节点信息</h3>
    <dl class="panel-meta-grid">
      <div>
        <dt>
          <CalendarClock :size="13" />
          预期完成
        </dt>
        <dd>{{ detail.dueAt ? formatDateTime(detail.dueAt) : "未设置" }}</dd>
      </div>
      <div v-if="detail.completedAt">
        <dt>实际完成</dt>
        <dd>{{ formatDateTime(detail.completedAt) }}</dd>
      </div>
      <div>
        <dt>创建时间</dt>
        <dd>{{ formatDateTime(detail.createdAt) }}</dd>
      </div>
      <div>
        <dt>最近更新</dt>
        <dd :title="formatDateTime(detail.updatedAt)">
          {{ timeAgo(detail.updatedAt) }}
        </dd>
      </div>
      <div>
        <dt>包含节点</dt>
        <dd>{{ detail.subtreeNodeCount }}</dd>
      </div>
      <div>
        <dt>有效成员</dt>
        <dd>{{ detail.effectiveMemberCount }}</dd>
      </div>
      <div>
        <dt>图片附件</dt>
        <dd>{{ detail.subtreeAttachmentCount }}</dd>
      </div>
    </dl>
  </section>
</template>

<style scoped>
.panel-section {
  padding: 19px 24px;
  border-bottom: 1px solid var(--graph-panel-divider-subtle);
}

.panel-section h3 {
  margin: 0 0 13px;
  color: var(--graph-section-title-color);
  font-size: 15px;
  font-weight: 700;
}

.panel-meta-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 13px 18px;
  margin: 0;
}

.panel-meta-grid div {
  min-width: 0;
}

.panel-meta-grid dt {
  display: flex;
  align-items: center;
  gap: 5px;
  margin-bottom: 4px;
  color: var(--graph-meta-label);
  font-size: 13px;
}

.panel-meta-grid dd {
  margin: 0;
  color: var(--graph-body-color);
  font-size: 15px;
  font-variant-numeric: tabular-nums;
  overflow-wrap: anywhere;
}

@media (max-width: 640px) {
  .panel-section {
    padding-right: 18px;
    padding-left: 18px;
  }

  .panel-meta-grid {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
