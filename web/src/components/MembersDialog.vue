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
const open = defineModel<boolean>('open', { required: true })
const props = defineProps<{ node: NodeDetail }>(),
  emit = defineEmits<{ saved: [node: NodeDetail] }>()
const tab = ref('direct'),
  leaders = ref<string[]>([]),
  members = ref<string[]>([]),
  effective = ref<EffectiveMembers | null>(null),
  loading = ref(false),
  loadingChoices = ref(false),
  saving = ref(false),
  loadError = ref<string | null>(null),
  saveError = ref<string | null>(null)
async function load() {
  loading.value = true
  loadError.value = null
  try {
    effective.value = await fetchNodeMembers(props.node.id)
  } catch (err) {
    loadError.value = errorMessage(err, '无法加载继承成员')
  } finally {
    loading.value = false
  }
}
watch(
  open,
  (isOpen) => {
    if (!isOpen) return
    tab.value = 'direct'
    effective.value = null
    loadError.value = null
    saveError.value = null
    leaders.value = props.node.directLeaders.map((user) => user.subject)
    members.value = props.node.directMembers.map((user) => user.subject)
    void load()
  },
  { immediate: true },
)
async function save() {
  if (saving.value || loadingChoices.value) return
  saveError.value = null
  if (!leaders.value.length) {
    saveError.value = '节点至少需要一个直接负责人'
    return
  }
  saving.value = true
  try {
    emit(
      'saved',
      await saveNodeMembers(props.node.id, leaders.value, members.value),
    )
    open.value = false
  } catch (err) {
    saveError.value = errorMessage(err)
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
    @close="open = false"
    ><NSpace vertical :size="20"
      ><NTabs v-model:value="tab" type="line"
        ><NTabPane name="direct" tab="直接成员"
          ><NForm
            label-placement="top"
            :disabled="saving"
            @submit.prevent="save"
            ><MemberFields
              v-model:leaders="leaders"
              v-model:members="members"
              v-model:loading="loadingChoices"
              :open="open"
              :parent-id="node.parentId"
              :root-id="node.breadcrumb[0]?.id ?? node.id"
              :disabled="saving"
            /><NText depth="3" class="member-hint"
              >负责人可管理此节点及下级节点。成员可在此节点及上级节点发布进展。</NText
            ><NAlert
              v-if="saveError"
              type="error"
              role="alert"
              class="member-save-error"
              >{{ saveError }}</NAlert
            ></NForm
          ></NTabPane
        ><NTabPane name="inherited" tab="全部成员（含继承）"
          ><NAlert v-if="loadError" type="error" role="alert"
            >{{ loadError }}
            <NButton text @click="load">重新加载</NButton></NAlert
          ><NSpin v-else :show="loading"
            ><div v-if="loading && !effective" class="inherited-placeholder" />
            <NGrid v-else cols="1 s:2" responsive="screen" :x-gap="16" :y-gap="16"
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
                        ><template v-if="entry.direct">{{
                          group === 'leaders' ? '直接负责人' : '直接成员'
                        }}</template
                        ><template v-else
                          >继承自
                          <template
                            v-for="(source, index) in entry.sources"
                            :key="source.id"
                            ><template v-if="index > 0">、</template
                            ><RouterLink
                              :to="{ name: 'node', params: { id: source.id } }"
                              class="source-link"
                              @click="open = false"
                              >{{ source.title }}</RouterLink
                            ></template
                          ></template
                        ></NText
                      ></NSpace
                    ></NListItem
                  ></NList
                ><NEmpty
                  v-if="effective && !effective[group].length"
                  description="暂无成员"
                  size="small" /></NGi></NGrid></NSpin></NTabPane></NTabs
      ><NSpace justify="end" class="member-actions"
        ><NButton :disabled="saving" @click="open = false">关闭</NButton
        ><NButton
          v-if="tab === 'direct'"
          type="primary"
          :loading="saving"
          :disabled="loadingChoices"
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
.member-save-error {
  margin-top: 16px;
}
.inherited-placeholder {
  min-height: 120px;
}
.member-source {
  font-size: var(--guild-meta-size);
}
.source-link {
  color: var(--guild-primary);
  text-decoration: none;
}
.source-link:hover {
  text-decoration: underline;
}
.member-actions {
  padding-top: 16px;
  border-top: 1px solid var(--guild-border);
}
</style>
