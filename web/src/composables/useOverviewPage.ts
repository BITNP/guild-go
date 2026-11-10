export function useOverviewPage() {
  const router = useRouter();
  const { isManager } = useAuth();
  const { nodes, error: nodesError, reload: reloadNodes } = useNodeStore();
  const overview = ref<Overview | null>(null);
  const loading = ref(true);
  const error = ref<string | null>(null);
  const createOpen = ref(false);
  const orderOpen = ref(false);
  const statistics = computed(() => [
    { label: "全部节点", value: nodes.value.length },
    {
      label: "我负责",
      value: nodes.value.filter((node) => node.myRelation === "leader").length,
    },
    {
      label: "未完成",
      value: nodes.value.filter((node) => !node.completedAt).length,
    },
    {
      label: "已逾期",
      value: nodes.value.filter((node) =>
        isOverdue(node.dueAt, node.completedAt),
      ).length,
    },
  ]);

  async function load() {
    loading.value = true;
    error.value = null;
    try {
      overview.value = await fetchOverview();
    } catch (err) {
      error.value = errorMessage(err, "无法加载总览");
    } finally {
      loading.value = false;
    }
  }

  async function refresh() {
    await Promise.all([load(), reloadNodes()]);
  }

  async function created(node: NodeDetail) {
    createOpen.value = false;
    await refresh();
    await router.push({ name: "node", params: { id: node.id } });
  }

  async function ordered() {
    orderOpen.value = false;
    await refresh();
  }

  onMounted(load);

  return {
    isManager,
    nodesError,
    overview,
    loading,
    error,
    createOpen,
    orderOpen,
    statistics,
    load,
    refresh,
    created,
    ordered,
  };
}
