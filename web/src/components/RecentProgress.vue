<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { NAlert, NButton, NCard, NEmpty, NRadioButton, NRadioGroup, NSpin } from 'naive-ui'
import ActivityPreview from '@/components/ActivityPreview.vue'
import SectionHeading from '@/components/SectionHeading.vue'
import type { ActivityScope, RecentActivity } from '@/lib/api'

const props = defineProps<{
  activities: RecentActivity[]
  rootIds: ReadonlySet<number>
  loading: boolean
  error: string | null
}>()
const scope = defineModel<ActivityScope>('scope', { required: true })
const emit = defineEmits<{ retry: [] }>()

interface ProgressEntry {
  activity: RecentActivity
  occurredAt: Date
}

interface ProgressGroup {
  nodeId: number
  latest: ProgressEntry
  earlier: ProgressEntry[]
}

const groupsPerPage = 5
const visibleGroupCount = ref(groupsPerPage)
const shortDate = new Intl.DateTimeFormat('zh-CN', { month: 'long', day: 'numeric' })
const fullDate = new Intl.DateTimeFormat('zh-CN', {
  year: 'numeric',
  month: 'long',
  day: 'numeric',
})

const days = computed(() => {
  const entries = props.activities.flatMap((activity): ProgressEntry[] => {
    const timestamp = activity.kind === 'node_updated' ? activity.completedAt : activity.createdAt
    if (!timestamp) return []
    const occurredAt = new Date(timestamp)
    return Number.isNaN(occurredAt.getTime()) ? [] : [{ activity, occurredAt }]
  })
  entries.sort((a, b) => b.occurredAt.getTime() - a.occurredAt.getTime())

  const today = new Date()
  const yesterday = new Date(today.getFullYear(), today.getMonth(), today.getDate() - 1)
  const todayKey = today.toDateString()
  const yesterdayKey = yesterday.toDateString()
  const byDay = new Map<string, { label: string; nodes: Map<number, ProgressGroup> }>()

  for (const entry of entries) {
    const { activity, occurredAt } = entry
    const key = occurredAt.toDateString()
    let day = byDay.get(key)
    if (!day) {
      const formatter = occurredAt.getFullYear() === today.getFullYear() ? shortDate : fullDate
      const label = key === todayKey ? '今天' : key === yesterdayKey ? '昨天' : formatter.format(occurredAt)
      day = { label, nodes: new Map() }
      byDay.set(key, day)
    }
    const { nodeId } = activity
    const group = day.nodes.get(nodeId)
    if (group) group.earlier.push(entry)
    else day.nodes.set(nodeId, { nodeId, latest: entry, earlier: [] })
  }
  return Array.from(byDay, ([key, day]) => ({
    key,
    label: day.label,
    groups: [...day.nodes.values()],
  }))
})

const visibleDays = computed(() => {
  let remaining = visibleGroupCount.value
  const visible = []
  for (const day of days.value) {
    if (remaining <= 0) break
    const groups = day.groups.slice(0, remaining)
    visible.push({ ...day, groups })
    remaining -= groups.length
  }
  return visible
})
const hasMore = computed(
  () => days.value.reduce((total, day) => total + day.groups.length, 0) > visibleGroupCount.value,
)

watch(scope, () => {
  visibleGroupCount.value = groupsPerPage
})
</script>

<template>
  <section class="recent-progress" aria-label="最近进展">
    <SectionHeading title="最近进展">
      <NRadioGroup v-model:value="scope" size="small" aria-label="动态范围">
        <NRadioButton value="related">与我有关</NRadioButton>
        <NRadioButton value="all">全部</NRadioButton>
      </NRadioGroup>
    </SectionHeading>
    <NCard size="small" :aria-busy="loading">
      <div v-if="loading" class="progress-placeholder" role="status" aria-label="正在加载进展">
        <NSpin size="small" />
      </div>
      <NAlert v-else-if="error" type="error">
        {{ error }}
        <NButton text @click="emit('retry')">重试</NButton>
      </NAlert>
      <NEmpty
        v-else-if="!days.length"
        class="progress-placeholder"
        size="small"
        :description="scope === 'related' ? '暂无与我有关的进展' : '暂无进展'"
      />
      <template v-else>
        <section v-for="day in visibleDays" :key="day.key" class="progress-day">
          <h3>{{ day.label }}</h3>
          <ul class="progress-groups">
            <li v-for="{ nodeId, latest, earlier } in day.groups" :key="nodeId" class="progress-group">
              <div class="progress-node">
                <RouterLink :to="{ name: 'node', params: { id: nodeId } }">
                  {{ latest.activity.nodeTitle }}
                </RouterLink>
                <span v-if="!rootIds.has(nodeId)">{{ latest.activity.rootTitle }}</span>
              </div>
              <ActivityPreview :activity="latest.activity" :occurred-at="latest.occurredAt" />
              <details v-if="earlier.length" class="progress-more">
                <summary>另有 {{ earlier.length }} 条动态</summary>
                <ul class="earlier-entries">
                  <li v-for="entry in earlier" :key="`${entry.activity.kind}-${entry.activity.id}`">
                    <ActivityPreview :activity="entry.activity" :occurred-at="entry.occurredAt" />
                  </li>
                </ul>
              </details>
            </li>
          </ul>
        </section>
        <div v-if="hasMore" class="progress-footer">
          <NButton text type="primary" @click="visibleGroupCount += groupsPerPage">
            查看更多进展
          </NButton>
        </div>
      </template>
    </NCard>
  </section>
</template>

<style scoped>
.recent-progress {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.progress-placeholder {
  padding: 20px 0;
  text-align: center;
}
.progress-day + .progress-day {
  margin-top: 16px;
}
.progress-day h3 {
  display: flex;
  align-items: center;
  gap: 12px;
  margin: 0;
  color: var(--guild-muted);
  font-size: var(--guild-meta-size);
  font-weight: 600;
}
.progress-day h3::after {
  flex: 1;
  border-top: 1px solid var(--guild-border);
  content: '';
}
.progress-groups,
.earlier-entries {
  margin: 0;
  padding: 0;
  list-style: none;
}
.progress-group {
  padding: 9px 0;
}
.progress-group:last-child {
  padding-bottom: 0;
}
.progress-group + .progress-group {
  border-top: 1px solid var(--guild-border);
}
.progress-node {
  display: flex;
  align-items: baseline;
  flex-wrap: wrap;
  gap: 0 8px;
  margin-bottom: 3px;
  overflow-wrap: anywhere;
}
.progress-node a {
  color: var(--guild-text);
  font-size: var(--guild-item-title-size);
  font-weight: 650;
  text-decoration: none;
}
.progress-node a:hover {
  color: var(--guild-primary);
  text-decoration: underline;
}
.progress-node span {
  color: var(--guild-muted);
  font-size: var(--guild-meta-size);
}
.progress-more {
  margin-top: 3px;
}
.progress-more summary {
  width: fit-content;
  padding: 3px 0;
  color: var(--guild-primary);
  font-size: var(--guild-meta-size);
  cursor: pointer;
}
.earlier-entries {
  display: grid;
  gap: 8px;
  margin-top: 3px;
  padding-inline-start: 12px;
  border-inline-start: 2px solid var(--guild-border);
}
.progress-footer {
  margin-top: 13px;
  padding-top: 12px;
  border-top: 1px solid var(--guild-border);
  text-align: center;
}
</style>
