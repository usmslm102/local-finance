import { PageLoading, QueryError } from '@/components/ui/query-state'
import React from 'react'
import { useQuery } from '@tanstack/react-query'
import { fetchCategories, fetchRules } from '@/lib/api'
import { Tags, Sliders, CheckCircle2 } from 'lucide-react'
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'

export const RulesManager: React.FC = () => {
  const {
    data: categories,
    isLoading: categoriesLoading,
    isError: categoriesError,
    refetch: refetchCategories,
  } = useQuery({
    queryKey: ['categories'],
    queryFn: fetchCategories,
  })

  const {
    data: rules,
    isLoading: rulesLoading,
    isError: rulesError,
    refetch: refetchRules,
  } = useQuery({
    queryKey: ['rules'],
    queryFn: fetchRules,
  })

  if (categoriesLoading || rulesLoading) return <PageLoading label="Loading categories and rules" />
  if (categoriesError || rulesError)
    return (
      <QueryError
        message="Unable to load categories and rules"
        onRetry={() => Promise.all([refetchCategories(), refetchRules()])}
      />
    )

  return (
    <div className="flex flex-col gap-8">
      {/* Categories Grid */}
      <Card>
        <CardHeader className="pb-3">
          <div className="flex items-center justify-between">
            <div>
              <CardTitle className="text-base font-semibold flex items-center gap-2">
                <Tags className="size-4 text-primary" /> Spending Categories
              </CardTitle>
              <CardDescription className="text-xs">
                Classification system tailored for Indian personal and household finances
              </CardDescription>
            </div>
            <Badge variant="secondary">{categories?.length || 0} Categories</Badge>
          </div>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 gap-3">
            {categories?.map((cat) => (
              <div
                key={cat.id}
                className="flex items-center gap-3 rounded-lg border border-border/70 bg-muted/30 p-3 transition-colors hover:border-border"
              >
                <div
                  className="size-3 rounded-full flex-shrink-0"
                  style={{ backgroundColor: cat.color_hex }}
                />
                <div className="truncate">
                  <p className="text-xs font-semibold text-foreground truncate">{cat.name}</p>
                  <p className="text-[10px] text-muted-foreground font-medium">
                    {cat.is_system ? 'System' : 'Custom'}
                  </p>
                </div>
              </div>
            ))}
          </div>
        </CardContent>
      </Card>

      {/* Auto-Categorization Rules Table */}
      <Card>
        <CardHeader className="pb-3">
          <div className="flex items-center justify-between">
            <div>
              <CardTitle className="text-base font-semibold flex items-center gap-2">
                <Sliders className="size-4 text-primary" /> Auto-Categorization Rules
              </CardTitle>
              <CardDescription className="text-xs">
                Keyword and regex pattern matchers that classify incoming UPI VPAs, POS merchant
                names, and narrations
              </CardDescription>
            </div>
            <Badge variant="secondary">{rules?.length || 0} Rules</Badge>
          </div>
        </CardHeader>
        <CardContent className="pt-0">
          <div className="overflow-x-auto">
            <Table>
              <TableHeader>
                <TableRow className="hover:bg-transparent">
                  <TableHead className="text-xs font-semibold uppercase">Priority</TableHead>
                  <TableHead className="text-xs font-semibold uppercase">Match Field</TableHead>
                  <TableHead className="text-xs font-semibold uppercase">Type</TableHead>
                  <TableHead className="text-xs font-semibold uppercase">
                    Pattern / Keyword
                  </TableHead>
                  <TableHead className="text-xs font-semibold uppercase">
                    Assigned Category
                  </TableHead>
                  <TableHead className="text-xs font-semibold uppercase text-right">
                    Status
                  </TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {rules?.map((r) => (
                  <TableRow key={r.id} className="hover:bg-muted/40 text-xs">
                    <TableCell className="font-mono text-muted-foreground">{r.priority}</TableCell>
                    <TableCell className="font-mono text-foreground">{r.match_field}</TableCell>
                    <TableCell>
                      <Badge variant="outline" className="uppercase">
                        {r.match_type}
                      </Badge>
                    </TableCell>
                    <TableCell className="font-mono font-medium text-success">
                      {r.match_pattern}
                    </TableCell>
                    <TableCell className="font-semibold text-foreground">
                      {r.target_category || r.target_category_id}
                    </TableCell>
                    <TableCell className="text-right">
                      <span className="inline-flex items-center text-[11px] font-medium text-success">
                        <CheckCircle2 className="mr-1 size-3.5" /> Active
                      </span>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
