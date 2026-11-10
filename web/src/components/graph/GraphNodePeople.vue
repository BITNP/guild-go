<script setup lang="ts">
const emit = defineEmits<{ updated: [node: NodeDetail] }>();
defineProps<{ node: NodeDetail }>();

const membersOpen = ref(false);

function onSaved(node: NodeDetail) {
  membersOpen.value = false;
  emit("updated", node);
}
</script>

<template>
  <GraphPanelSection title="参与人员">
    <template #actions>
      <GraphPanelButton
        v-if="node.permissions.canManageMembers"
        @click="membersOpen = true"
      >
        <Pencil :size="12" />
        管理成员
      </GraphPanelButton>
    </template>

    <div class="people-group">
      <h4>直接负责人</h4>
      <ul v-if="node.directLeaders.length > 0" class="people-list">
        <li v-for="person in node.directLeaders" :key="person.subject">
          <UserIdentity :size="24" :user="person" />
        </li>
      </ul>
      <p v-else class="people-empty">暂无直接负责人</p>
    </div>

    <div class="people-group">
      <h4>直接成员</h4>
      <ul v-if="node.directMembers.length > 0" class="people-list">
        <li v-for="person in node.directMembers" :key="person.subject">
          <UserIdentity :size="24" :user="person" />
        </li>
      </ul>
      <p v-else class="people-empty">暂无直接成员</p>
    </div>

    <p
      v-if="node.inheritedLeaderCount || node.inheritedMemberCount"
      class="inherited-count"
    >
      另有 {{ node.inheritedLeaderCount }} 位上级负责人、
      {{ node.inheritedMemberCount }} 位继承成员。
    </p>

    <MembersDialog
      :node
      :open="membersOpen"
      @close="membersOpen = false"
      @saved="onSaved"
    />
  </GraphPanelSection>
</template>

<style scoped>
.people-group + .people-group {
  margin-top: 15px;
}

.people-group h4 {
  margin: 0 0 7px;
  color: #7e9296;
  font-size: 13px;
  font-weight: 500;
}

.people-list {
  display: grid;
  gap: 7px;
  margin: 0;
  padding: 0;
  list-style: none;
  color: #c8d2d2;
  font-size: 15px;
}

.people-empty {
  margin: 0;
  color: #71868b;
  font-size: 14px;
}

.inherited-count {
  margin: 15px 0 0;
  color: #788c90;
  font-size: 13px;
  line-height: 1.6;
}

[data-theme="light"] .people-group h4 {
  color: #607580;
}

[data-theme="light"] .people-list {
  color: #344650;
}

[data-theme="light"] .people-empty {
  color: #758993;
}

[data-theme="light"] .inherited-count {
  color: #70848e;
}
</style>
