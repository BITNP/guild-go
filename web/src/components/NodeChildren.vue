<script setup lang="ts">
const emit = defineEmits<{ reorder: []; create: [] }>();
defineProps<{ node: NodeDetail }>();
</script>

<template>
  <section aria-label="子节点" class="node-children">
    <SectionHeading :count="node.children.length" title="子节点">
      <NButton
        v-if="node.permissions.canReorderChildren && node.children.length > 1"
        size="small"
        @click="emit('reorder')"
      >
        调整顺序
      </NButton>
      <NButton
        v-if="node.permissions.canAddChild"
        size="small"
        type="primary"
        @click="emit('create')"
      >
        新建子节点
      </NButton>
    </SectionHeading>
    <NCard size="small">
      <NList v-if="node.children.length > 0">
        <NodeListItem
          v-for="child in node.children"
          :key="child.id"
          :node="child"
        />
      </NList>
      <NEmpty v-else description="暂无子节点" size="small" />
    </NCard>
  </section>
</template>

<style scoped>
.node-children {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
</style>
