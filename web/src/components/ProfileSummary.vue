<script setup lang="ts">
import {
  NCard,
  NDescriptions,
  NDescriptionsItem,
  NSpace,
  NTag,
  NText,
} from 'naive-ui'
import UserIdentity from '@/components/UserIdentity.vue'
import type { Profile } from '@/lib/api'
import { departmentLabel } from '@/lib/utils'
defineProps<{ profile: Profile; showPermissions: boolean }>()
</script>
<template>
  <NCard>
    <header class="profile-header">
      <UserIdentity :user="profile" :size="56" :show-name="false" />
      <div class="profile-identity">
        <h2>{{ profile.userName }}</h2>
        <p v-if="profile.email" class="profile-email">{{ profile.email }}</p>
        <NSpace
          ><NTag
            v-for="position in profile.positions"
            :key="position.code"
            type="info"
            :bordered="false"
            >{{ position.label }}</NTag
          ><NText v-if="!profile.positions.length" depth="3"
            >暂无职务信息</NText
          ></NSpace
        >
      </div>
    </header>
    <section
      v-if="showPermissions && profile.permissions"
      class="profile-permissions"
      aria-label="账号权限"
    >
      <h3>账号权限</h3>
      <NDescriptions :column="2" label-placement="top" bordered
        ><NDescriptionsItem label="系统管理员">{{
          profile.permissions.systemAdmin ? '是' : '否'
        }}</NDescriptionsItem
        ><NDescriptionsItem label="可创建根节点">{{
          profile.permissions.manager ? '是' : '否'
        }}</NDescriptionsItem
        ><NDescriptionsItem label="部门访问">{{
          profile.permissions.accessDepartments
            .map(departmentLabel)
            .join('、') || '—'
        }}</NDescriptionsItem
        ><NDescriptionsItem label="部门管理">{{
          profile.permissions.managedDepartments
            .map(departmentLabel)
            .join('、') || '—'
        }}</NDescriptionsItem></NDescriptions
      >
    </section>
    <div v-if="$slots.default" class="profile-footer"><slot /></div>
  </NCard>
</template>
<style scoped>
.profile-email {
  overflow-wrap: anywhere;
  color: var(--guild-muted);
  margin: 4px 0 12px;
}
.profile-header {
  display: flex;
  align-items: flex-start;
  gap: 16px;
}
.profile-identity {
  min-width: 0;
}
.profile-identity h2 {
  margin: 0;
  font-size: 22px;
  line-height: 30px;
  font-weight: 650;
  overflow-wrap: anywhere;
}
.profile-permissions,
.profile-footer {
  margin-top: 24px;
  padding-top: 20px;
  border-top: 1px solid var(--guild-border);
}
.profile-permissions h3 {
  margin: 0 0 12px;
  font-size: 16px;
  font-weight: 600;
}
.profile-footer {
  display: flex;
  flex-direction: column;
  gap: 16px;
  font-size: 13px;
}
</style>
