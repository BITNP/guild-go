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
  saveNodeMembers,
  type EffectiveMembers,
  type NodeDetail,
} from '@/lib/api'
import { errorMessage } from '@/lib/utils'
const props = defineProps<{ open: boolean; node: NodeDetail }>(),
  emit = defineEmits<{ close: []; saved: [node: NodeDetail] }>()
const tab = ref('direct'),
  leaders = ref<string[]>([]),
  members = ref<string[]>([]),
  effective = ref<EffectiveMembers | null>(null),
  loading = ref(false),
  loadingChoices = ref(false),
  saving = ref(false),
  error = ref<string | null>(null)
async function load() {
  loading.value = true
  error.value = null
  try {
    effective.value = await fetchNodeMembers(props.node.id)
  } catch (err) {
    error.value = errorMessage(err, '无法加载继承成员')
  } finally {
    loading.value = false
  }
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
  if (saving.value || loading.value || loadingChoices.value) return
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
                v-model:loading="loadingChoices"
                :open="open"
                :parent-id="node.parentId"
                :root-id="node.breadcrumb[0]?.id ?? node.id"
                :disabled="saving || loading"
              /><NText depth="3" class="member-hint"
                >负责人可管理此节点及下级节点。成员可在此节点及上级节点发表评论。</NText
              ></NForm
            ></NTabPane
          ><NTabPane name="inherited" tab="全部成员（含继承）"
            ><NGrid cols="1 s:2" responsive="screen" :x-gap="16" :y-gap="16"
              ><NGi
                v-for="group in ['leaders', 'members'] as const"
                :key="group"
                ><NH3 class="member-group-title">{{
                  group === 'leaders' ? '全部负责人' : '全部成员'
                }}</NH3
                ><NList
                  ><NListItem
                    v-for="entry in effective?.[group] ?? []"
                    :key="entry.user.subject"
                    ><NSpace vertical :size="4"
                      ><UserIdentity :user="entry.user" /><NText
                        depth="3"
                        class="member-source"
                        >{{
                          entry.direct
                            ? group === 'leaders'
                              ? '直接负责人'
                              : '直接成员'
                            : `继承自 ${entry.sources.map((source) => source.title).join('、')}`
                        }}</NText
                      ></NSpace
                    ></NListItem
                  ></NList
                ><NEmpty
                  v-if="effective && !effective[group].length"
                  description="暂无成员"
                  size="small" /></NGi></NGrid></NTabPane></NTabs></NSpin
      ><NSpace justify="end" class="member-actions"
        ><NButton :disabled="saving" @click="emit('close')">关闭</NButton
        ><NButton
          v-if="tab === 'direct'"
          type="primary"
          :loading="saving"
          :disabled="loading || loadingChoices"
          @click="save"
          >保存成员</NButton
        ></NSpace
      ></NSpace
    ></AppModal
  >
</template>
<style scoped>
.member-group-title {
  margin: 4px 0 12px;
  font-size: 14px;
  font-weight: 600;
}
.member-hint {
  display: block;
  margin-top: 12px;
  font-size: 13px;
}
.member-source {
  font-size: var(--guild-meta-size);
}
.member-actions {
  padding-top: 16px;
  border-top: 1px solid var(--guild-border);
}
</style>
