import React, { useId, useMemo, useState, useRef, useEffect } from 'react'
import type { SankeyData, SankeyNode } from '@/types'
import { cn, formatINR } from '@/lib/utils'
import { FLOW_LAYERS, flowPath, layoutCashFlow, type FlowNode } from '@/lib/sankey-layout'
import { usePrivacy } from '@/components/privacy-provider'
import { Button } from '@/components/ui/button'
import { Empty, EmptyHeader, EmptyMedia, EmptyTitle, EmptyDescription } from '@/components/ui/empty'
import { Tooltip, TooltipTrigger, TooltipContent } from '@/components/ui/tooltip'
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group'
import { PiggyBank, List, GitBranch, ChevronDown } from 'lucide-react'

interface SankeyDiagramProps {
  data: SankeyData
  onNodeClick?: (node: SankeyNode) => void
  selectedNodeId?: string | null
}

export const SankeyDiagram: React.FC<SankeyDiagramProps> = ({
  data,
  onNodeClick,
  selectedNodeId,
}) => {
  const gradientId = useId().replace(/:/g, '')
  const [expanded, setExpanded] = useState(false)
  const [view, setView] = useState('diagram')
  const diagramRef = useRef<HTMLDivElement>(null)
  const [diagramWidth, setDiagramWidth] = useState(1120)
  useEffect(() => {
    const element = diagramRef.current
    if (!element) return
    const observer = new ResizeObserver(([entry]) => {
      if (entry.contentRect.width > 0) setDiagramWidth(entry.contentRect.width)
    })
    observer.observe(element)
    return () => observer.disconnect()
  }, [data.nodes.length])
  const [hovered, setHovered] = useState<string | null>(null)
  const { isPrivacyMode } = usePrivacy()
  const money = (amount: number) => (isPrivacyMode ? '₹••••••' : formatINR(amount, isPrivacyMode))
  const layout = useMemo(
    () => layoutCashFlow(data, expanded, diagramWidth),
    [data, expanded, diagramWidth]
  )
  const compact = useMemo(
    () => (expanded ? layoutCashFlow(data) : layout),
    [data, expanded, layout]
  )
  const focusNode = hovered || selectedNodeId
  const connected = new Set(focusNode ? [focusNode] : [])
  layout.links.forEach((link) => {
    if (link.source.id === focusNode || link.target.id === focusNode) {
      connected.add(link.source.id)
      connected.add(link.target.id)
    }
  })
  const canInspect = (node: FlowNode) => !!node.members || !!onNodeClick
  const inspect = (node: FlowNode) => {
    if (node.members) setExpanded(true)
    else if (canInspect(node)) onNodeClick?.(node)
  }

  if (!layout.nodes.length)
    return (
      <Empty className="border py-12">
        <EmptyHeader>
          <EmptyMedia variant="icon">
            <PiggyBank />
          </EmptyMedia>
          <EmptyTitle>No cash flow in this period</EmptyTitle>
          <EmptyDescription>
            Import a statement or choose another month to see where your money goes.
          </EmptyDescription>
        </EmptyHeader>
      </Empty>
    )

  return (
    <div className="flex min-w-0 flex-col gap-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <p className="text-xs text-muted-foreground">
          {compact.groupedCount > 0
            ? 'Largest flows shown first. Smaller flows are grouped without changing totals.'
            : 'Select a source, account, card, or category to inspect its transactions.'}
        </p>
        <div className="flex flex-wrap items-center gap-2">
          {compact.groupedCount > 0 && (
            <Button
              variant="outline"
              size="sm"
              onClick={() => setExpanded(!expanded)}
              aria-expanded={expanded}
            >
              <ChevronDown data-icon="inline-start" />
              {expanded ? 'Compact view' : 'Show every flow'}
            </Button>
          )}
          <ToggleGroup
            value={[view]}
            onValueChange={(values) => {
              if (values[0]) setView(values[0])
            }}
            aria-label="Cash flow presentation"
            className="hidden md:flex"
            variant="outline"
            size="sm"
          >
            <ToggleGroupItem value="diagram">
              <GitBranch data-icon="inline-start" />
              Diagram
            </ToggleGroupItem>
            <ToggleGroupItem value="list">
              <List data-icon="inline-start" />
              List
            </ToggleGroupItem>
          </ToggleGroup>
        </div>
      </div>

      <div
        ref={diagramRef}
        className={cn(
          'overflow-auto rounded-lg border p-3',
          view === 'list' ? 'hidden' : 'hidden md:block'
        )}
        style={{ maxHeight: 680 }}
        tabIndex={0}
        role="region"
        aria-label="Cash flow diagram. Scroll to explore all flows."
      >
        <div style={{ width: layout.width }}>
          <div className="relative mb-2 h-6">
            {layout.activeLayers.map(({ layer }) => {
              const node = layout.nodes.find((node) => node.layer === layer)!
              return (
                <h3
                  key={layer}
                  className="absolute text-center text-xs font-medium text-muted-foreground"
                  style={{ left: node.x, width: node.width }}
                >
                  {FLOW_LAYERS[layer]}
                </h3>
              )
            })}
          </div>
          <svg
            width={layout.width}
            height={layout.height}
            viewBox={`0 0 ${layout.width} ${layout.height}`}
            aria-label="Money flowing from income to accounts, spending and savings"
          >
            <defs>
              {layout.links.map((link, index) => (
                <linearGradient
                  key={index}
                  id={`${gradientId}-${index}`}
                  gradientUnits="userSpaceOnUse"
                  x1={link.source.x + link.source.width}
                  y1={0}
                  x2={link.target.x}
                  y2={0}
                >
                  <stop offset="0%" stopColor={link.source.color_hex} />
                  <stop offset="100%" stopColor={link.target.color_hex} />
                </linearGradient>
              ))}
            </defs>
            {layout.links.map((link, index) => (
              <Tooltip key={index}>
                <TooltipTrigger
                  render={
                    <path
                      d={flowPath(link)}
                      fill={`url(#${gradientId}-${index})`}
                      className="transition-opacity"
                      style={{
                        opacity: focusNode
                          ? link.source.id === focusNode || link.target.id === focusNode
                            ? 0.7
                            : 0.07
                          : 0.25,
                      }}
                    />
                  }
                />
                <TooltipContent>
                  {link.source.name} → {link.target.name}: {money(link.value)}
                </TooltipContent>
              </Tooltip>
            ))}
            {layout.nodes.map((node) => (
              <foreignObject
                key={node.id}
                x={node.x}
                y={node.y}
                width={node.width}
                height={node.height}
              >
                <Button
                  variant="outline"
                  onClick={() => inspect(node)}
                  aria-label={`${node.name}, ${money(node.total_value)}${node.members ? '. Expand grouped flows' : canInspect(node) ? '. View transactions' : ''}`}
                  aria-pressed={selectedNodeId === node.id}
                  onMouseEnter={() => setHovered(node.id)}
                  onMouseLeave={() => setHovered(null)}
                  onFocus={() => setHovered(node.id)}
                  onBlur={() => setHovered(null)}
                  className={cn(
                    'h-full w-full flex-col items-start justify-center gap-1 px-3 whitespace-normal text-left',
                    focusNode && !connected.has(node.id) && 'opacity-40',
                    selectedNodeId === node.id && 'ring-2 ring-ring'
                  )}
                >
                  <span className="flex w-full min-w-0 items-center gap-2">
                    <span
                      className="size-2 shrink-0 rounded-full"
                      style={{ backgroundColor: node.color_hex }}
                    />
                    <span className="truncate" title={node.name}>
                      {node.name}
                    </span>
                  </span>
                  <span className="flex w-full items-baseline justify-between gap-1">
                    <span className="tabular-nums">{money(node.total_value)}</span>
                    <span className="text-xs text-muted-foreground">
                      {node.tx_count ? `${node.tx_count} tx` : node.members ? 'Expand' : ''}
                    </span>
                  </span>
                </Button>
              </foreignObject>
            ))}
          </svg>
        </div>
      </div>

      <div className={cn('flex flex-col gap-5', view === 'diagram' && 'md:hidden')}>
        {layout.activeLayers.map(({ layer }) => (
          <section key={layer} className="flex flex-col gap-2" aria-label={FLOW_LAYERS[layer]}>
            <h3 className="text-sm font-medium">{FLOW_LAYERS[layer]}</h3>
            {layout.nodes
              .filter((node) => node.layer === layer)
              .map((node) => (
                <Button
                  key={node.id}
                  variant="outline"
                  onClick={() => inspect(node)}
                  className="h-auto min-h-11 w-full justify-between gap-3 whitespace-normal px-3 py-2 text-left"
                  aria-pressed={selectedNodeId === node.id}
                >
                  <span className="flex min-w-0 items-center gap-2">
                    <span
                      className="size-2 shrink-0 rounded-full"
                      style={{ backgroundColor: node.color_hex }}
                    />
                    <span className="truncate">{node.name}</span>
                  </span>
                  <span className="shrink-0 tabular-nums">{money(node.total_value)}</span>
                </Button>
              ))}
          </section>
        ))}
      </div>
      <p className="text-xs text-muted-foreground">
        Ribbon widths show relative flow. Account-to-card routing is an overview of recorded
        spending, not proof of a specific bill payment. Savings and shortfalls are calculated
        totals.
      </p>
    </div>
  )
}
