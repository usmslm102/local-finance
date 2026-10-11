import { cn } from '@/lib/utils'
import { buttonVariants } from '@/components/ui/button'
import {
  Table,
  TableHeader,
  TableRow,
  TableHead,
  TableBody,
  TableCell,
} from '@/components/ui/table'
import React, { useState, useMemo } from 'react'
import { Link } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { fetchParsers } from '@/lib/api'
import { GUIDE_TOPICS, GUIDE_CATEGORIES, GuideTopic } from './guideData'
import {
  Search,
  ArrowRight,
  ArrowLeft,
  CheckCircle2,
  Sparkles,
  ShieldCheck,
  AlertTriangle,
  Clock,
  Menu,
  X,
  Cpu,
  Building2,
} from 'lucide-react'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'

export const GuideView: React.FC = () => {
  // Get topic from URL search param if present (e.g., /guide?topic=providers)
  const [activeTopicId, setActiveTopicId] = useState<string>(() => {
    if (typeof window !== 'undefined') {
      const params = new URLSearchParams(window.location.search)
      const t = params.get('topic')
      if (t && GUIDE_TOPICS.some((item) => item.id === t)) {
        return t
      }
    }
    return 'providers' // Default to Supported Providers as requested!
  })

  const [searchQuery, setSearchQuery] = useState('')
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false)

  // Fetch live backend parsers for the "providers" topic
  const { data: liveParsers, isLoading: parsersLoading } = useQuery({
    queryKey: ['parsers'],
    queryFn: fetchParsers,
  })

  // Sync URL when active topic changes
  const handleSelectTopic = (topicId: string) => {
    setActiveTopicId(topicId)
    setMobileMenuOpen(false)
    if (typeof window !== 'undefined') {
      const url = new URL(window.location.href)
      url.searchParams.set('topic', topicId)
      window.history.replaceState({}, '', url.toString())
    }
  }

  // Active topic object
  const activeTopic = useMemo(() => {
    return GUIDE_TOPICS.find((t) => t.id === activeTopicId) || GUIDE_TOPICS[0]
  }, [activeTopicId])

  // Filtered topics based on search
  const filteredTopics = useMemo(() => {
    if (!searchQuery.trim()) return GUIDE_TOPICS
    const q = searchQuery.toLowerCase()
    return GUIDE_TOPICS.filter((t) => {
      const matchTitle = t.title.toLowerCase().includes(q) || t.shortTitle.toLowerCase().includes(q)
      const matchSummary = t.summary.toLowerCase().includes(q)
      const matchSteps = t.steps.some(
        (s) => s.title.toLowerCase().includes(q) || s.content.toLowerCase().includes(q)
      )
      return matchTitle || matchSummary || matchSteps
    })
  }, [searchQuery])

  // Group filtered topics by category
  const groupedTopics = useMemo(() => {
    const groups: { category: string; topics: GuideTopic[] }[] = []
    GUIDE_CATEGORIES.forEach((cat) => {
      const items = filteredTopics.filter((t) => t.category === cat)
      if (items.length > 0) {
        groups.push({ category: cat, topics: items })
      }
    })
    return groups
  }, [filteredTopics])

  // Calculate Previous and Next topics for sequential reading
  const currentIdx = GUIDE_TOPICS.findIndex((t) => t.id === activeTopic.id)
  const prevTopic = currentIdx > 0 ? GUIDE_TOPICS[currentIdx - 1] : null
  const nextTopic = currentIdx < GUIDE_TOPICS.length - 1 ? GUIDE_TOPICS[currentIdx + 1] : null

  const ActiveIcon = activeTopic.icon

  return (
    <div className="max-w-7xl mx-auto flex flex-col gap-6">
      {/* Mobile Sub-Nav Header Button */}
      <div className="flex items-center justify-between lg:hidden border-b pb-3 gap-2">
        <div className="flex items-center gap-2 min-w-0">
          <div
            className="flex size-7 shrink-0 items-center justify-center rounded-md text-white shadow-xs"
            style={{ backgroundColor: activeTopic.iconColor }}
          >
            <ActiveIcon className="size-4" />
          </div>
          <span className="font-bold text-xs truncate text-foreground">
            {activeTopic.shortTitle}
          </span>
        </div>

        <Button
          variant="outline"
          size="sm"
          onClick={() => setMobileMenuOpen(!mobileMenuOpen)}
          className="h-8 gap-1.5 shrink-0"
        >
          {mobileMenuOpen ? <X className="size-3.5" /> : <Menu className="size-3.5" />}
          <span>Topics ({GUIDE_TOPICS.length})</span>
        </Button>
      </div>

      {/* Main Two-Column Layout */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-8 items-start">
        {/* ================= LEFT SUB-SIDE-NAV ================= */}
        <aside
          className={cn(
            'lg:col-span-4 xl:col-span-3 lg:sticky lg:top-20 flex flex-col gap-4',
            mobileMenuOpen ? 'block' : 'hidden lg:block'
          )}
        >
          {/* Search Box */}
          <div className="relative">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 size-3.5 text-muted-foreground" />
            <Input
              aria-label="Search guide &amp; topics"
              type="text"
              placeholder="Search guide &amp; topics..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="h-8.5 pl-8.5"
            />
          </div>

          {/* Navigation Groups */}
          <nav className="flex flex-col gap-5 rounded-xl border bg-card/60 backdrop-blur-xs p-3 shadow-xs max-h-[calc(100vh-12rem)] overflow-y-auto">
            {groupedTopics.map((group) => (
              <div key={group.category} className="flex flex-col gap-1.5">
                <h3 className="px-2.5 text-[10px] font-bold uppercase tracking-wider text-muted-foreground">
                  {group.category}
                </h3>

                <div className="flex flex-col gap-1">
                  {group.topics.map((t) => {
                    const Icon = t.icon
                    const isActive = activeTopic.id === t.id

                    return (
                      <Button
                        variant="ghost"
                        key={t.id}
                        onClick={() => handleSelectTopic(t.id)}
                        className={cn(
                          'w-full flex items-center justify-between gap-2.5 px-2.5 py-2 rounded-lg text-left text-xs transition-all cursor-pointer',
                          isActive
                            ? 'bg-primary text-primary-foreground font-semibold shadow-xs'
                            : 'text-muted-foreground hover:bg-muted/70 hover:text-foreground'
                        )}
                      >
                        <div className="flex items-center gap-2.5 min-w-0">
                          <Icon
                            className={cn(
                              'size-4 shrink-0',
                              isActive ? 'text-primary-foreground' : 'text-muted-foreground'
                            )}
                          />
                          <span className="truncate">{t.shortTitle}</span>
                        </div>

                        <span
                          className={cn(
                            'text-[9px] font-mono shrink-0 px-1 py-0.2 rounded-sm',
                            isActive
                              ? 'bg-white/20 text-primary-foreground'
                              : 'text-muted-foreground/80'
                          )}
                        >
                          {t.readTime.split(' ')[0]}m
                        </span>
                      </Button>
                    )
                  })}
                </div>
              </div>
            ))}
          </nav>

          {/* Offline Security Badge Card */}
          <div className="rounded-xl border border-success/20 bg-success/5 p-3.5 flex flex-col gap-1 text-xs text-foreground">
            <div className="flex items-center gap-1.5 font-semibold text-success">
              <ShieldCheck className="size-4 text-success" />
              <span>100% Offline Architecture</span>
            </div>
            <p className="text-[11px] text-muted-foreground leading-relaxed">
              All statements and SQLite tables live on your local hard drive. Zero telemetry, zero
              cloud calls.
            </p>
          </div>
        </aside>

        {/* ================= RIGHT BLOG-STYLE ARTICLE READER ================= */}
        <main className="lg:col-span-8 xl:col-span-9 flex flex-col gap-8 min-w-0">
          {/* Article Header Card */}
          <div className="rounded-2xl border bg-card/80 backdrop-blur-xs p-6 sm:p-8 shadow-xs flex flex-col gap-4">
            <div className="flex flex-wrap items-center justify-between gap-3 border-b border-border/50 pb-4">
              <div className="flex items-center gap-2">
                <Badge variant="outline" className="uppercase tracking-wider">
                  {activeTopic.category}
                </Badge>
                <span className="text-muted-foreground">•</span>
                <span className="text-[11px] text-muted-foreground flex items-center gap-1 font-medium">
                  <Clock className="size-3" /> {activeTopic.readTime}
                </span>
              </div>

              {/* Action CTA Button */}
              <Link
                to={activeTopic.ctaUrl}
                className={buttonVariants({
                  size: 'sm',
                  className: 'gap-2 text-xs font-semibold shadow-xs',
                })}
              >
                <span>{activeTopic.ctaText}</span>
                <ArrowRight className="size-3.5" />
              </Link>
            </div>

            {/* Title & Subtitle */}
            <div className="flex flex-col gap-2">
              <div className="flex items-center gap-3">
                <div
                  className="flex size-10 shrink-0 items-center justify-center rounded-xl text-white shadow-xs"
                  style={{ backgroundColor: activeTopic.iconColor }}
                >
                  <ActiveIcon className="size-5" />
                </div>
                <h1 className="text-2xl font-bold tracking-tight text-foreground sm:text-3xl">
                  {activeTopic.title}
                </h1>
              </div>
              <p className="text-sm text-muted-foreground leading-relaxed pt-1">
                {activeTopic.summary}
              </p>
            </div>

            {/* Step Jump Chips */}
            <div className="flex items-center gap-1.5 overflow-x-auto pt-2">
              <span className="text-[11px] font-semibold text-muted-foreground shrink-0 mr-1">
                Jump to:
              </span>
              {activeTopic.steps.map((s) => (
                <a
                  key={s.number}
                  href={`#step-${s.number}`}
                  className="px-2 py-0.5 rounded-md border text-[11px] font-medium text-muted-foreground hover:text-foreground hover:bg-muted/60 transition-colors shrink-0"
                >
                  Step {s.number}
                </a>
              ))}
            </div>
          </div>

          {/* SPECIAL WIDGET: Live Parser Registry Table (Displayed on 'providers' topic) */}
          {activeTopic.id === 'providers' && (
            <div className="rounded-xl border bg-card/60 backdrop-blur-xs p-5 shadow-xs flex flex-col gap-3">
              <div className="flex items-center justify-between">
                <h2 className="text-xs font-bold uppercase tracking-wider text-muted-foreground flex items-center gap-2">
                  <Cpu className="size-4 text-primary" />
                  Live Compiled Parser Registry ({liveParsers?.length ?? 5} Modules)
                </h2>
                <Badge variant="outline" className="text-success">
                  <CheckCircle2 className="size-3 mr-1" />
                  Ready in Binary
                </Badge>
              </div>

              <div className="overflow-x-auto">
                <Table className="w-full text-left text-xs">
                  <TableHeader className="bg-muted/50 border-b text-[10px] font-bold uppercase tracking-wider text-muted-foreground">
                    <TableRow>
                      <TableHead className="py-2.5 px-3">Parser Identifier</TableHead>
                      <TableHead className="py-2.5 px-3">Display Name</TableHead>
                      <TableHead className="py-2.5 px-3">Supported Type</TableHead>
                      <TableHead className="py-2.5 px-3 text-right">Status</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody className="divide-y divide-border/60">
                    {parsersLoading ? (
                      <TableRow>
                        <TableCell colSpan={4} className="py-4 text-center text-muted-foreground">
                          Querying Go binary parser registry...
                        </TableCell>
                      </TableRow>
                    ) : liveParsers && liveParsers.length > 0 ? (
                      liveParsers.map((p) => (
                        <TableRow key={p.id} className="hover:bg-muted/30 transition-colors">
                          <TableCell className="py-2 px-3 font-mono text-[11px] text-foreground font-semibold">
                            {p.id}
                          </TableCell>
                          <TableCell className="py-2 px-3 text-foreground">{p.name}</TableCell>
                          <TableCell className="py-2 px-3">
                            <span className="font-mono text-[10px] bg-muted/60 px-1.5 py-0.5 rounded border">
                              {p.supported_types.join(', ')}
                            </span>
                          </TableCell>
                          <TableCell className="py-2 px-3 text-right">
                            <Badge className="text-success px-1.5 py-0">Active Engine</Badge>
                          </TableCell>
                        </TableRow>
                      ))
                    ) : (
                      <TableRow>
                        <TableCell colSpan={4} className="py-3 text-center text-muted-foreground">
                          No parsers returned
                        </TableCell>
                      </TableRow>
                    )}
                  </TableBody>
                </Table>
              </div>
            </div>
          )}

          {/* ================= STEP-BY-STEP BLOG CONTENT ================= */}
          <div className="flex flex-col gap-6">
            {activeTopic.steps.map((step) => {
              return (
                <div
                  key={step.number}
                  id={`step-${step.number}`}
                  className="rounded-xl border bg-card/70 backdrop-blur-xs p-6 shadow-xs flex flex-col gap-3.5 scroll-mt-24 transition-all hover:border-border/90"
                >
                  {/* Step Header */}
                  <div className="flex items-center gap-3">
                    <div className="flex size-7 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary font-mono font-bold text-xs">
                      {step.number}
                    </div>
                    <h2 className="text-base font-bold text-foreground">{step.title}</h2>
                  </div>

                  {/* Prose Content */}
                  <div className="text-xs text-muted-foreground leading-relaxed whitespace-pre-line pl-10">
                    {step.content}
                  </div>

                  {/* Sample Recipe Table if available */}
                  {step.sampleData && (
                    <div className="ml-10 rounded-lg border bg-background/60 overflow-hidden text-xs">
                      <div className="bg-muted/60 px-3.5 py-2 font-bold text-[11px] text-foreground border-b flex items-center gap-2">
                        <Building2 className="size-3.5 text-primary" />
                        <span>{step.sampleData.label}</span>
                      </div>
                      <div className="divide-y divide-border/60">
                        {step.sampleData.rows.map((row, rIdx) => (
                          <div key={rIdx} className="p-3 grid grid-cols-1 sm:grid-cols-3 gap-2">
                            <span className="font-semibold text-foreground text-xs sm:col-span-1">
                              {row.key}
                            </span>
                            <span className="text-muted-foreground text-[11px] sm:col-span-2 leading-relaxed">
                              {row.value}
                            </span>
                          </div>
                        ))}
                      </div>
                    </div>
                  )}

                  {/* Pro-Tip Callout Box */}
                  {step.proTip && (
                    <div
                      className={cn(
                        'ml-10 rounded-lg border p-3.5 text-xs flex items-start gap-2.5',
                        step.calloutType === 'security'
                          ? 'border-success/30 bg-success/5 text-success '
                          : step.calloutType === 'warning'
                            ? 'border-warning/30 bg-warning/5 text-warning '
                            : 'border-primary/30 bg-primary/5 text-foreground'
                      )}
                    >
                      {step.calloutType === 'security' ? (
                        <ShieldCheck className="size-4 text-success shrink-0 mt-0.5" />
                      ) : step.calloutType === 'warning' ? (
                        <AlertTriangle className="size-4 text-warning shrink-0 mt-0.5" />
                      ) : (
                        <Sparkles className="size-4 text-primary shrink-0 mt-0.5" />
                      )}
                      <div className="flex flex-col gap-0.5">
                        <span className="font-bold text-[10px] uppercase tracking-wider block">
                          {step.calloutType === 'security'
                            ? 'Security Note'
                            : step.calloutType === 'warning'
                              ? 'Important Warning'
                              : 'Pro-Tip'}
                        </span>
                        <p className="text-[11px] leading-relaxed opacity-90">{step.proTip}</p>
                      </div>
                    </div>
                  )}
                </div>
              )
            })}
          </div>

          {/* Footer Notes if any */}
          {activeTopic.footerNotes && (
            <div className="rounded-xl border border-dashed bg-muted/20 p-4 text-xs text-muted-foreground text-center">
              {activeTopic.footerNotes}
            </div>
          )}

          {/* Article Bottom Action Ribbon */}
          <div className="rounded-2xl border bg-card/60 backdrop-blur-xs p-6 shadow-xs flex flex-col sm:flex-row items-center justify-between gap-4">
            <div className="flex flex-col gap-1 text-center sm:text-left">
              <h3 className="text-sm font-bold text-foreground">
                Ready to try this in your finances?
              </h3>
              <p className="text-xs text-muted-foreground">
                Jump directly to the interactive feature workbench with 1 click.
              </p>
            </div>

            <Link
              to={activeTopic.ctaUrl}
              className={buttonVariants({
                size: 'sm',
                className: 'gap-2 text-xs font-semibold shadow-xs',
              })}
            >
              <span>{activeTopic.ctaText}</span>
              <ArrowRight className="size-3.5" />
            </Link>
          </div>

          {/* Sequential Reading Navigation (Prev / Next Buttons) */}
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 pt-4 border-t border-border/60">
            {prevTopic ? (
              <Button
                variant="ghost"
                onClick={() => handleSelectTopic(prevTopic.id)}
                className="flex items-center gap-3 p-4 text-left transition-all cursor-pointer group"
              >
                <ArrowLeft
                  data-icon="inline-start"
                  className="text-muted-foreground group-hover:text-primary transition-transform group-hover:-translate-x-1"
                />
                <div className="min-w-0">
                  <span className="text-[10px] font-bold uppercase tracking-wider text-muted-foreground block">
                    Previous Topic
                  </span>
                  <span className="text-xs font-semibold text-foreground truncate block">
                    {prevTopic.shortTitle}
                  </span>
                </div>
              </Button>
            ) : (
              <div />
            )}

            {nextTopic ? (
              <Button
                variant="ghost"
                onClick={() => handleSelectTopic(nextTopic.id)}
                className="flex items-center justify-between sm:justify-end gap-3 p-4 text-right transition-all cursor-pointer group"
              >
                <div className="min-w-0">
                  <span className="text-[10px] font-bold uppercase tracking-wider text-muted-foreground block">
                    Next Topic
                  </span>
                  <span className="text-xs font-semibold text-foreground truncate block">
                    {nextTopic.shortTitle}
                  </span>
                </div>
                <ArrowRight
                  data-icon="inline-start"
                  className="text-muted-foreground group-hover:text-primary transition-transform group-hover:translate-x-1"
                />
              </Button>
            ) : (
              <div />
            )}
          </div>
        </main>
      </div>
    </div>
  )
}
