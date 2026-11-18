<script setup lang="ts">
import { Pencil } from '@lucide/vue'
import UserIdentity from '@/components/UserIdentity.vue'
import type { NodeDetail } from '@/lib/api'

defineProps<{ detail: NodeDetail }>()
const emit = defineEmits<{ manage: [] }>()
</script>

<template>
  <section class="panel-section" aria-labelledby="panel-people-title">
    <div class="people-heading">
      <h3 id="panel-people-title">参与人员</h3>
      <button
        v-if="detail.permissions.canManageMembers"
        type="button"
        class="panel-edit-button"
        @click="emit('manage')"
      >
        <Pencil :size="12" />
        管理成员
      </button>
    </div>

    <div class="people-group">
      <h4>直接负责人</h4>
      <ul v-if="detail.directLeaders.length" class="people-list">
        <li v-for="person in detail.directLeaders" :key="person.subject">
          <UserIdentity :user="person" :size="24" />
        </li>
      </ul>
      <p v-else class="panel-empty-copy">暂无直接负责人</p>
    </div>

    <div class="people-group">
      <h4>直接成员</h4>
      <ul v-if="detail.directMembers.length" class="people-list">
        <li v-for="person in detail.directMembers" :key="person.subject">
          <UserIdentity :user="person" :size="24" />
        </li>
      </ul>
      <p v-else class="panel-empty-copy">暂无直接成员</p>
    </div>

    <p
      v-if="detail.inheritedLeaderCount || detail.inheritedMemberCount"
      class="inherited-count"
    >
      另有 {{ detail.inheritedLeaderCount }} 位上级负责人、
      {{ detail.inheritedMemberCount }} 位继承成员。
    </p>
  </section>
</template>

<style scoped>
.panel-section {
  padding: 19px 24px;
  border-bottom: 1px solid var(--graph-panel-divider-subtle);
}

.people-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 13px;
}

.people-heading h3 {
  margin: 0;
  color: var(--graph-section-title-color);
  font-size: 15px;
  font-weight: 700;
}

.panel-edit-button {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 5px 9px;
  border: 1px solid var(--graph-button-border);
  border-radius: 4px;
  background: var(--graph-button-bg);
  color: var(--graph-button-color);
  font-size: 13px;
  cursor: pointer;
}

.panel-edit-button:hover {
  border-color: var(--graph-button-hover-border);
  background: var(--graph-button-hover-bg);
  color: var(--graph-button-hover-color);
}

.people-group + .people-group {
  margin-top: 15px;
}

.people-group h4 {
  margin: 0 0 7px;
  color: var(--graph-muted-label);
  font-size: 13px;
  font-weight: 500;
}

.people-list {
  display: grid;
  gap: 7px;
  margin: 0;
  padding: 0;
  list-style: none;
  color: var(--graph-body-color);
  font-size: 15px;
}

.panel-empty-copy {
  margin: 0;
  color: var(--graph-empty-copy);
  font-size: 14px;
}

.inherited-count {
  margin: 15px 0 0;
  color: var(--graph-inherited-color);
  font-size: 13px;
  line-height: 1.6;
}

@media (max-width: 640px) {
  .panel-section {
    padding-right: 18px;
    padding-left: 18px;
  }
}
</style>
