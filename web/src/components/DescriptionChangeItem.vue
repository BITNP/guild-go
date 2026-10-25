<script setup lang="ts">
import { computed, ref } from 'vue'
import { NButton, NCollapseTransition, NListItem, NText } from 'naive-ui'
import UserIdentity from '@/components/UserIdentity.vue'
import type { DescriptionChange } from '@/lib/api'
import { formatDateTime, timeAgo } from '@/lib/utils'

const props = defineProps<{ change: DescriptionChange }>()
const expanded = ref(false)
const action = computed(() =>
  !props.change.before
    ? '添加了描述'
    : !props.change.after
      ? '清空了描述'
      : '修改了描述',
)
</script>

<template>
  <NListItem class="description-row">
    <article class="description-change">
      <div class="change-header">
        <UserIdentity :user="change.author" :size="32" />
        <span>{{ action }}</span>
        <NText depth="3">
          <time :datetime="change.createdAt" :title="formatDateTime(change.createdAt)">
            {{ timeAgo(change.createdAt) }}
          </time>
        </NText>
        <NButton
          text
          size="small"
          class="change-toggle"
          :aria-expanded="expanded"
          :aria-controls="`description-change-${change.id}`"
          @click="expanded = !expanded"
        >{{ expanded ? '收起' : '查看变更' }}</NButton>
      </div>
      <NCollapseTransition :show="expanded">
        <div
          :id="`description-change-${change.id}`"
          class="description-comparison"
          :class="{ 'is-creation': !change.before }"
        >
          <section v-if="change.before" class="description-version">
            <h3>修改前</h3>
            <p>{{ change.before }}</p>
          </section>
          <section class="description-version">
            <h3>{{ change.before ? '修改后' : '描述内容' }}</h3>
            <p v-if="change.after">{{ change.after }}</p>
            <NText v-else depth="3">描述已清空</NText>
          </section>
        </div>
      </NCollapseTransition>
    </article>
  </NListItem>
</template>

<style scoped>
.description-row {
  padding: 8px 0;
}
.description-row:first-child {
  padding-top: 0;
}
.description-row:last-child {
  padding-bottom: 0;
}
.description-change {
  min-width: 0;
}
.change-header {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px 8px;
  font-size: var(--guild-meta-size);
  color: var(--guild-muted);
}
.change-header :deep(.user-name) {
  color: var(--guild-text);
  font-size: 14px;
  font-weight: 600;
}
.change-toggle {
  margin-left: auto;
  color: var(--guild-primary);
}
.description-comparison {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
  margin: 10px 0 0 40px;
}
.description-comparison.is-creation {
  grid-template-columns: minmax(0, 1fr);
}
.description-version {
  min-width: 0;
  padding: 10px 12px;
  background: var(--guild-surface-subtle);
  border-left: 2px solid var(--guild-border);
}
.description-version h3 {
  margin: 0 0 6px;
  color: var(--guild-muted);
  font-size: var(--guild-meta-size);
  font-weight: 500;
}
.description-version p {
  margin: 0;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
  max-height: 360px;
  overflow: auto;
  line-height: 1.8;
}
@media (max-width: 639px) {
  .description-comparison {
    grid-template-columns: minmax(0, 1fr);
    margin-left: 0;
  }
}
</style>
