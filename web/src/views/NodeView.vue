<script setup lang="ts">
import { CalendarDays, Check, Circle } from "@lucide/vue";
import {
  NAlert,
  NBreadcrumb,
  NBreadcrumbItem,
  NButton,
  NCard,
  NEmpty,
  NH1,
  NList,
  NSpace,
  NSpin,
  NTag,
} from "naive-ui";
import { computed, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";

import AppShell from "@/components/AppShell.vue";
import DeleteNodeDialog from "@/components/DeleteNodeDialog.vue";
import MembersDialog from "@/components/MembersDialog.vue";
import MoveNodeDialog from "@/components/MoveNodeDialog.vue";
import NodeDetails from "@/components/NodeDetails.vue";
import NodeFormDialog from "@/components/NodeFormDialog.vue";
import NodeListItem from "@/components/NodeListItem.vue";
import NodeStatusTag from "@/components/NodeStatusTag.vue";
import OrderDialog from "@/components/OrderDialog.vue";
import ProgressSection from "@/components/ProgressSection.vue";
import SectionHeading from "@/components/SectionHeading.vue";
import { useNodeStore } from "@/composables/useNodeStore";
import type { NodeDetail } from "@/lib/api";
import { fetchNode, updateNodeFromDetail } from "@/lib/api";
import { errorMessage, formatDateTime, linkify } from "@/lib/utils";

const route = useRoute();
const router = useRouter();
const { reload: reloadNodes } = useNodeStore();
const nodeId = computed(() => Number(route.params.id));
const detail = ref<NodeDetail | null>(null);
const loading = ref(true);
const error = ref<string | null>(null);
const actionError = ref<string | null>(null);
const toggling = ref(false);
const editOpen = ref(false);
const createOpen = ref(false);
const membersOpen = ref(false);
const moveOpen = ref(false);
const orderOpen = ref(false);
const deleteOpen = ref(false);
const descriptionSegments = computed(() =>
  linkify(detail.value?.description ?? ""),
);

async function load(silent = false) {
  const requested = nodeId.value;
  if (!Number.isSafeInteger(requested) || requested <= 0) {
    detail.value = null;
    error.value = "节点不存在";
    loading.value = false;

    return;
  }
  if (!silent) {
    loading.value = true;
  }
  error.value = null;
  actionError.value = null;
  try {
    const node = await fetchNode(requested);
    if (requested === nodeId.value) {
      detail.value = node;
    }
  } catch (err) {
    if (requested === nodeId.value) {
      error.value = errorMessage(err, "无法加载节点");
      detail.value = null;
    }
  } finally {
    if (requested === nodeId.value) {
      loading.value = false;
    }
  }
}

async function toggleComplete() {
  const node = detail.value;
  if (!node || !node.permissions.canEdit || toggling.value) {
    return;
  }
  toggling.value = true;
  actionError.value = null;
  try {
    const updated = await updateNodeFromDetail(node, {
      completed: !node.completedAt,
    });
    if (node.id === nodeId.value) {
      detail.value = updated;
    }
    await reloadNodes();
  } catch (err) {
    if (node.id === nodeId.value) {
      actionError.value = errorMessage(err, "更新完成状态失败");
    }
  } finally {
    toggling.value = false;
  }
}

function onSaved(node: NodeDetail) {
  if (node.id === nodeId.value) {
    detail.value = node;
  }
  editOpen.value = false;
  membersOpen.value = false;
  moveOpen.value = false;
  void reloadNodes();
}

async function onCreated(node: NodeDetail) {
  createOpen.value = false;
  await reloadNodes();
  await router.push({ name: "node", params: { id: node.id } });
}

async function onOrdered() {
  orderOpen.value = false;
  await Promise.all([load(true), reloadNodes()]);
}

async function onDeleted() {
  const parentId = detail.value?.parentId;
  deleteOpen.value = false;
  await reloadNodes();
  await router.push(
    parentId ? { name: "node", params: { id: parentId } } : { name: "home" },
  );
}

async function onCommentChanged() {
  await Promise.all([load(true), reloadNodes()]);
}

watch(
  nodeId,
  () => {
    editOpen.value = false;
    createOpen.value = false;
    membersOpen.value = false;
    moveOpen.value = false;
    orderOpen.value = false;
    deleteOpen.value = false;
    void load();
  },
  { immediate: true },
);
</script>

<template>
  <AppShell>
    <section
      :aria-busy="loading"
      class="node-page"
      :inert="loading || undefined"
    >
      <NSpin :show="loading">
        <NSpace :size="24" vertical>
          <NAlert v-if="error" type="error">
            {{ error }}
            <NButton text @click="load()">重试</NButton>
            <NButton text @click="router.push({ name: 'home' })">
              返回总览
            </NButton>
          </NAlert>
          <div v-else-if="loading && !detail" class="node-placeholder" />
          <template v-else-if="detail">
            <header class="node-header">
              <NBreadcrumb aria-label="节点路径" class="node-breadcrumb">
                <NBreadcrumbItem class="breadcrumb-item">
                  <RouterLink :to="{ name: 'home' }">总览</RouterLink>
                </NBreadcrumbItem>
                <NBreadcrumbItem
                  v-for="item in detail.breadcrumb"
                  :key="item.id"
                  class="breadcrumb-item"
                >
                  <RouterLink
                    class="breadcrumb-label"
                    :to="{ name: 'node', params: { id: item.id } }"
                  >
                    {{ item.title }}
                  </RouterLink>
                </NBreadcrumbItem>
                <NBreadcrumbItem
                  aria-current="page"
                  class="breadcrumb-item"
                  :clickable="false"
                >
                  <span class="breadcrumb-label">{{ detail.title }}</span>
                </NBreadcrumbItem>
              </NBreadcrumb>
              <NCard class="node-summary" size="small">
                <div class="node-heading">
                  <div class="node-title">
                    <NButton
                      :aria-label="
                        detail.completedAt ? '标记为未完成' : '标记为已完成'
                      "
                      :aria-pressed="!!detail.completedAt"
                      circle
                      class="node-completion"
                      :disabled="!detail.permissions.canEdit"
                      :loading="toggling"
                      quaternary
                      :title="
                        detail.completedAt ? '标记为未完成' : '标记为已完成'
                      "
                      :type="detail.completedAt ? 'success' : 'default'"
                      @click="toggleComplete"
                    >
                      <Check v-if="detail.completedAt" :size="24" />
                      <Circle v-else :size="26" />
                    </NButton>
                    <NH1 :class="{ completed: !!detail.completedAt }">
                      {{ detail.title }}
                    </NH1>
                  </div>
                  <NSpace align="center" class="node-metadata" :size="8">
                    <NodeStatusTag
                      :completed-at="detail.completedAt"
                      :due-at="detail.dueAt"
                    />
                    <NTag v-for="tag in detail.tags" :key="tag" size="small">
                      {{ tag }}
                    </NTag>
                    <span v-if="detail.dueAt" class="due-date">
                      <CalendarDays aria-hidden="true" :size="13" />
                      预期完成 {{ formatDateTime(detail.dueAt) }}
                    </span>
                  </NSpace>
                  <NSpace
                    v-if="
                      detail.permissions.canEdit ||
                      detail.permissions.canManageMembers ||
                      detail.permissions.canDelete
                    "
                    class="node-actions"
                  >
                    <NButton
                      v-if="detail.permissions.canEdit"
                      size="small"
                      @click="editOpen = true"
                    >
                      编辑节点
                    </NButton>
                    <NButton
                      v-if="detail.permissions.canManageMembers"
                      size="small"
                      @click="membersOpen = true"
                    >
                      管理成员
                    </NButton>
                    <NButton
                      v-if="detail.permissions.canEdit"
                      size="small"
                      @click="moveOpen = true"
                    >
                      移动节点
                    </NButton>
                    <NButton
                      v-if="detail.permissions.canDelete"
                      quaternary
                      size="small"
                      type="error"
                      @click="deleteOpen = true"
                    >
                      删除节点
                    </NButton>
                  </NSpace>
                </div>
                <p v-if="detail.description" class="node-description">
                  <template
                    v-for="(segment, index) in descriptionSegments"
                    :key="index"
                  >
                    <a
                      v-if="segment.type === 'link'"
                      :href="segment.href"
                      rel="noopener noreferrer"
                      target="_blank"
                    >
                      {{ segment.value }}
                    </a>
                    <template v-else>{{ segment.value }}</template>
                  </template>
                </p>
              </NCard>
            </header>
            <NAlert v-if="actionError" type="error">{{ actionError }}</NAlert>
            <section aria-label="子节点" class="node-children">
              <SectionHeading :count="detail.children.length" title="子节点">
                <NButton
                  v-if="
                    detail.permissions.canReorderChildren &&
                    detail.children.length > 1
                  "
                  size="small"
                  @click="orderOpen = true"
                >
                  调整顺序
                </NButton>
                <NButton
                  v-if="detail.permissions.canAddChild"
                  size="small"
                  type="primary"
                  @click="createOpen = true"
                >
                  新建子节点
                </NButton>
              </SectionHeading>
              <NCard size="small">
                <NList v-if="detail.children.length > 0">
                  <NodeListItem
                    v-for="child in detail.children"
                    :key="child.id"
                    :node="child"
                  />
                </NList>
                <NEmpty v-else description="暂无子节点" size="small" />
              </NCard>
            </section>
            <ProgressSection
              :key="detail.id"
              :can-comment="detail.permissions.canComment"
              :node-id="detail.id"
              :updated-at="detail.updatedAt"
              @changed="onCommentChanged"
            />
          </template>
        </NSpace>
      </NSpin>
    </section>
    <template #rail>
      <div :aria-busy="loading" :inert="loading || undefined">
        <NodeDetails
          v-if="detail && !error"
          :node="detail"
          @manage-members="membersOpen = true"
        />
      </div>
    </template>
  </AppShell>
  <template v-if="detail">
    <NodeFormDialog
      v-model:open="editOpen"
      mode="edit"
      :node="detail"
      @saved="onSaved"
    />
    <NodeFormDialog
      v-model:open="createOpen"
      mode="create"
      :parent-id="detail.id"
      :parent-title="detail.title"
      @saved="onCreated"
    />
    <MembersDialog v-model:open="membersOpen" :node="detail" @saved="onSaved" />
    <MoveNodeDialog v-model:open="moveOpen" :node="detail" @saved="onSaved" />
    <OrderDialog
      v-model:open="orderOpen"
      :children="detail.children"
      :parent-id="detail.id"
      :parent-title="detail.title"
      @saved="onOrdered"
    />
    <DeleteNodeDialog
      v-model:open="deleteOpen"
      :node="detail"
      @deleted="onDeleted"
    />
  </template>
</template>

<style scoped>
.node-page {
  max-width: 1080px;
  margin: auto;
  padding: 20px;
}
.node-header {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.node-breadcrumb {
  font-size: 12px;
  white-space: normal;
}
.breadcrumb-item {
  max-width: 100%;
}
.breadcrumb-label {
  overflow-wrap: anywhere;
}
.node-metadata {
  padding-left: 46px;
  grid-area: metadata;
}
.due-date {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  color: var(--guild-muted);
  font-size: 12px;
}
.node-children {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.node-heading {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  grid-template-areas: "title actions" "metadata metadata";
  gap: 8px 16px;
}
.node-actions {
  grid-area: actions;
  align-self: start;
}
.node-title {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  grid-area: title;
  min-width: 0;
}
.node-title h1 {
  font-size: var(--guild-page-title-size);
  line-height: 36px;
  font-weight: 600;
  margin: 0;
  overflow-wrap: anywhere;
}
.node-completion {
  flex: none;
}
.completed {
  text-decoration: line-through;
  color: var(--guild-muted);
}
.node-description {
  margin: 16px 0 0;
  padding-top: 12px;
  border-top: 1px solid var(--guild-border);
  white-space: pre-wrap;
  overflow-wrap: anywhere;
  line-height: 1.8;
}
.node-description a {
  color: var(--guild-primary);
  text-decoration: underline;
  text-underline-offset: 3px;
}
.node-placeholder {
  min-height: 280px;
}
@media (max-width: 639px) {
  .node-page {
    padding: 16px;
  }
  .node-title h1 {
    font-size: 22px;
  }
  .node-heading {
    grid-template-columns: minmax(0, 1fr);
    grid-template-areas: "title" "metadata" "actions";
  }
  .node-actions {
    margin-top: 4px;
  }
  .node-metadata {
    padding-left: 0;
  }
}
</style>
