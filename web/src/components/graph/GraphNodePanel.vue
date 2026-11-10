<script setup lang="ts">
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

const panel = ref<HTMLElement | null>(null);
const title = computed(() => props.detail?.title ?? props.selected.title);
const {
  editingDue,
  dueValue,
  savingDue,
  dueError,
  savingCompletion,
  completionError,
  startDueEdit,
  cancelDueEdit,
  saveDue,
  toggleCompletion,
} = useNodeInlineEditing(
  () => props.detail,
  (node) => emit("updated", node),
);

usePanelTilt(panel);

function requestCreateNode() {
  if (props.isVirtualRoot) {
    emit("createRoot");
  } else {
    emit("createChild");
  }
}
</script>

<template>
  <aside
    ref="panel"
    aria-label="节点控制面板"
    class="node-control-panel"
    data-graph-node-panel
  >
    <header class="panel-header">
      <button
        aria-label="关闭节点面板"
        class="panel-close"
        :disabled="savingDue || savingCompletion"
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
          :disabled="savingCompletion || editingDue || savingDue"
          type="button"
          @click="toggleCompletion"
        >
          {{
            savingCompletion
              ? "正在更新…"
              : detail.completedAt
                ? "标记为未完成"
                : "标记为已完成"
          }}
        </button>
      </div>
      <p v-if="completionError" class="status-error" role="alert">
        {{ completionError }}
      </p>
      <div v-if="detail?.tags.length" class="panel-tag-list">
        <span v-for="tag in detail.tags" :key="tag" class="panel-tag">
          {{ tag }}
        </span>
      </div>
      <div v-if="detail && !isVirtualRoot" class="panel-node-actions">
        <GraphPanelButton
          v-if="detail.permissions.canEdit"
          @click="emit('edit')"
        >
          <Pencil :size="12" />
          编辑节点
        </GraphPanelButton>
        <GraphPanelButton
          v-if="detail.permissions.canEdit"
          @click="emit('move')"
        >
          <FolderInput :size="12" />
          移动节点
        </GraphPanelButton>
        <GraphPanelButton
          v-if="detail.permissions.canDelete"
          tone="danger"
          @click="emit('remove')"
        >
          <Trash2 :size="12" />
          删除节点
        </GraphPanelButton>
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
        <GraphNodeStructure
          :is-virtual-root
          :node="detail"
          :parent-id
          :parent-title
          @create="requestCreateNode"
          @navigate="emit('navigate', $event)"
          @reorder="emit('reorder')"
        />

        <GraphPanelSection v-if="!isVirtualRoot" title="节点描述">
          <p v-if="detail.description" class="panel-description">
            {{ detail.description }}
          </p>
          <p v-else class="panel-empty-copy">暂无节点描述</p>
        </GraphPanelSection>

        <GraphPanelSection v-if="!isVirtualRoot" title="节点信息">
          <template #actions>
            <GraphPanelButton
              v-if="detail.permissions.canEdit && !editingDue"
              :disabled="savingCompletion"
              @click="startDueEdit"
            >
              <Pencil :size="12" />
              修改预期完成
            </GraphPanelButton>
          </template>

          <div v-if="editingDue" class="meta-editor">
            <label class="meta-editor-field">
              <span>预期完成时间</span>
              <NDatePicker
                v-model:value="dueValue"
                clearable
                :disabled="savingDue"
                format="yyyy-MM-dd HH:mm"
                :time-picker-props="{ format: 'HH:mm' }"
                type="datetime"
              />
            </label>
            <NAlert v-if="dueError" role="alert" type="error">
              {{ dueError }}
            </NAlert>
            <div class="meta-editor-actions">
              <button
                :disabled="savingDue"
                type="button"
                @click="cancelDueEdit"
              >
                取消
              </button>
              <button
                class="is-primary"
                :disabled="savingDue"
                type="button"
                @click="saveDue"
              >
                {{ savingDue ? "保存中…" : "保存修改" }}
              </button>
            </div>
          </div>

          <dl v-else class="panel-meta-grid">
            <div>
              <dt>
                <CalendarClock :size="13" />
                预期完成
              </dt>
              <dd>{{ formatDateTime(detail.dueAt) }}</dd>
            </div>
            <div>
              <dt>创建时间</dt>
              <dd>{{ formatDateTime(detail.createdAt) }}</dd>
            </div>
            <div>
              <dt>最近更新</dt>
              <dd :title="formatDateTime(detail.updatedAt)">
                {{ timeAgo(detail.updatedAt) }}
              </dd>
            </div>
            <div>
              <dt>下级节点</dt>
              <dd>{{ detail.subtreeNodeCount }}</dd>
            </div>
            <div>
              <dt>有效成员</dt>
              <dd>{{ detail.effectiveMemberCount }}</dd>
            </div>
            <div>
              <dt>进展</dt>
              <dd>{{ detail.subtreeCommentCount }}</dd>
            </div>
          </dl>
        </GraphPanelSection>

        <GraphNodePeople
          v-if="!isVirtualRoot"
          :key="detail.id"
          :node="detail"
          @updated="emit('updated', $event)"
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
  </aside>
</template>

<style scoped>
.node-control-panel {
  --panel-base-rotate-y: -2.4deg;
  --panel-tilt-x: 0deg;
  --panel-tilt-y: 0deg;
  position: absolute;
  z-index: 20;
  top: 16px;
  right: 16px;
  bottom: 16px;
  display: flex;
  width: clamp(330px, 36vw, 620px);
  min-height: 0;
  flex-direction: column;
  overflow: hidden;
  border: 1px solid rgba(222, 174, 69, 0.34);
  border-right-color: rgba(240, 187, 78, 0.68);
  border-radius: 4px;
  background:
    linear-gradient(155deg, rgba(25, 24, 17, 0.9), transparent 34%),
    linear-gradient(180deg, rgba(10, 17, 20, 0.97), rgba(5, 10, 13, 0.98));
  box-shadow:
    -24px 0 70px rgba(0, 0, 0, 0.42),
    inset 0 0 50px rgba(225, 173, 70, 0.025);
  color: #d9e1e1;
  pointer-events: auto;
  backdrop-filter: blur(22px);
  transform: perspective(1500px) rotateX(var(--panel-tilt-x))
    rotateY(calc(var(--panel-base-rotate-y) + var(--panel-tilt-y)));
  transform-origin: 100% 50%;
  transform-style: preserve-3d;
  will-change: transform, opacity;
}

.node-panel-enter-active {
  transition:
    transform 360ms cubic-bezier(0.22, 0.76, 0.24, 1),
    opacity 240ms ease-out;
}

.node-panel-leave-active {
  transition:
    transform 300ms cubic-bezier(0.55, 0, 0.8, 0.42),
    opacity 220ms ease-in;
}

.node-panel-enter-from,
.node-panel-leave-to {
  opacity: 0;
  transform: perspective(1500px) rotateX(var(--panel-tilt-x))
    rotateY(calc(var(--panel-base-rotate-y) + var(--panel-tilt-y)))
    translate3d(calc(100% + 32px), 0, -20px);
}

.node-panel-enter-to,
.node-panel-leave-from {
  opacity: 1;
  transform: perspective(1500px) rotateX(var(--panel-tilt-x))
    rotateY(calc(var(--panel-base-rotate-y) + var(--panel-tilt-y)))
    translate3d(0, 0, 0);
}

.node-control-panel::before {
  position: absolute;
  top: 0;
  right: 0;
  left: 0;
  height: 1px;
  background: linear-gradient(
    90deg,
    transparent,
    rgba(244, 192, 82, 0.78),
    transparent
  );
  content: "";
}

.panel-header {
  position: relative;
  flex: none;
  padding: 22px 24px 18px;
  border-bottom: 1px solid rgba(139, 164, 169, 0.16);
  background: rgba(8, 14, 17, 0.58);
}

.panel-close {
  position: absolute;
  top: 16px;
  right: 16px;
  display: grid;
  width: 30px;
  height: 30px;
  place-items: center;
  border: 1px solid rgba(150, 171, 175, 0.22);
  border-radius: 2px;
  background: rgba(10, 16, 19, 0.72);
  color: #a5b4b7;
  cursor: pointer;
}

.panel-close:hover:not(:disabled) {
  border-color: rgba(238, 185, 77, 0.62);
  color: #f2c66f;
}

.panel-close:disabled {
  cursor: wait;
  opacity: 0.55;
}

.panel-header h2 {
  margin: 0 40px 0 0;
  color: #f4f0e6;
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
  border: 1px solid rgba(221, 176, 75, 0.34);
  border-radius: 2px;
  background: rgba(218, 170, 67, 0.07);
  color: #e5c47c;
  font-size: 13px;
  cursor: pointer;
}

.status-toggle:hover {
  border-color: rgba(241, 190, 80, 0.72);
  color: #ffe2a3;
}

.status-toggle:disabled {
  cursor: wait;
  opacity: 0.58;
}

.status-error {
  margin: 8px 0 0;
  color: #ef9c83;
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
  border: 1px solid rgba(116, 150, 157, 0.28);
  border-radius: 2px;
  color: #9eb2b6;
  font-size: 13px;
}

.panel-node-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 12px;
}

.panel-scroll {
  min-height: 0;
  flex: 1 1 auto;
  overflow-y: auto;
  overscroll-behavior: contain;
  scrollbar-color: rgba(221, 176, 75, 0.45) transparent;
  scrollbar-width: thin;
}

.panel-empty-copy {
  margin: 0;
  color: #71868b;
  font-size: 14px;
}

.panel-description {
  margin: 0;
  color: #aebbbc;
  font-size: 16px;
  line-height: 1.8;
  overflow-wrap: anywhere;
  white-space: pre-wrap;
}

.panel-meta-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 13px 18px;
  margin: 0;
}

.panel-meta-grid div {
  min-width: 0;
}

.panel-meta-grid dt {
  display: flex;
  align-items: center;
  gap: 5px;
  margin-bottom: 4px;
  color: #72878b;
  font-size: 13px;
}

.panel-meta-grid dd {
  margin: 0;
  color: #cad4d4;
  font-size: 15px;
  font-variant-numeric: tabular-nums;
  overflow-wrap: anywhere;
}

.meta-editor {
  display: grid;
  gap: 14px;
}

.meta-editor-field {
  display: grid;
  gap: 7px;
  color: #91a4a8;
  font-size: 14px;
}

.meta-editor-field :deep(.n-date-picker),
.meta-editor-field :deep(.n-input) {
  width: 100%;
}

.meta-editor-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  padding-top: 4px;
}

.meta-editor-actions button {
  min-height: 32px;
  padding: 6px 12px;
  border: 1px solid rgba(139, 164, 169, 0.28);
  border-radius: 2px;
  background: rgba(11, 18, 21, 0.72);
  color: #bdcbcc;
  font-size: 13px;
  cursor: pointer;
}

.meta-editor-actions button.is-primary {
  border-color: rgba(232, 181, 75, 0.62);
  background: rgba(218, 170, 67, 0.12);
  color: #f1cb7b;
}

.meta-editor-actions button:disabled {
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
  color: #91a5a8;
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
  color: #e0b661;
}

.panel-error button {
  justify-self: center;
  min-height: 32px;
  padding: 6px 12px;
  border: 1px solid rgba(230, 180, 75, 0.35);
  background: transparent;
  color: #e5c077;
  font-size: 13px;
  cursor: pointer;
}

.panel-footer {
  flex: none;
  padding: 14px 18px 13px;
  border-top: 1px solid rgba(222, 174, 69, 0.22);
  background: rgba(7, 12, 15, 0.92);
}

.panel-page-link {
  display: flex;
  height: 40px;
  align-items: center;
  justify-content: center;
  gap: 8px;
  border: 1px solid rgba(231, 181, 76, 0.52);
  border-radius: 2px;
  background:
    linear-gradient(90deg, rgba(231, 181, 76, 0.16), transparent),
    rgba(24, 23, 17, 0.8);
  color: #f0ca79;
  font-size: 14px;
  font-weight: 600;
  text-decoration: none;
}

.panel-page-link:hover {
  border-color: rgba(246, 198, 92, 0.86);
  color: #fff0c7;
}

[data-theme="light"] .node-control-panel {
  border-color: rgba(157, 105, 25, 0.28);
  border-right-color: rgba(151, 94, 12, 0.5);
  background:
    linear-gradient(155deg, rgba(255, 249, 235, 0.92), transparent 34%),
    linear-gradient(
      180deg,
      rgba(255, 255, 255, 0.97),
      rgba(246, 249, 250, 0.98)
    );
  box-shadow:
    -24px 0 70px rgba(51, 66, 75, 0.16),
    inset 0 0 50px rgba(167, 107, 17, 0.02);
  color: #263640;
}

[data-theme="light"] .panel-header {
  border-bottom-color: rgba(81, 105, 116, 0.15);
  background: rgba(255, 255, 255, 0.62);
}

[data-theme="light"] .panel-header h2 {
  color: #17242d;
}

[data-theme="light"] .panel-close {
  border-color: rgba(84, 107, 117, 0.22);
  background: rgba(255, 255, 255, 0.76);
  color: #5c7079;
}

[data-theme="light"] .status-toggle {
  border-color: rgba(155, 101, 17, 0.3);
  background: rgba(184, 119, 21, 0.06);
  color: #80510d;
}

[data-theme="light"] .panel-tag {
  border-color: rgba(87, 119, 132, 0.26);
  color: #607783;
}

[data-theme="light"] .panel-meta-grid dt {
  color: #607580;
}

[data-theme="light"] .panel-empty-copy {
  color: #758993;
}

[data-theme="light"] .panel-description,
[data-theme="light"] .panel-meta-grid dd {
  color: #344650;
}

[data-theme="light"] .meta-editor-field,
[data-theme="light"] .panel-loading,
[data-theme="light"] .panel-error {
  color: #627781;
}

[data-theme="light"] .meta-editor-actions button {
  border-color: rgba(81, 106, 118, 0.25);
  background: rgba(255, 255, 255, 0.78);
  color: #4f6570;
}

[data-theme="light"] .meta-editor-actions button.is-primary {
  border-color: rgba(156, 101, 17, 0.46);
  background: rgba(187, 123, 24, 0.09);
  color: #7d4e0a;
}

[data-theme="light"] .panel-error p {
  color: #8d590d;
}

[data-theme="light"] .panel-error button {
  border-color: rgba(156, 101, 17, 0.34);
  color: #7f500b;
}

[data-theme="light"] .panel-footer {
  border-top-color: rgba(157, 105, 22, 0.2);
  background: rgba(255, 255, 255, 0.9);
}

[data-theme="light"] .panel-page-link {
  border-color: rgba(157, 102, 18, 0.38);
  background:
    linear-gradient(90deg, rgba(190, 127, 29, 0.1), transparent),
    rgba(255, 252, 246, 0.86);
  color: #81510b;
}

@keyframes panel-spin {
  to {
    transform: rotate(360deg);
  }
}

@media (max-width: 980px) {
  .node-control-panel {
    width: min(460px, calc(100vw - 32px));
  }
}

@media (max-width: 640px) {
  .node-control-panel {
    top: 10px;
    right: 10px;
    bottom: 10px;
    width: calc(100vw - 20px);
  }

  .panel-header {
    padding-right: 18px;
    padding-left: 18px;
  }

  .panel-meta-grid {
    grid-template-columns: minmax(0, 1fr);
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
