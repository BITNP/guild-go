<script setup lang="ts">
import {
  computed,
  onBeforeUnmount,
  onMounted,
  ref,
  watch,
} from 'vue'
import { RouterLink } from 'vue-router'
import {
  NAlert,
  NDatePicker,
} from 'naive-ui'
import {
  ArrowUpDown,
  CalendarClock,
  ChevronRight,
  ExternalLink,
  Network,
  Pencil,
  Plus,
  Trash2,
  UserRound,
  X,
} from '@lucide/vue'
import MembersDialog from '@/components/MembersDialog.vue'
import NodeStatusTag from '@/components/NodeStatusTag.vue'
import UserIdentity from '@/components/UserIdentity.vue'
import type { GraphNode } from '@/graph/types'
import { updateNode, type NodeDetail } from '@/lib/api'
import { errorMessage, formatDateTime, timeAgo } from '@/lib/utils'

const props = defineProps<{
  selected: GraphNode
  detail: NodeDetail | null
  parentId: number | null
  parentTitle: string | null
  loading: boolean
  error: string | null
  isVirtualRoot: boolean
}>()

const emit = defineEmits<{
  close: []
  navigate: [id: number]
  retry: []
  updated: [node: NodeDetail]
  edit: []
  createChild: []
  createRoot: []
  remove: []
  reorder: []
}>()

const membersOpen = ref(false)
const editingMeta = ref(false)
const quickDue = ref<number | null>(null)
const savingMeta = ref(false)
const metaError = ref<string | null>(null)
const savingStatus = ref(false)
const statusError = ref<string | null>(null)
const panel = ref<HTMLElement | null>(null)
const title = computed(() => props.detail?.title ?? props.selected.title)
let pointerTargetX = 0
let pointerTargetY = 0
let pointerX = 0
let pointerY = 0
let tiltFrame = 0
let lastTiltFrame = performance.now()
let reduceMotion = false

function updatePointerTilt(event: PointerEvent) {
  pointerTargetX = (event.clientX / Math.max(window.innerWidth, 1)) * 2 - 1
  pointerTargetY = (event.clientY / Math.max(window.innerHeight, 1)) * 2 - 1
}

function animateTilt(now: number) {
  const delta = Math.min(0.05, (now - lastTiltFrame) / 1000)
  lastTiltFrame = now
  const damping = 1 - Math.exp(-delta * 5)
  pointerX += (pointerTargetX - pointerX) * damping
  pointerY += (pointerTargetY - pointerY) * damping
  panel.value?.style.setProperty('--panel-tilt-x', `${pointerY * 0.8}deg`)
  panel.value?.style.setProperty('--panel-tilt-y', `${-pointerX * 1.15}deg`)
  tiltFrame = requestAnimationFrame(animateTilt)
}

onMounted(() => {
  reduceMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches
  window.addEventListener('pointermove', updatePointerTilt, { passive: true })
  if (!reduceMotion) {
    lastTiltFrame = performance.now()
    tiltFrame = requestAnimationFrame(animateTilt)
  }
})

onBeforeUnmount(() => {
  window.removeEventListener('pointermove', updatePointerTilt)
  cancelAnimationFrame(tiltFrame)
})

watch(
  () => props.selected.id,
  () => {
    membersOpen.value = false
    editingMeta.value = false
    metaError.value = null
    statusError.value = null
  },
)

function onMembersSaved(node: NodeDetail) {
  membersOpen.value = false
  emit('updated', node)
}

function createNode() {
  if (props.isVirtualRoot) {
    emit('createRoot')
  } else {
    emit('createChild')
  }
}

function startMetaEdit() {
  if (!props.detail || savingStatus.value) return
  quickDue.value = props.detail.dueAt
    ? new Date(props.detail.dueAt).getTime()
    : null
  metaError.value = null
  editingMeta.value = true
}

function cancelMetaEdit() {
  if (savingMeta.value) return
  editingMeta.value = false
  metaError.value = null
}

async function saveMetaEdit() {
  const node = props.detail
  if (!node || savingMeta.value) return
  savingMeta.value = true
  metaError.value = null
  try {
    const updated = await updateNode(node.id, {
      title: node.title,
      description: node.description,
      dueAt: quickDue.value === null
        ? null
        : new Date(quickDue.value).toISOString(),
      tags: node.tags,
      completed: !!node.completedAt,
    })
    editingMeta.value = false
    emit('updated', updated)
  } catch (err) {
    metaError.value = errorMessage(err, '保存到期时间失败')
  } finally {
    savingMeta.value = false
  }
}

async function toggleCompletion() {
  const node = props.detail
  if (
    !node ||
    !node.permissions.canEdit ||
    savingStatus.value ||
    editingMeta.value ||
    savingMeta.value
  ) {
    return
  }
  savingStatus.value = true
  statusError.value = null
  try {
    const updated = await updateNode(node.id, {
      title: node.title,
      description: node.description,
      dueAt: node.dueAt,
      tags: node.tags,
      completed: !node.completedAt,
    })
    emit('updated', updated)
  } catch (err) {
    statusError.value = errorMessage(err, '更新完成状态失败')
  } finally {
    savingStatus.value = false
  }
}
</script>

<template>
  <aside
    ref="panel"
    class="node-control-panel"
    data-graph-node-panel
    aria-label="节点控制面板"
  >
    <header class="panel-header">
      <div class="panel-kicker">
        <Network :size="13" />
        <span>NODE CONTROL</span>
      </div>
      <button
        type="button"
        class="panel-close"
        aria-label="关闭节点控制面板"
        @click="emit('close')"
      >
        <X :size="17" />
      </button>
      <h2>{{ title }}</h2>
      <div v-if="detail && !isVirtualRoot" class="panel-status-row">
        <NodeStatusTag
          :due-at="detail.dueAt"
          :completed-at="detail.completedAt"
        />
        <button
          v-if="detail.permissions.canEdit"
          type="button"
          class="status-toggle"
          :disabled="savingStatus || editingMeta || savingMeta"
          :aria-pressed="!!detail.completedAt"
          @click="toggleCompletion"
        >
          {{
            savingStatus
              ? '正在更新…'
              : detail.completedAt
                ? '标记为未完成'
                : '标记为已完成'
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
          type="button"
          class="panel-edit-button"
          @click="emit('edit')"
        >
          <Pencil :size="12" />
          编辑节点
        </button>
        <button
          v-if="detail.permissions.canDelete"
          type="button"
          class="panel-delete-button"
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
        class="panel-loading"
        aria-live="polite"
      >
        <span class="panel-loading-mark" />
        正在连接到开拓者网络……
      </div>

      <div v-else-if="error && !detail" class="panel-error" role="alert">
        <p>{{ error }}</p>
        <button type="button" @click="emit('retry')">重新加载</button>
      </div>

      <template v-else-if="detail">
        <section class="panel-section" aria-labelledby="panel-structure-title">
          <h3 id="panel-structure-title">
            {{ isVirtualRoot ? '始祖节点' : '层级关系' }}
          </h3>
          <div v-if="!isVirtualRoot" class="panel-relation">
            <span class="relation-label">父节点</span>
            <button
              v-if="parentId != null"
              type="button"
              class="relation-link"
              @click="emit('navigate', parentId)"
            >
              <span>{{ parentTitle ?? `节点 #${parentId}` }}</span>
              <ChevronRight :size="14" />
            </button>
            <span v-else class="relation-empty">当前为根节点</span>
          </div>

          <div class="panel-children">
            <div class="children-heading">
              <span>{{ isVirtualRoot ? '根节点' : '子节点' }}</span>
              <span>{{ detail.children.length }}</span>
            </div>
            <div class="children-actions">
              <button
                v-if="
                  detail.permissions.canReorderChildren &&
                  detail.children.length > 1
                "
                type="button"
                class="panel-edit-button"
                @click="emit('reorder')"
              >
                <ArrowUpDown :size="12" />
                {{ isVirtualRoot ? '调整根节点顺序' : '调整子节点顺序' }}
              </button>
              <button
                v-if="detail.permissions.canAddChild"
                type="button"
                class="panel-edit-button is-primary"
                @click="createNode"
              >
                <Plus :size="12" />
                {{ isVirtualRoot ? '新建根节点' : '新建子节点' }}
              </button>
            </div>
            <div v-if="detail.children.length" class="children-list">
              <button
                v-for="child in detail.children"
                :key="child.id"
                type="button"
                class="child-link"
                @click="emit('navigate', child.id)"
              >
                <span class="child-orb" aria-hidden="true" />
                <span class="child-title">{{ child.title }}</span>
                <span class="child-count">{{ child.childCount }}</span>
                <ChevronRight :size="13" />
              </button>
            </div>
            <p v-else class="relation-empty">
              {{ isVirtualRoot ? '暂无根节点' : '暂无子节点' }}
            </p>
          </div>
        </section>

        <section
          v-if="!isVirtualRoot"
          class="panel-section"
          aria-labelledby="panel-description-title"
        >
          <h3 id="panel-description-title">节点描述</h3>
          <p v-if="detail.description" class="panel-description">
            {{ detail.description }}
          </p>
          <p v-else class="panel-empty-copy">暂无节点描述</p>
        </section>

        <section
          v-if="!isVirtualRoot"
          class="panel-section"
          aria-labelledby="panel-meta-title"
        >
          <div class="panel-section-heading">
            <h3 id="panel-meta-title">节点信息</h3>
            <button
              v-if="detail.permissions.canEdit && !editingMeta"
              type="button"
              class="panel-edit-button"
              :disabled="savingStatus"
              @click="startMetaEdit"
            >
              <Pencil :size="12" />
              编辑到期时间
            </button>
          </div>

          <div v-if="editingMeta" class="meta-editor">
            <label class="meta-editor-field">
              <span>预期完成时间</span>
              <NDatePicker
                v-model:value="quickDue"
                type="datetime"
                format="yyyy-MM-dd HH:mm"
                :time-picker-props="{ format: 'HH:mm' }"
                clearable
                :disabled="savingMeta"
              />
            </label>
            <NAlert v-if="metaError" type="error" role="alert">
              {{ metaError }}
            </NAlert>
            <div class="meta-editor-actions">
              <button
                type="button"
                :disabled="savingMeta"
                @click="cancelMetaEdit"
              >
                取消
              </button>
              <button
                type="button"
                class="is-primary"
                :disabled="savingMeta"
                @click="saveMetaEdit"
              >
                {{ savingMeta ? '保存中…' : '保存修改' }}
              </button>
            </div>
          </div>

          <dl v-else class="panel-meta-grid">
            <div>
              <dt><CalendarClock :size="13" />到期时间</dt>
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
        </section>

        <section
          v-if="!isVirtualRoot"
          class="panel-section panel-people"
          aria-labelledby="panel-people-title"
        >
          <div class="people-heading">
            <h3 id="panel-people-title">负责人 / 参与者</h3>
            <button
              v-if="detail.permissions.canManageMembers"
              type="button"
              class="panel-edit-button"
              @click="membersOpen = true"
            >
              <Pencil :size="12" />
              编辑成员
            </button>
          </div>

          <div class="people-group">
            <h4>直接负责人</h4>
            <ul v-if="detail.directLeaders.length" class="people-list">
              <li v-for="person in detail.directLeaders" :key="person.subject">
                <UserIdentity :user="person" :size="24" />
              </li>
            </ul>
            <p v-else class="panel-empty-copy">暂无直接负责人</p>
          </div>

          <div class="people-group">
            <h4>直接参与者</h4>
            <ul v-if="detail.directMembers.length" class="people-list">
              <li v-for="person in detail.directMembers" :key="person.subject">
                <UserIdentity :user="person" :size="24" />
              </li>
            </ul>
            <p v-else class="panel-empty-copy">暂无直接参与者</p>
          </div>

          <p
            v-if="detail.inheritedLeaderCount || detail.inheritedMemberCount"
            class="inherited-count"
          >
            另有 {{ detail.inheritedLeaderCount }} 位上级负责人、
            {{ detail.inheritedMemberCount }} 位继承成员。
          </p>
        </section>
      </template>
    </div>

    <footer class="panel-footer">
      <RouterLink
        v-if="!isVirtualRoot"
        :to="{ name: 'node', params: { id: selected.id } }"
        class="panel-page-link"
      >
        <span>打开完整节点页面</span>
        <ExternalLink :size="15" />
      </RouterLink>
      <p>
        <UserRound :size="12" />
        {{
          isVirtualRoot
            ? '始祖节点汇集所有根节点，并从这里扩展新的分支'
            : '节点详情与成员变更由普通页面同步管理'
        }}
      </p>
    </footer>

    <MembersDialog
      v-if="detail && !isVirtualRoot"
      :open="membersOpen"
      :node="detail"
      @close="membersOpen = false"
      @saved="onMembersSaved"
    />
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
  transform:
    perspective(1500px)
    rotateX(var(--panel-tilt-x))
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
  transform:
    perspective(1500px)
    rotateX(var(--panel-tilt-x))
    rotateY(calc(var(--panel-base-rotate-y) + var(--panel-tilt-y)))
    translate3d(calc(100% + 32px), 0, -20px);
}

.node-panel-enter-to,
.node-panel-leave-from {
  opacity: 1;
  transform:
    perspective(1500px)
    rotateX(var(--panel-tilt-x))
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
  content: '';
}

.panel-header {
  position: relative;
  flex: none;
  padding: 22px 24px 18px;
  border-bottom: 1px solid rgba(139, 164, 169, 0.16);
  background: rgba(8, 14, 17, 0.58);
}

.panel-kicker {
  display: flex;
  align-items: center;
  gap: 7px;
  color: rgba(230, 181, 79, 0.72);
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0.18em;
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

.panel-close:hover {
  border-color: rgba(238, 185, 77, 0.62);
  color: #f2c66f;
}

.panel-header h2 {
  margin: 18px 40px 0 0;
  color: #f4f0e6;
  font-size: clamp(22px, 1.85vw, 31px);
  font-weight: 650;
  line-height: 1.35;
  letter-spacing: 0.015em;
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
  border-bottom: 1px solid rgba(139, 164, 169, 0.13);
}

.panel-section h3,
.people-heading h3 {
  margin: 0 0 13px;
  color: #e7c47b;
  font-size: 14px;
  font-weight: 700;
  letter-spacing: 0.12em;
  text-transform: uppercase;
}

.panel-section-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  margin-bottom: 13px;
}

.panel-section-heading h3 {
  margin: 0;
}

.panel-relation {
  display: grid;
  grid-template-columns: 64px minmax(0, 1fr);
  align-items: center;
  gap: 10px;
}

.relation-label {
  color: #84979b;
  font-size: 13px;
}

.relation-link,
.child-link {
  border: 0;
  background: transparent;
  color: inherit;
  cursor: pointer;
}

.relation-link {
  display: flex;
  min-width: 0;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 8px 10px;
  border: 1px solid rgba(221, 176, 75, 0.22);
  border-radius: 2px;
  background: rgba(218, 170, 67, 0.055);
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
  border-color: rgba(238, 185, 77, 0.56);
  background: rgba(221, 174, 70, 0.09);
  color: #f5d38e;
}

.relation-empty,
.panel-empty-copy {
  margin: 0;
  color: #71868b;
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
  color: #84979b;
  font-size: 13px;
}

.children-heading span:last-child {
  color: #d8b35f;
  font-variant-numeric: tabular-nums;
}

.children-list {
  display: grid;
  gap: 4px;
}

.child-link {
  display: grid;
  grid-template-columns: 7px minmax(0, 1fr) auto 14px;
  align-items: center;
  gap: 9px;
  width: 100%;
  padding: 8px 9px;
  border: 1px solid transparent;
  border-radius: 2px;
  color: #c6d1d2;
  font-size: 15px;
  text-align: left;
}

.child-orb {
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: #d9aa49;
  box-shadow: 0 0 8px rgba(227, 176, 67, 0.72);
}

.child-title {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.child-count {
  color: #70858a;
  font-size: 12px;
  font-variant-numeric: tabular-nums;
}

.child-count::after {
  margin-left: 2px;
  content: '子项';
  font-size: 11px;
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

.people-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 13px;
}

.people-heading h3 {
  margin: 0;
}

.panel-edit-button {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 5px 9px;
  border: 1px solid rgba(221, 176, 75, 0.3);
  border-radius: 2px;
  background: transparent;
  color: #d9b96f;
  font-size: 13px;
  cursor: pointer;
}

.panel-edit-button:hover {
  border-color: rgba(240, 187, 78, 0.68);
  color: #ffe0a2;
}

.panel-edit-button.is-primary {
  border-color: rgba(232, 181, 75, 0.58);
  background: rgba(218, 170, 67, 0.1);
  color: #f0cb7e;
}

.panel-delete-button {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 5px 9px;
  border: 1px solid rgba(226, 125, 104, 0.34);
  border-radius: 2px;
  background: transparent;
  color: #e3a091;
  font-size: 13px;
  cursor: pointer;
}

.panel-delete-button:hover {
  border-color: rgba(241, 143, 119, 0.7);
  background: rgba(177, 69, 50, 0.12);
  color: #ffb9a8;
}

.panel-edit-button:disabled {
  cursor: wait;
  opacity: 0.55;
}

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

.inherited-count {
  margin: 15px 0 0;
  color: #788c90;
  font-size: 13px;
  line-height: 1.6;
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
  letter-spacing: 0.04em;
  text-decoration: none;
}

.panel-page-link:hover {
  border-color: rgba(246, 198, 92, 0.86);
  color: #fff0c7;
}

.panel-footer p {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 5px;
  margin: 8px 0 0;
  color: #61767b;
  font-size: 12px;
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

  .panel-header,
  .panel-section {
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
