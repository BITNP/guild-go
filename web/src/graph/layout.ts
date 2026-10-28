import type { GraphNode } from '@/graph/types'

export const VIRTUAL_ROOT_ID = -1
export const NODE_WIDTH = 4.8
export const LEVEL_DEPTH = 3.7

const FIRST_RING_RADIUS = 9.2
const RING_GAP = 5.8
const MIN_NODE_ARC = 4.8

export interface GraphPosition {
  x: number
  z: number
  depth: number
  angle: number
  radius: number
}

export interface GraphEdgeLayout {
  id: string
  parentId: number
  childId: number
}

export interface GraphLayout {
  positions: Map<number, GraphPosition>
  edges: GraphEdgeLayout[]
  centerX: number
  centerZ: number
  width: number
  depth: number
  radius: number
  ringCount: number
}

/**
 * Places each tree depth on its own concentric ring around the guild root.
 *
 * Each parent owns an angular sector and is centered in it. Its descendants
 * share that sector on the next ring, keeping every child near its parent.
 * Ring radii grow until the actual angular gaps can hold every node at the
 * minimum arc distance, so uneven branches cannot cause node collisions.
 */
export function layoutGraph(nodes: GraphNode[]): GraphLayout {
  // GraphScene supplies the synthetic root as a renderable node. It must not
  // participate as its own child in the tree layout.
  const actualNodes = nodes.filter((node) => node.id !== VIRTUAL_ROOT_ID)
  const byId = new Map(actualNodes.map((node) => [node.id, node]))
  const children = new Map<number, GraphNode[]>()

  for (const node of actualNodes) {
    const parentId = safeParentId(node, byId)
    const siblings = children.get(parentId) ?? []
    siblings.push(node)
    children.set(parentId, siblings)
  }
  for (const siblings of children.values()) {
    siblings.sort((a, b) => a.sortOrder - b.sortOrder || a.id - b.id)
  }

  const roots = children.get(VIRTUAL_ROOT_ID) ?? []
  const weights = new Map<number, number>()
  const weighing = new Set<number>()
  const subtreeWeight = (node: GraphNode): number => {
    const known = weights.get(node.id)
    if (known != null) return known
    if (weighing.has(node.id)) return 1
    weighing.add(node.id)
    const descendants = children.get(node.id) ?? []
    const weight = descendants.length === 0
      ? 1
      : Math.max(
          1,
          descendants.reduce((sum, child) => sum + subtreeWeight(child), 0),
        )
    weighing.delete(node.id)
    weights.set(node.id, weight)
    return weight
  }

  interface AngularPlacement {
    depth: number
    angle: number
  }

  const angularPlacements = new Map<number, AngularPlacement>()
  const placeSector = (
    node: GraphNode,
    depth: number,
    start: number,
    end: number,
  ) => {
    if (angularPlacements.has(node.id)) return
    angularPlacements.set(node.id, {
      depth,
      angle: (start + end) / 2,
    })
    const descendants = children.get(node.id) ?? []
    const totalWeight = descendants.reduce((sum, child) => sum + subtreeWeight(child), 0)
    let cursor = start
    for (const child of descendants) {
      const childEnd = cursor + (end - start) * (subtreeWeight(child) / totalWeight)
      placeSector(child, depth + 1, cursor, childEnd)
      cursor = childEnd
    }
  }

  const totalWeight = roots.reduce((sum, root) => sum + subtreeWeight(root), 0)
  let cursor = -Math.PI / 2
  for (const root of roots) {
    const rootEnd = cursor + Math.PI * 2 * (subtreeWeight(root) / totalWeight)
    placeSector(root, 1, cursor, rootEnd)
    cursor = rootEnd
  }
  const positions = new Map<number, GraphPosition>()
  positions.set(VIRTUAL_ROOT_ID, {
    x: 0,
    z: 0,
    depth: 0,
    angle: 0,
    radius: 0,
  })

  let previousRadius = 0
  let maxRadius = 0
  const anglesByDepth = new Map<number, number[]>()
  for (const placement of angularPlacements.values()) {
    const angles = anglesByDepth.get(placement.depth) ?? []
    angles.push(placement.angle)
    anglesByDepth.set(placement.depth, angles)
  }
  const depths = [...anglesByDepth.keys()].sort((a, b) => a - b)
  for (const depth of depths) {
    const angles = (anglesByDepth.get(depth) ?? []).sort((a, b) => a - b)
    const minimumGap = angles.length < 2
      ? Math.PI * 2
      : Math.min(
          ...angles.slice(1).map((angle, index) => angle - angles[index]),
          angles[0] + Math.PI * 2 - angles[angles.length - 1],
        )
    const populationRadius = minimumGap >= Math.PI * 2
      ? FIRST_RING_RADIUS
      : MIN_NODE_ARC / minimumGap
    const radius = Math.max(
      FIRST_RING_RADIUS,
      previousRadius + RING_GAP,
      populationRadius,
    )
    for (const [id, placement] of angularPlacements) {
      if (placement.depth !== depth) continue
      positions.set(id, {
        x: Math.cos(placement.angle) * radius,
        z: Math.sin(placement.angle) * radius,
        depth,
        angle: placement.angle,
        radius,
      })
    }
    previousRadius = radius
    maxRadius = radius
  }

  const edges: GraphEdgeLayout[] = []
  for (const node of actualNodes) {
    const parentId = safeParentId(node, byId)
    edges.push({ id: `${parentId}:${node.id}`, parentId, childId: node.id })
  }

  return {
    positions,
    edges,
    centerX: 0,
    centerZ: 0,
    width: Math.max(NODE_WIDTH, maxRadius * 2 + NODE_WIDTH),
    depth: Math.max(LEVEL_DEPTH, maxRadius * 2),
    radius: maxRadius,
    ringCount: depths.length,
  }
}

function safeParentId(node: GraphNode, byId: Map<number, GraphNode>): number {
  const parentId = node.parentId
  if (parentId == null || parentId === node.id || !byId.has(parentId)) {
    return VIRTUAL_ROOT_ID
  }

  const visited = new Set<number>([node.id])
  let cursor = parentId
  while (cursor !== VIRTUAL_ROOT_ID) {
    if (visited.has(cursor)) return VIRTUAL_ROOT_ID
    visited.add(cursor)
    const parent = byId.get(cursor)
    cursor = parent?.parentId != null &&
      parent.parentId !== parent.id &&
      byId.has(parent.parentId)
      ? parent.parentId
      : VIRTUAL_ROOT_ID
  }
  return parentId
}
