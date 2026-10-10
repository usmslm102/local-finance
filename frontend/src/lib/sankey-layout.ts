import type { SankeyData, SankeyNode } from '@/types'

export interface FlowNode extends SankeyNode {
  members?: SankeyNode[]
  x: number
  y: number
  width: number
  height: number
}

export interface FlowLink {
  source: FlowNode
  target: FlowNode
  value: number
  sourceY: number
  targetY: number
  sourceHeight: number
  targetHeight: number
}

export const FLOW_LAYERS = ['Income sources', 'Bank accounts', 'Credit cards', 'Spending & savings']

// Reserve room for labels first, then distribute the remaining space by value.
// A proportional height followed by a minimum clamp overfills crowded columns.
export function allocateFlowHeights(values: number[], available: number, minimum = 64): number[] {
  if (!values.length) return []
  const heights = values.map(() => minimum)
  const remaining = Math.max(0, available - values.length * minimum)
  const total = values.reduce((sum, value) => sum + Math.max(0, value), 0)
  if (!total) return heights
  values.forEach((value, index) => {
    heights[index] += (remaining * Math.max(0, value)) / total
  })
  return heights
}

export function layoutCashFlow(data: SankeyData, expanded = false, availableWidth = 1120) {
  const width = Math.max(840, Math.min(1120, availableWidth))
  const nodeWidth = Math.min(196, Math.floor(width / 5.5))
  const gap = 14
  const padding = 12
  const mapping = new Map<string, string>()
  const layers: SankeyNode[][] = FLOW_LAYERS.map(() => [])
  data.nodes.forEach((node) => {
    if (Number.isFinite(node.total_value) && node.total_value > 0) {
      layers[Math.min(3, Math.max(0, node.layer))].push(node)
    }
  })
  let groupedCount = 0
  const displayLayers = layers.map((nodes, layer) => {
    const sorted = [...nodes].sort(
      (a, b) => b.total_value - a.total_value || a.id.localeCompare(b.id)
    )
    sorted.forEach((node) => mapping.set(node.id, node.id))
    if (expanded || sorted.length <= 7) return sorted
    // Keep savings/deficit visible: these are outcomes, never an "other" category.
    const pinned = sorted.filter((node) => node.type === 'SURPLUS' || node.type === 'DEFICIT')
    const regular = sorted.filter((node) => !pinned.includes(node))
    const visible = regular.slice(0, Math.max(1, 6 - pinned.length))
    const members = regular.slice(visible.length)
    const id = `group_${layer}`
    members.forEach((node) => mapping.set(node.id, id))
    groupedCount += members.length
    return [
      ...visible,
      ...pinned,
      {
        id,
        name: `${members.length} more ${layer === 0 ? 'sources' : layer === 1 ? 'accounts' : layer === 2 ? 'cards' : 'categories'}`,
        type: members[0].type,
        layer,
        color_hex: 'var(--muted-foreground)',
        total_value: members.reduce((sum, node) => sum + node.total_value, 0),
        tx_count: members.reduce((sum, node) => sum + (node.tx_count ?? 0), 0),
        members,
      },
    ]
  })
  const activeLayers = displayLayers
    .map((nodes, layer) => ({ nodes, layer }))
    .filter(({ nodes }) => nodes.length)
  const height = Math.max(
    520,
    Math.max(0, ...displayLayers.map((nodes) => nodes.length)) * (64 + gap) - gap + padding * 2
  )
  const nodes: FlowNode[] = []
  activeLayers.forEach(({ nodes: column }, index) => {
    const heights = allocateFlowHeights(
      column.map((node) => node.total_value),
      height - padding * 2 - gap * (column.length - 1)
    )
    let y = padding
    column.forEach((node, row) => {
      nodes.push({
        ...node,
        x:
          activeLayers.length === 1
            ? (width - nodeWidth) / 2
            : (index * (width - nodeWidth)) / (activeLayers.length - 1),
        y,
        width: nodeWidth,
        height: heights[row],
      })
      y += heights[row] + gap
    })
  })
  const nodeMap = new Map(nodes.map((node) => [node.id, node]))
  const combined = new Map<string, { source: string; target: string; value: number }>()
  data.links.forEach((link) => {
    const source = mapping.get(link.source),
      target = mapping.get(link.target)
    if (!source || !target || !Number.isFinite(link.value) || link.value <= 0) return
    const key = JSON.stringify([source, target])
    const previous = combined.get(key)
    if (previous) previous.value += link.value
    else combined.set(key, { source, target, value: link.value })
  })
  const incoming = new Map<string, number>(),
    outgoing = new Map<string, number>()
  combined.forEach((link) => {
    incoming.set(link.target, (incoming.get(link.target) ?? 0) + link.value)
    outgoing.set(link.source, (outgoing.get(link.source) ?? 0) + link.value)
  })
  const sourceOffsets = new Map<string, number>(),
    targetOffsets = new Map<string, number>()
  const links: FlowLink[] = []
  const sortedLinks = [...combined.values()].sort((a, b) => {
    const sourceDiff = nodeMap.get(a.source)!.y - nodeMap.get(b.source)!.y
    return (
      sourceDiff ||
      nodeMap.get(a.target)!.y - nodeMap.get(b.target)!.y ||
      a.source.localeCompare(b.source)
    )
  })
  sortedLinks.forEach((link) => {
    const source = nodeMap.get(link.source)!,
      target = nodeMap.get(link.target)!
    // Never impose a per-ribbon minimum: many tiny ribbons would spill outside a node.
    const sourceHeight =
      (source.height * link.value) / Math.max(source.total_value, outgoing.get(source.id) ?? 0)
    const targetHeight =
      (target.height * link.value) / Math.max(target.total_value, incoming.get(target.id) ?? 0)
    const sourceOffset = sourceOffsets.get(source.id) ?? 0,
      targetOffset = targetOffsets.get(target.id) ?? 0
    links.push({
      source,
      target,
      value: link.value,
      sourceY: source.y + sourceOffset,
      targetY: target.y + targetOffset,
      sourceHeight,
      targetHeight,
    })
    sourceOffsets.set(source.id, sourceOffset + sourceHeight)
    targetOffsets.set(target.id, targetOffset + targetHeight)
  })
  return { nodes, links, activeLayers, width, height, groupedCount }
}

export function flowPath(link: FlowLink) {
  const x0 = link.source.x + link.source.width,
    x1 = link.target.x
  const c1 = x0 + (x1 - x0) * 0.45,
    c2 = x1 - (x1 - x0) * 0.45
  return `M ${x0} ${link.sourceY} C ${c1} ${link.sourceY}, ${c2} ${link.targetY}, ${x1} ${link.targetY} L ${x1} ${link.targetY + link.targetHeight} C ${c2} ${link.targetY + link.targetHeight}, ${c1} ${link.sourceY + link.sourceHeight}, ${x0} ${link.sourceY + link.sourceHeight} Z`
}
