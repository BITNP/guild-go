<script setup lang="ts">
const emit = defineEmits<{
  navigate: [id: number];
  create: [];
  reorder: [];
}>();

defineProps<{
  node: NodeDetail;
  parentId: number | null;
  parentTitle: string | null;
  isVirtualRoot: boolean;
}>();
</script>

<template>
  <GraphPanelSection :title="isVirtualRoot ? '全部根节点' : '层级关系'">
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
        <span>{{ node.children.length }}</span>
      </div>
      <div class="children-actions">
        <GraphPanelButton
          v-if="node.permissions.canReorderChildren && node.children.length > 1"
          @click="emit('reorder')"
        >
          <ArrowUpDown :size="12" />
          {{ isVirtualRoot ? "调整根节点顺序" : "调整子节点顺序" }}
        </GraphPanelButton>
        <GraphPanelButton
          v-if="node.permissions.canAddChild"
          tone="primary"
          @click="emit('create')"
        >
          <Plus :size="12" />
          {{ isVirtualRoot ? "新建根节点" : "新建子节点" }}
        </GraphPanelButton>
      </div>
      <div v-if="node.children.length > 0" class="children-list">
        <button
          v-for="child in node.children"
          :key="child.id"
          class="child-link"
          type="button"
          @click="emit('navigate', child.id)"
        >
          <span aria-hidden="true" class="child-orb" />
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
  </GraphPanelSection>
</template>

<style scoped>
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
  background: transparent;
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
  color: inherit;
  font-size: 14px;
  text-align: left;
}

.relation-link span,
.child-title {
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

.relation-empty {
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

.children-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin: -3px 0 9px;
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

.child-count {
  color: #70858a;
  font-size: 12px;
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}

[data-theme="light"] .relation-label,
[data-theme="light"] .children-heading {
  color: #607580;
}

[data-theme="light"] .relation-link,
[data-theme="light"] .child-link {
  color: #344650;
}

[data-theme="light"] .relation-link {
  border-color: rgba(157, 104, 21, 0.22);
  background: rgba(183, 120, 24, 0.05);
}

[data-theme="light"] .relation-link:hover,
[data-theme="light"] .child-link:hover {
  border-color: rgba(157, 100, 17, 0.48);
  background: rgba(183, 120, 24, 0.08);
  color: #714707;
}

[data-theme="light"] .relation-empty,
[data-theme="light"] .child-count {
  color: #758993;
}

[data-theme="light"] .children-heading span:last-child {
  color: #936013;
}
</style>
