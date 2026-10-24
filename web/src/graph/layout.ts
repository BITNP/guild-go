import type { GraphNode } from '@/lib/api'

export const VIRTUAL_ROOT_ID = -1
export const NODE_WIDTH = 4.8
export const CELL_WIDTH = 6.16
export const LEVEL_DEPTH = 3.7

export interface GraphPosition {
  x: number
  z: number
  depth: number
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
}

interface MutablePosition extends GraphPosition {
  width: number
}

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

  const positions = new Map<number, MutablePosition>()
  const measuring = new Set<number>()
  const measure = (id: number): number => {
    if (measuring.has(id)) return 1
    measuring.add(id)
    const descendants = children.get(id) ?? []
    const width = descendants.length === 0
      ? 1
      : descendants.reduce((sum, child) => sum + measure(child.id), 0)
    const current = positions.get(id) ?? { x: 0, z: 0, depth: 0, width: 0 }
    current.width = Math.max(1, width)
    positions.set(id, current)
    measuring.delete(id)
    return current.width
  }

  const roots = children.get(VIRTUAL_ROOT_ID) ?? []
  const totalWidth = roots.length === 0
    ? 1
    : roots.reduce((sum, root) => sum + measure(root.id), 0)

  const place = (node: GraphNode, depth: number, left: number): number => {
    const descendants = children.get(node.id) ?? []
    let cursor = left
    const childCenters: number[] = []
    for (const child of descendants) {
      const childPosition = positions.get(child.id)
      const childWidth = childPosition?.width ?? 1
      place(child, depth + 1, cursor)
      childCenters.push(positions.get(child.id)?.x ?? cursor + childWidth / 2)
      cursor += childWidth
    }
    const position = positions.get(node.id) ?? { x: 0, z: 0, depth, width: 1 }
    position.depth = depth
    position.z = depth * LEVEL_DEPTH
    position.x = childCenters.length === 0
      ? left + position.width / 2
      : childCenters.reduce((sum, center) => sum + center, 0) / childCenters.length
    positions.set(node.id, position)
    return position.x
  }

  let cursor = -totalWidth / 2
  const rootCenters: number[] = []
  for (const root of roots) {
    const width = positions.get(root.id)?.width ?? 1
    place(root, 1, cursor)
    rootCenters.push(positions.get(root.id)?.x ?? cursor + width / 2)
    cursor += width
  }
  const rootCenter = rootCenters.length > 0
    ? rootCenters.reduce((sum, center) => sum + center, 0) / rootCenters.length
    : 0
  positions.set(VIRTUAL_ROOT_ID, { x: rootCenter, z: 0, depth: 0, width: Math.max(1, totalWidth) })

  const edges: GraphEdgeLayout[] = []
  for (const node of actualNodes) {
    const parentId = safeParentId(node, byId)
    edges.push({ id: `${parentId}:${node.id}`, parentId, childId: node.id })
  }

  const numeric = [...positions.values()]
  const minX = Math.min(...numeric.map((position) => position.x * CELL_WIDTH - position.width * CELL_WIDTH / 2))
  const maxX = Math.max(...numeric.map((position) => position.x * CELL_WIDTH + position.width * CELL_WIDTH / 2))
  const maxZ = Math.max(0, ...numeric.map((position) => position.z))
  return {
    positions: new Map([...positions].map(([id, position]) => [
      id,
      { x: position.x * CELL_WIDTH, z: position.z, depth: position.depth },
    ])),
    edges,
    centerX: (minX + maxX) / 2,
    centerZ: maxZ / 2,
    width: Math.max(NODE_WIDTH, maxX - minX),
    depth: Math.max(LEVEL_DEPTH, maxZ),
  }
}

function safeParentId(node: GraphNode, byId: Map<number, GraphNode>): number {
  let parentId = node.parentId != null && byId.has(node.parentId) && node.parentId !== node.id
    ? node.parentId
    : VIRTUAL_ROOT_ID
  const visited = new Set<number>([node.id])
  while (parentId !== VIRTUAL_ROOT_ID) {
    if (visited.has(parentId)) return VIRTUAL_ROOT_ID
    visited.add(parentId)
    const parent = byId.get(parentId)
    parentId = parent?.parentId != null && byId.has(parent.parentId) && parent.parentId !== parent.id
      ? parent.parentId
      : VIRTUAL_ROOT_ID
  }
  return node.parentId != null && byId.has(node.parentId) && node.parentId !== node.id
    ? node.parentId
    : VIRTUAL_ROOT_ID
}
