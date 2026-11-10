export interface MoveTargetOption {
  label: string;
  value: number;
  [key: string]: unknown;
}

interface MoveTargetPermissions {
  canMoveAnyNode: boolean;
  canMoveToRoot: boolean;
}

export function buildMoveTargets(
  nodes: NodeSummary[],
  node: Pick<NodeDetail, "id" | "parentId">,
  permissions: MoveTargetPermissions,
): MoveTargetOption[] {
  const byId = new Map(nodes.map((entry) => [entry.id, entry]));
  const children = new Map<number, number[]>();

  for (const entry of nodes) {
    if (entry.parentId == null) {
      continue;
    }
    const siblings = children.get(entry.parentId) ?? [];

    siblings.push(entry.id);
    children.set(entry.parentId, siblings);
  }

  const excluded = new Set<number>();
  const pending = [node.id];

  while (pending.length > 0) {
    const id = pending.pop()!;

    if (excluded.has(id)) {
      continue;
    }
    excluded.add(id);
    pending.push(...(children.get(id) ?? []));
  }

  function path(entry: NodeSummary) {
    const titles: string[] = [];
    const seen = new Set<number>();
    let current: NodeSummary | undefined = entry;

    while (current && !seen.has(current.id)) {
      seen.add(current.id);
      titles.unshift(current.title);
      current =
        current.parentId == null ? undefined : byId.get(current.parentId);
    }

    return titles.join(" / ");
  }

  const result = nodes
    .filter(
      (entry) =>
        !excluded.has(entry.id) &&
        entry.id !== node.parentId &&
        (permissions.canMoveAnyNode || entry.myRelation === "leader"),
    )
    .map((entry) => ({ label: path(entry), value: entry.id }))
    .sort(
      (a, b) => a.label.localeCompare(b.label, "zh-CN") || a.value - b.value,
    );

  if (permissions.canMoveToRoot && node.parentId != null) {
    result.unshift({ label: "总览（移动为根节点）", value: 0 });
  }

  return result;
}
