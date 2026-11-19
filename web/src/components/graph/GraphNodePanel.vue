<script setup lang="ts">
import {
  ArrowUpDown,
  Check,
  ChevronRight,
  Circle,
  ExternalLink,
  FolderInput,
  Pencil,
  Plus,
  Trash2,
  X,
} from "@lucide/vue";
import { computed, ref, watch } from "vue";
import { RouterLink } from "vue-router";

import MembersDialog from "@/components/MembersDialog.vue";
import NodeStatusTag from "@/components/NodeStatusTag.vue";
import GraphNodeMetaSection from "@/components/graph/GraphNodeMetaSection.vue";
import GraphNodePeopleSection from "@/components/graph/GraphNodePeopleSection.vue";
import type { GraphNode } from "@/graph/types";
import type { NodeDetail } from "@/lib/api";
import { updateNodeFromDetail } from "@/lib/api";
import { errorMessage, isOverdue, linkify } from "@/lib/utils";

const emit = defineEmits<{
  close: [];
  navigate: [id: number];
  retry: [];
  updated: [node: NodeDetail];
  edit: [];
  move: [];
  createChild: [];
  createRoot: [];
  remove: [];
  reorder: [];
}>();

const props = defineProps<{
  selected: GraphNode;
  detail: NodeDetail | null;
  parentId: number | null;
  parentTitle: string | null;
  loading: boolean;
  error: string | null;
  isVirtualRoot: boolean;
}>();

const membersOpen = ref(false);
const savingStatus = ref(false);
const statusError = ref<string | null>(null);
const title = computed(() => props.detail?.title ?? props.selected.title);
const descriptionSegments = computed(() =>
  linkify(props.detail?.description ?? ""),
);

watch(
  () => props.selected.id,
  () => {
    membersOpen.value = false;
    statusError.value = null;
  },
);

function onMembersSaved(node: NodeDetail) {
  membersOpen.value = false;
  emit("updated", node);
}

function createNode() {
  if (props.isVirtualRoot) {
    emit("createRoot");
  } else {
    emit("createChild");
  }
}

async function toggleCompletion() {
  const node = props.detail;
  if (!node || !node.permissions.canEdit || savingStatus.value) {
    return;
  }
  savingStatus.value = true;
  statusError.value = null;
  try {
    const updated = await updateNodeFromDetail(node, {
      completed: !node.completedAt,
    });
    emit("updated", updated);
  } catch (err) {
    statusError.value = errorMessage(err, "更新完成状态失败");
  } finally {
    savingStatus.value = false;
  }
}
</script>

<template>
  <aside
    aria-label="节点控制面板"
    class="node-control-panel"
    data-graph-node-panel
  >
    <header class="panel-header">
      <button
        aria-label="关闭节点面板"
        class="panel-close"
        type="button"
        @click="emit('close')"
      >
        <X :size="17" />
      </button>
      <h2>{{ title }}</h2>
      <div v-if="detail && !isVirtualRoot" class="panel-status-row">
        <NodeStatusTag
          :completed-at="detail.completedAt"
          :due-at="detail.dueAt"
        />
        <button
          v-if="detail.permissions.canEdit"
          :aria-pressed="!!detail.completedAt"
          class="status-toggle"
          :disabled="savingStatus"
          type="button"
          @click="toggleCompletion"
        >
          {{
            savingStatus
              ? "正在更新…"
              : detail.completedAt
                ? "标记为未完成"
                : "标记为已完成"
          }}
        </button>
      </div>
      <p v-if="statusError" class="status-error" role="alert">
        {{ statusError }}
      </p>
      <div v-if="detail?.tags.length" class="panel-tag-list">
        <span v-for="tag in detail.tags" :key="tag" class="panel-tag">
          {{ tag }}
        </span>
      </div>
      <div v-if="detail && !isVirtualRoot" class="panel-node-actions">
        <button
          v-if="detail.permissions.canEdit"
          class="panel-edit-button"
          type="button"
          @click="emit('edit')"
        >
          <Pencil :size="12" />
          编辑节点
        </button>
        <button
          v-if="detail.permissions.canEdit"
          class="panel-edit-button"
          type="button"
          @click="emit('move')"
        >
          <FolderInput :size="12" />
          移动节点
        </button>
        <button
          v-if="detail.permissions.canDelete"
          class="panel-delete-button"
          type="button"
          @click="emit('remove')"
        >
          <Trash2 :size="12" />
          删除节点
        </button>
      </div>
    </header>

    <div class="panel-scroll">
      <div
        v-if="!detail && (loading || !error)"
        aria-live="polite"
        class="panel-loading"
      >
        <span class="panel-loading-mark" />
        正在加载节点…
      </div>

      <div v-else-if="error && !detail" class="panel-error" role="alert">
        <p>{{ error }}</p>
        <button type="button" @click="emit('retry')">重新加载</button>
      </div>

      <template v-else-if="detail">
        <section aria-labelledby="panel-structure-title" class="panel-section">
          <h3 id="panel-structure-title">
            {{ isVirtualRoot ? "全部根节点" : "层级关系" }}
          </h3>
          <div v-if="!isVirtualRoot" class="panel-relation">
            <span class="relation-label">父节点</span>
            <button
              v-if="parentId != null"
              class="relation-link"
              type="button"
              @click="emit('navigate', parentId)"
            >
              <span>{{ parentTitle ?? `节点 #${parentId}` }}</span>
              <ChevronRight :size="14" />
            </button>
            <span v-else class="relation-empty">当前为根节点</span>
          </div>

          <div class="panel-children">
            <div class="children-heading">
              <span>{{ isVirtualRoot ? "根节点" : "子节点" }}</span>
              <span>{{ detail.children.length }}</span>
            </div>
            <div class="children-actions">
              <button
                v-if="
                  detail.permissions.canReorderChildren &&
                  detail.children.length > 1
                "
                class="panel-edit-button"
                type="button"
                @click="emit('reorder')"
              >
                <ArrowUpDown :size="12" />
                {{ isVirtualRoot ? "调整根节点顺序" : "调整子节点顺序" }}
              </button>
              <button
                v-if="detail.permissions.canAddChild"
                class="panel-edit-button is-primary"
                type="button"
                @click="createNode"
              >
                <Plus :size="12" />
                {{ isVirtualRoot ? "新建根节点" : "新建子节点" }}
              </button>
            </div>
            <div v-if="detail.children.length > 0" class="children-list">
              <button
                v-for="child in detail.children"
                :key="child.id"
                class="child-link"
                type="button"
                @click="emit('navigate', child.id)"
              >
                <Check
                  v-if="child.completedAt"
                  aria-label="已完成"
                  class="child-icon is-completed"
                  :size="14"
                />
                <Circle
                  v-else
                  aria-label="未完成"
                  class="child-icon"
                  :class="[
                    { 'is-overdue': isOverdue(child.dueAt, child.completedAt) },
                  ]"
                  :size="14"
                />
                <span class="child-title">{{ child.title }}</span>
                <span v-if="child.childCount" class="child-count">
                  {{ child.childCount }} 个子节点
                </span>
                <ChevronRight :size="13" />
              </button>
            </div>
            <p v-else class="relation-empty">
              {{ isVirtualRoot ? "暂无根节点" : "暂无子节点" }}
            </p>
          </div>
        </section>

        <section
          v-if="!isVirtualRoot"
          aria-labelledby="panel-description-title"
          class="panel-section"
        >
          <h3 id="panel-description-title">节点描述</h3>
          <p v-if="detail.description" class="panel-description">
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
          <p v-else class="panel-empty-copy">暂无节点描述</p>
        </section>

        <GraphNodeMetaSection v-if="!isVirtualRoot" :detail />

        <GraphNodePeopleSection
          v-if="!isVirtualRoot"
          :detail
          @manage="membersOpen = true"
        />
      </template>
    </div>

    <footer v-if="!isVirtualRoot" class="panel-footer">
      <RouterLink
        class="panel-page-link"
        :to="{ name: 'node', params: { id: selected.id } }"
      >
        <span>打开节点页面</span>
        <ExternalLink aria-hidden="true" :size="15" />
      </RouterLink>
    </footer>

    <MembersDialog
      v-if="detail && !isVirtualRoot"
      v-model:open="membersOpen"
      :node="detail"
      @saved="onMembersSaved"
    />
  </aside>
</template>

<style scoped>
.node-control-panel {
  position: absolute;
  z-index: 20;
  top: 16px;
  right: 16px;
  bottom: 16px;
  display: flex;
  width: clamp(320px, 28vw, 400px);
  min-height: 0;
  flex-direction: column;
  overflow: hidden;
  border: 1px solid var(--graph-panel-border);
  border-radius: 8px;
  background: var(--graph-panel-bg);
  box-shadow: var(--graph-panel-shadow);
  color: var(--graph-panel-text);
  pointer-events: auto;
  backdrop-filter: blur(22px);
  will-change: transform, opacity;
}

.node-panel-enter-active {
  transition:
    transform 260ms cubic-bezier(0.22, 0.76, 0.24, 1),
    opacity 200ms ease-out;
}

.node-panel-leave-active {
  transition:
    transform 200ms cubic-bezier(0.55, 0, 0.8, 0.42),
    opacity 180ms ease-in;
}

.node-panel-enter-from,
.node-panel-leave-to {
  opacity: 0;
  transform: translateX(calc(100% + 24px));
}

.node-panel-enter-to,
.node-panel-leave-from {
  opacity: 1;
  transform: translateX(0);
}

.panel-header {
  position: relative;
  flex: none;
  padding: 22px 24px 18px;
  border-bottom: 1px solid var(--graph-panel-divider);
  background: var(--graph-panel-header-bg);
}

.panel-close {
  position: absolute;
  top: 16px;
  right: 16px;
  display: grid;
  width: 30px;
  height: 30px;
  place-items: center;
  border: 1px solid var(--graph-close-border);
  border-radius: 4px;
  background: var(--graph-close-bg);
  color: var(--graph-close-color);
  cursor: pointer;
}

.panel-close:hover {
  border-color: var(--graph-close-hover-border);
  color: var(--graph-close-hover-color);
}

.panel-header h2 {
  margin: 0 40px 0 0;
  color: var(--graph-panel-title-color);
  font-size: clamp(22px, 1.7vw, 28px);
  font-weight: 650;
  line-height: 1.35;
  overflow-wrap: anywhere;
}

.panel-status-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 7px;
  margin-top: 11px;
}

.panel-status-row :deep(.n-tag) {
  font-size: 13px;
}

.status-toggle {
  padding: 3px 8px;
  border: 1px solid var(--graph-button-border);
  border-radius: 4px;
  background: var(--graph-button-bg);
  color: var(--graph-button-color);
  font-size: 13px;
  cursor: pointer;
}

.status-toggle:hover {
  border-color: var(--graph-button-hover-border);
  background: var(--graph-button-hover-bg);
  color: var(--graph-button-hover-color);
}

.status-toggle:disabled {
  cursor: wait;
  opacity: 0.58;
}

.status-error {
  margin: 8px 0 0;
  color: var(--graph-status-error-color);
  font-size: 13px;
}

.panel-tag-list {
  display: flex;
  flex-wrap: wrap;
  gap: 5px;
  margin-top: 9px;
}

.panel-tag {
  padding: 2px 7px;
  border: 1px solid var(--graph-tag-border);
  border-radius: 4px;
  color: var(--graph-tag-color);
  font-size: 13px;
}

.panel-node-actions,
.children-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.panel-node-actions {
  margin-top: 12px;
}

.children-actions {
  margin: -3px 0 9px;
}

.panel-scroll {
  min-height: 0;
  flex: 1 1 auto;
  overflow-y: auto;
  overscroll-behavior: contain;
  scrollbar-color: rgba(221, 176, 75, 0.45) transparent;
  scrollbar-width: thin;
}

.panel-section {
  padding: 19px 24px;
  border-bottom: 1px solid var(--graph-panel-divider-subtle);
}

.panel-section h3 {
  margin: 0 0 13px;
  color: var(--graph-section-title-color);
  font-size: 15px;
  font-weight: 700;
}

.panel-relation {
  display: grid;
  grid-template-columns: 64px minmax(0, 1fr);
  align-items: center;
  gap: 10px;
}

.relation-label {
  color: var(--graph-muted-label);
  font-size: 13px;
}

.relation-link,
.child-link {
  border: 0;
  background: transparent;
  color: var(--graph-link-color);
  cursor: pointer;
}

.relation-link {
  display: flex;
  min-width: 0;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 8px 10px;
  border: 1px solid var(--graph-link-border);
  border-radius: 4px;
  background: var(--graph-link-bg);
  font-size: 14px;
  text-align: left;
}

.relation-link span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.relation-link:hover,
.child-link:hover {
  border-color: var(--graph-link-hover-border);
  background: var(--graph-link-hover-bg);
  color: var(--graph-link-hover-color);
}

.relation-empty,
.panel-empty-copy {
  margin: 0;
  color: var(--graph-empty-copy);
  font-size: 14px;
}

.panel-children {
  margin-top: 16px;
}

.children-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 7px;
  color: var(--graph-muted-label);
  font-size: 13px;
}

.children-heading span:last-child {
  color: var(--graph-count-accent);
  font-variant-numeric: tabular-nums;
}

.children-list {
  display: grid;
  gap: 4px;
}

.child-link {
  display: grid;
  grid-template-columns: 14px minmax(0, 1fr) auto 14px;
  align-items: center;
  gap: 9px;
  width: 100%;
  padding: 8px 9px;
  border: 1px solid transparent;
  border-radius: 4px;
  font-size: 15px;
  text-align: left;
}

.child-icon {
  color: var(--graph-child-icon-color);
}

.child-icon.is-completed {
  color: var(--graph-child-icon-completed);
}

.child-icon.is-overdue {
  color: var(--graph-child-icon-overdue);
}

.child-title {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.child-count {
  color: var(--graph-empty-copy);
  font-size: 12px;
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}

.panel-description {
  margin: 0;
  color: var(--graph-description-color);
  font-size: 15px;
  line-height: 1.8;
  overflow-wrap: anywhere;
  white-space: pre-wrap;
}

.panel-description a {
  color: var(--graph-content-link-color);
  text-decoration: underline;
  text-underline-offset: 3px;
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

.panel-edit-button.is-primary {
  border-color: var(--graph-primary-btn-border);
  background: var(--graph-primary-btn-bg);
  color: var(--graph-primary-btn-color);
}

.panel-edit-button.is-primary:hover {
  border-color: var(--graph-primary-btn-hover-border);
  background: var(--graph-primary-btn-hover-bg);
  color: var(--graph-primary-btn-hover-color);
}

.panel-delete-button {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 5px 9px;
  border: 1px solid var(--graph-danger-border);
  border-radius: 4px;
  background: transparent;
  color: var(--graph-danger-color);
  font-size: 13px;
  cursor: pointer;
}

.panel-delete-button:hover {
  border-color: var(--graph-danger-hover-border);
  background: var(--graph-danger-hover-bg);
  color: var(--graph-danger-hover-color);
}

.panel-edit-button:disabled {
  cursor: wait;
  opacity: 0.55;
}

.panel-loading,
.panel-error {
  display: grid;
  min-height: 220px;
  place-content: center;
  gap: 13px;
  padding: 24px;
  color: var(--graph-panel-muted-color);
  text-align: center;
  font-size: 15px;
}

.panel-loading-mark {
  width: 18px;
  height: 18px;
  margin: auto;
  border: 1px solid rgba(226, 177, 73, 0.25);
  border-top-color: #e2b149;
  border-radius: 50%;
  animation: panel-spin 750ms linear infinite;
}

.panel-error p {
  margin: 0;
  color: var(--graph-panel-error-text);
}

.panel-error button {
  justify-self: center;
  min-height: 32px;
  padding: 6px 12px;
  border: 1px solid var(--graph-panel-error-border);
  border-radius: 4px;
  background: transparent;
  color: var(--graph-panel-error-btn-color);
  font-size: 13px;
  cursor: pointer;
}

.panel-error button:hover {
  border-color: var(--graph-panel-error-btn-hover-border);
  background: var(--graph-panel-error-btn-hover-bg);
  color: var(--graph-panel-error-btn-hover-color);
}

.panel-footer {
  flex: none;
  padding: 14px 18px 13px;
  border-top: 1px solid var(--graph-panel-footer-border);
  background: var(--graph-panel-footer-bg);
}

.panel-page-link {
  display: flex;
  height: 40px;
  align-items: center;
  justify-content: center;
  gap: 8px;
  border: 1px solid var(--graph-page-link-border);
  border-radius: 6px;
  background: var(--graph-page-link-bg);
  color: var(--graph-page-link-color);
  font-size: 14px;
  font-weight: 600;
  text-decoration: none;
}

.panel-page-link:hover {
  border-color: var(--graph-page-link-hover-border);
  background: var(--graph-page-link-hover-bg);
  color: var(--graph-page-link-hover-color);
}

@keyframes panel-spin {
  to {
    transform: rotate(360deg);
  }
}

@media (max-width: 980px) {
  .node-control-panel {
    width: min(380px, calc(100vw - 32px));
  }
}

@media (max-width: 640px) {
  .node-control-panel {
    top: 10px;
    right: 10px;
    bottom: 10px;
    width: calc(100vw - 20px);
  }

  .panel-header,
  .panel-section {
    padding-right: 18px;
    padding-left: 18px;
  }
}

@media (prefers-reduced-motion: reduce) {
  .panel-loading-mark {
    animation: none;
  }

  .node-panel-enter-active,
  .node-panel-leave-active {
    transition-duration: 1ms;
  }
}
</style>
