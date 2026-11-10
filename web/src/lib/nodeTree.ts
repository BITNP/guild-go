export interface NodeTreeFilters {
  search: string;
  relation: string;
  tag: string | null;
}

export function visibleNodeIds(
  nodes: NodeSummary[],
  byId: ReadonlyMap<number, NodeSummary>,
  filters: NodeTreeFilters,
): Set<number> | null {
  const query = filters.search.trim().toLowerCase();

  if (!query && filters.relation === "all" && !filters.tag) {
    return null;
  }

  const result = new Set<number>();

  for (const node of nodes) {
    if (filters.relation === "leader" && node.myRelation !== "leader") {
      continue;
    }
    if (filters.relation === "member" && !node.myRelation) {
      continue;
    }
    if (filters.tag && !node.tags.includes(filters.tag)) {
      continue;
    }
    if (
      query &&
      !node.title.toLowerCase().includes(query) &&
      !node.tags.some((value) => value.toLowerCase().includes(query))
    ) {
      continue;
    }

    let current: NodeSummary | undefined = node;

    while (current && !result.has(current.id)) {
      result.add(current.id);
      current =
        current.parentId == null ? undefined : byId.get(current.parentId);
    }
  }

  return result;
}
