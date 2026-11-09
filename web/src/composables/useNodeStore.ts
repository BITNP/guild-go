import type { NodeSummary } from "@/api";

const nodes = ref<NodeSummary[]>([]);
const loading = ref(false);
const error = ref<string | null>(null);
let inflight: Promise<void> | null = null;
let generation = 0;

const byId = computed(() => {
  const map = new Map<number, NodeSummary>();
  for (const node of nodes.value) {
    map.set(node.id, node);
  }

  return map;
});

const roots = computed(() =>
  nodes.value
    .filter((node) => node.parentId == null)
    .sort((a, b) => a.sortOrder - b.sortOrder || a.id - b.id),
);

const childrenOf = (parentId: number): NodeSummary[] =>
  nodes.value
    .filter((node) => node.parentId === parentId)
    .sort((a, b) => a.sortOrder - b.sortOrder || a.id - b.id);

function removeSubtree(rootId: number) {
  const removed = new Set<number>([rootId]);
  let changed = true;
  while (changed) {
    changed = false;
    for (const node of nodes.value) {
      if (
        node.parentId != null &&
        removed.has(node.parentId) &&
        !removed.has(node.id)
      ) {
        removed.add(node.id);
        changed = true;
      }
    }
  }
  nodes.value = nodes.value.filter((node) => !removed.has(node.id));
}

async function load(force = false): Promise<void> {
  if (inflight && !force) {
    return inflight;
  }
  const current = ++generation;
  const request = (async () => {
    loading.value = true;
    error.value = null;
    try {
      const result = await fetchNodes();
      if (current === generation) {
        nodes.value = result;
        error.value = null;
      }
    } catch (err) {
      if (current === generation) {
        error.value = err instanceof Error ? err.message : "无法加载节点树";
      }
    } finally {
      if (current === generation) {
        loading.value = false;
        inflight = null;
      }
    }
  })();
  inflight = request;

  return request;
}

async function reload(): Promise<void> {
  await load(true);
}

export const useNodeStore = () => ({
  nodes,
  byId,
  roots,
  loading,
  error,
  load,
  reload,
  childrenOf,
  removeSubtree,
});
