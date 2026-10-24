import type { NodeSummary } from '@/lib/api'

// The graph is rendered from the same flat node list as the sidebar, so it only
// depends on the fields it draws instead of introducing a second API shape.
export type GraphNode = Pick<
  NodeSummary,
  | 'id'
  | 'parentId'
  | 'title'
  | 'sortOrder'
  | 'completedAt'
  | 'childCount'
  | 'commentCount'
>

export interface GraphNeighborhood {
  node: GraphNode
  parent: GraphNode | null
  children: GraphNode[]
}
