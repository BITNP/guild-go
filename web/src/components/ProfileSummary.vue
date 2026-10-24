<script setup lang="ts">
import {
  NCard,
  NDescriptions,
  NDescriptionsItem,
  NSpace,
  NTag,
  NText,
  NThing,
} from 'naive-ui'
import UserIdentity from '@/components/UserIdentity.vue'
import type { Profile } from '@/lib/api'
import { departmentLabel } from '@/lib/utils'
defineProps<{ profile: Profile; showPermissions: boolean }>()
</script>
<template>
  <NCard
    ><NSpace vertical :size="24"
      ><NThing
        ><template #avatar
          ><UserIdentity
            :user="profile"
            :size="56"
            :show-name="false" /></template
        ><template #header>{{ profile.userName }}</template
        ><template #description
          ><span class="profile-email">{{ profile.email }}</span></template
        ><NSpace
          ><NTag
            v-for="position in profile.positions"
            :key="position.code"
            type="info"
            :bordered="false"
            >{{ position.label }}</NTag
          ><NText v-if="!profile.positions.length" depth="3"
            >暂无职务信息</NText
          ></NSpace
        ></NThing
      ><NDescriptions
        v-if="showPermissions && profile.permissions"
        title="权限"
        :column="2"
        label-placement="top"
        bordered
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
      ><slot /></NSpace
  ></NCard>
</template>
<style scoped>
.profile-email {
  overflow-wrap: anywhere;
}
</style>
