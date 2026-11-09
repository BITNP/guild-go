<script setup lang="ts">
const emit = defineEmits<{ close: []; saved: [node: NodeDetail] }>();
const props = defineProps<{ open: boolean; node: NodeDetail }>();
const tab = ref("direct");
const leaders = ref<string[]>([]);
const members = ref<string[]>([]);
const effective = ref<EffectiveMembers | null>(null);
const loading = ref(false);
const loadingChoices = ref(false);
const saving = ref(false);
const error = ref<string | null>(null);

async function load() {
  loading.value = true;
  error.value = null;
  try {
    effective.value = await fetchNodeMembers(props.node.id);
  } catch (err) {
    error.value = errorMessage(err, "无法加载继承成员");
  } finally {
    loading.value = false;
  }
}

watch(
  () => props.open,
  (open) => {
    if (!open) {
      return;
    }
    tab.value = "direct";
    effective.value = null;
    leaders.value = props.node.directLeaders.map((user) => user.subject);
    members.value = props.node.directMembers.map((user) => user.subject);
    void load();
  },
  { immediate: true },
);

async function save() {
  if (saving.value || loading.value || loadingChoices.value) {
    return;
  }
  error.value = null;
  if (leaders.value.length === 0) {
    error.value = "节点至少需要一个直接负责人";

    return;
  }
  saving.value = true;
  try {
    emit(
      "saved",
      await saveNodeMembers(props.node.id, leaders.value, members.value),
    );
    emit("close");
  } catch (err) {
    error.value = errorMessage(err);
  } finally {
    saving.value = false;
  }
}
</script>

<template>
  <AppModal :busy="saving" :open title="管理成员" wide @close="emit('close')">
    <NSpace :size="20" vertical>
      <NAlert v-if="error" role="alert" type="error">
        {{ error }}
        <NButton text @click="load">重新加载</NButton>
      </NAlert>
      <NSpin :show="loading">
        <NTabs v-model:value="tab" type="line">
          <NTabPane name="direct" tab="直接成员">
            <NForm
              :disabled="saving || loading"
              label-placement="top"
              @submit.prevent="save"
            >
              <MemberFields
                v-model:leaders="leaders"
                v-model:loading="loadingChoices"
                v-model:members="members"
                :disabled="saving || loading"
                :open
                :parent-id="node.parentId"
                :root-id="node.breadcrumb[0]?.id ?? node.id"
              />
              <NText class="member-hint" depth="3">
                负责人可管理此节点及下级节点。成员可在此节点及上级节点发布进展。
              </NText>
            </NForm>
          </NTabPane>
          <NTabPane name="inherited" tab="全部成员（含继承）">
            <NGrid cols="1 s:2" responsive="screen" :x-gap="16" :y-gap="16">
              <NGi
                v-for="group in ['leaders', 'members'] as const"
                :key="group"
              >
                <NH3 class="member-group-title">
                  {{ group === "leaders" ? "全部负责人" : "全部成员" }}
                </NH3>
                <NList>
                  <NListItem
                    v-for="entry in effective?.[group] ?? []"
                    :key="entry.user.subject"
                  >
                    <NSpace :size="4" vertical>
                      <UserIdentity :user="entry.user" />
                      <NText class="member-source" depth="3">
                        {{
                          entry.direct
                            ? group === "leaders"
                              ? "直接负责人"
                              : "直接成员"
                            : `继承自 ${entry.sources.map((source) => source.title).join("、")}`
                        }}
                      </NText>
                    </NSpace>
                  </NListItem>
                </NList>
                <NEmpty
                  v-if="effective && effective[group].length === 0"
                  description="暂无成员"
                  size="small"
                />
              </NGi>
            </NGrid>
          </NTabPane>
        </NTabs>
      </NSpin>
      <NSpace class="member-actions" justify="end">
        <NButton :disabled="saving" @click="emit('close')">关闭</NButton>
        <NButton
          v-if="tab === 'direct'"
          :disabled="loading || loadingChoices"
          :loading="saving"
          type="primary"
          @click="save"
        >
          保存成员
        </NButton>
      </NSpace>
    </NSpace>
  </AppModal>
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
