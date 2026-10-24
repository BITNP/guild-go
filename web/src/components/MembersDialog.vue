<script setup lang="ts">
import { ref, watch } from 'vue'
import {
  NAlert,
  NButton,
  NEmpty,
  NForm,
  NGrid,
  NGi,
  NH3,
  NList,
  NListItem,
  NSpace,
  NSpin,
  NTabPane,
  NTabs,
  NText,
} from 'naive-ui'
import AppModal from '@/components/AppModal.vue'
import MemberFields from '@/components/MemberFields.vue'
import UserIdentity from '@/components/UserIdentity.vue'
import {
  fetchNodeMembers,
  fetchUsers,
  saveNodeMembers,
  type EffectiveMembers,
  type NodeDetail,
  type UserSummary,
} from '@/lib/api'
import { errorMessage } from '@/lib/utils'
const props = defineProps<{ open: boolean; node: NodeDetail }>(),
  emit = defineEmits<{ close: []; saved: [node: NodeDetail] }>()
const tab = ref('direct'),
  leaders = ref<string[]>([]),
  members = ref<string[]>([]),
  users = ref<UserSummary[]>([]),
  effective = ref<EffectiveMembers | null>(null),
  loading = ref(false),
  saving = ref(false),
  error = ref<string | null>(null)
async function load() {
  loading.value = true
  error.value = null
  const result = await Promise.allSettled([
    fetchUsers(),
    fetchNodeMembers(props.node.id),
  ])
  if (result[0].status === 'fulfilled') users.value = result[0].value
  else error.value = errorMessage(result[0].reason, '无法加载用户列表')
  if (result[1].status === 'fulfilled') effective.value = result[1].value
  else error.value = errorMessage(result[1].reason, '无法加载继承成员')
  loading.value = false
}
watch(
  () => props.open,
  (open) => {
    if (!open) return
    tab.value = 'direct'
    effective.value = null
    leaders.value = props.node.directLeaders.map((user) => user.subject)
    members.value = props.node.directMembers.map((user) => user.subject)
    void load()
  },
  { immediate: true },
)
async function save() {
  if (saving.value) return
  error.value = null
  if (!leaders.value.length) {
    error.value = '节点至少需要一个直接负责人'
    return
  }
  saving.value = true
  try {
    emit(
      'saved',
      await saveNodeMembers(props.node.id, leaders.value, members.value),
    )
    emit('close')
  } catch (err) {
    error.value = errorMessage(err)
  } finally {
    saving.value = false
  }
}
</script>
<template>
  <AppModal
    :open="open"
    title="管理成员"
    :busy="saving"
    wide
    @close="emit('close')"
    ><NSpace vertical :size="20"
      ><NAlert v-if="error" type="error" role="alert"
        >{{ error }} <NButton text @click="load">重新加载</NButton></NAlert
      ><NSpin :show="loading"
        ><NTabs v-model:value="tab" type="line"
          ><NTabPane name="direct" tab="直接成员"
            ><NForm
              label-placement="top"
              :disabled="saving || loading"
              @submit.prevent="save"
              ><MemberFields
                v-model:leaders="leaders"
                v-model:members="members"
                :users="users"
                :loading="loading"
                :disabled="saving || loading"
              /><NText depth="3"
                >负责人可管理此节点及下级节点。成员可在此节点及上级节点发表评论。</NText
              ></NForm
            ></NTabPane
          ><NTabPane name="inherited" tab="全部成员（含继承）"
            ><NGrid cols="1 s:2" responsive="screen" :x-gap="16" :y-gap="16"
              ><NGi
                v-for="group in ['leaders', 'members'] as const"
                :key="group"
                ><NH3>{{ group === 'leaders' ? '全部负责人' : '全部成员' }}</NH3
                ><NList
                  ><NListItem
                    v-for="entry in effective?.[group] ?? []"
                    :key="entry.user.subject"
                    ><NSpace vertical :size="4"
                      ><UserIdentity :user="entry.user" /><NText depth="3">{{
                        entry.direct
                          ? group === 'leaders'
                            ? '直接负责人'
                            : '直接成员'
                          : `继承自 ${entry.sources.map((source) => source.title).join('、')}`
                      }}</NText></NSpace
                    ></NListItem
                  ></NList
                ><NEmpty
                  v-if="effective && !effective[group].length"
                  description="暂无成员"
                  size="small" /></NGi></NGrid></NTabPane></NTabs></NSpin
      ><NSpace justify="end"
        ><NButton :disabled="saving" @click="emit('close')">关闭</NButton
        ><NButton
          v-if="tab === 'direct'"
          type="primary"
          :loading="saving"
          :disabled="loading"
          @click="save"
          >保存成员</NButton
        ></NSpace
      ></NSpace
    ></AppModal
  >
</template>
