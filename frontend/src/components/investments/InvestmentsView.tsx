import { Field, FieldLabel } from '@/components/ui/field'
import { useMemo, useState, useId } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { TrendingUp } from 'lucide-react'
import { Link } from '@tanstack/react-router'
import { fetchInvestments, deleteInvestment } from '@/lib/api'
import { formatMoney } from '@/lib/formatters'
import { usePrivacy } from '@/components/privacy-provider'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  SelectGroup,
} from '@/components/ui/select'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'
import { InvestmentTotals } from './InvestmentTotals'

export function InvestmentsView() {
  const fieldId = useId()
  const client = useQueryClient()
  const { isPrivacyMode, maskValue } = usePrivacy()
  const [selected, setSelected] = useState('')
  const [search, setSearch] = useState('')
  const [asset, setAsset] = useState('ALL')
  const [sourceSheet, setSourceSheet] = useState('')
  const [showSource, setShowSource] = useState(false)
  const [message, setMessage] = useState('')
  const query = useQuery({ queryKey: ['investments'], queryFn: fetchInvestments })
  const snapshot = query.data?.find((s) => s.id === selected) ?? query.data?.[0]
  const remove = useMutation({
    mutationFn: deleteInvestment,
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: ['investments'] })
      setSelected('')
      setMessage('Snapshot deleted.')
    },
  })
  const holdings = useMemo(
    () =>
      snapshot?.holdings.filter(
        (h) =>
          (asset === 'ALL' || h.asset_class === asset) &&
          `${h.symbol} ${h.isin}`.toLowerCase().includes(search.toLowerCase())
      ) ?? [],
    [snapshot, asset, search]
  )
  const money = (value: number | null) =>
    formatMoney(value, snapshot?.currency ?? 'INR', isPrivacyMode)
  const price = (value: number | null) =>
    formatMoney(value, snapshot?.currency ?? 'INR', isPrivacyMode, 6)
  const percent = (value: number | null) =>
    value === null ? 'Not available' : maskValue(`${value.toFixed(2)}%`)
  const sheet = snapshot?.sheets.find((s) => s.name === sourceSheet) ?? snapshot?.sheets[0]
  const sourceRows = sheet?.rows.filter((row) => row?.some((cell) => cell.trim() !== '')) ?? []
  const error = remove.error ?? query.error

  return (
    <div className="flex flex-col gap-6">
      <div>
        <h1 className="text-2xl font-semibold flex items-center gap-2">
          <TrendingUp className="size-6" />
          Investments
        </h1>
        <p className="text-sm text-muted-foreground mt-1">
          Dated portfolio holdings and unrealized returns. Kept separate from income and expense
          totals.
        </p>
      </div>
      <InvestmentTotals snapshots={query.data ?? []} />
      <Link
        to="/import"
        search={{ tab: 'investments' }}
        className="inline-flex rounded-lg border px-3 py-2 text-sm font-medium"
      >
        Import investment statements
      </Link>
      {error && (
        <p role="alert" className="text-sm text-destructive">
          {error.message}
        </p>
      )}
      {message && (
        <p role="status" className="text-sm">
          {message}
        </p>
      )}
      {query.isPending && <p>Loading investments…</p>}
      {!query.isPending && !query.error && !snapshot && (
        <Card>
          <CardContent className="py-12 text-center text-muted-foreground">
            Import your first holdings statement to see your portfolio.
          </CardContent>
        </Card>
      )}
      {snapshot && (
        <>
          <div className="flex flex-wrap items-center justify-between gap-3">
            <Field className="flex flex-col gap-2">
              <FieldLabel htmlFor={`${fieldId}-54`}>Portfolio snapshot</FieldLabel>
              <Select
                value={snapshot.id}
                onValueChange={(value) => {
                  setSelected(value ?? '')
                  setSourceSheet('')
                  setAsset('ALL')
                }}
              >
                <SelectTrigger
                  id={`${fieldId}-54`}
                  className="w-full sm:w-[420px]"
                  aria-label="Portfolio snapshot"
                >
                  <SelectValue>
                    {snapshot.provider} · {maskValue(snapshot.account_ref)} · {snapshot.as_of}
                  </SelectValue>
                </SelectTrigger>
                <SelectContent>
                  <SelectGroup>
                    {query.data?.map((s) => (
                      <SelectItem key={s.id} value={s.id}>
                        {s.provider} · {maskValue(s.account_ref)} · {s.as_of} ·{' '}
                        {new Date(s.imported_at).toLocaleString()}
                      </SelectItem>
                    ))}
                  </SelectGroup>
                </SelectContent>
              </Select>
              <p className="text-xs text-muted-foreground">
                Holdings as of {snapshot.as_of} in {snapshot.currency}. Market valuations are shown
                only when provided by the statement. Earlier snapshots are not added to this one.
              </p>
              <p className="text-xs text-muted-foreground">
                {maskValue(snapshot.filename)} · Imported{' '}
                {new Date(snapshot.imported_at).toLocaleString()}
              </p>
            </Field>
            <Dialog>
              <DialogTrigger render={<Button variant="outline" disabled={remove.isPending} />}>
                Delete snapshot
              </DialogTrigger>
              <DialogContent>
                <DialogHeader>
                  <DialogTitle>Delete this investment snapshot?</DialogTitle>
                  <DialogDescription>
                    This removes the imported holdings and statement details from LocalFinance. You
                    can import the file again.
                  </DialogDescription>
                </DialogHeader>
                <DialogFooter>
                  <DialogClose render={<Button variant="outline" />}>Cancel</DialogClose>
                  <DialogClose
                    render={<Button variant="destructive" />}
                    onClick={() => remove.mutate(snapshot.id)}
                  >
                    Delete
                  </DialogClose>
                </DialogFooter>
              </DialogContent>
            </Dialog>
          </div>
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            {[
              ['Invested amount', money(snapshot.invested_value)],
              ['Current value', money(snapshot.current_value)],
              ['Unrealized return', money(snapshot.unrealized_return)],
              ['Return on cost', percent(snapshot.return_percent)],
            ].map(([label, value]) => (
              <Card key={label}>
                <CardHeader className="pb-2">
                  <CardTitle className="text-sm text-muted-foreground">{label}</CardTitle>
                </CardHeader>
                <CardContent className="text-xl font-semibold">{value}</CardContent>
              </Card>
            ))}
          </div>
          <p className="text-xs text-muted-foreground">
            Returns exclude realized trades, dividends, charges, and cash flows not included in the
            statement. Annualized return and XIRR are unavailable from a holdings snapshot.
          </p>
          {snapshot.warnings.map((w) => (
            <p key={w} role="status" className="text-sm">
              {w}
            </p>
          ))}
          <Card>
            <CardHeader>
              <CardTitle>Holdings ({holdings.length})</CardTitle>
              <div className="flex flex-wrap gap-3 pt-2">
                <Input
                  className="sm:max-w-xs"
                  placeholder="Search symbol or ISIN"
                  aria-label="Search holdings"
                  value={search}
                  onChange={(e) => setSearch(e.target.value)}
                />
                <Select value={asset} onValueChange={(value) => setAsset(value ?? 'ALL')}>
                  <SelectTrigger className="w-[180px]" aria-label="Asset class">
                    <SelectValue>{asset === 'ALL' ? 'All asset classes' : asset}</SelectValue>
                  </SelectTrigger>
                  <SelectContent>
                    <SelectGroup>
                      <SelectItem value="ALL">All asset classes</SelectItem>
                      {[...new Set(snapshot.holdings.map((h) => h.asset_class))].map((a) => (
                        <SelectItem key={a} value={a}>
                          {a}
                        </SelectItem>
                      ))}
                    </SelectGroup>
                  </SelectContent>
                </Select>
              </div>
            </CardHeader>
            <CardContent>
              <Table>
                <TableHeader>
                  <TableRow>
                    {[
                      'Holding',
                      'Quantity',
                      'Average cost price',
                      'Statement price',
                      'Invested',
                      'Current value',
                      'Unrealized return',
                      'Return %',
                    ].map((h) => (
                      <TableHead key={h} className="whitespace-nowrap">
                        {h}
                      </TableHead>
                    ))}
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {holdings.map((h) => (
                    <TableRow key={h.isin || h.symbol}>
                      <TableCell className="min-w-[220px]">
                        <div className="font-medium">{h.symbol}</div>
                        <div className="text-xs text-muted-foreground">
                          {h.isin ? `${h.isin} · ` : ''}
                          {h.asset_class}
                        </div>
                      </TableCell>
                      <TableCell>
                        {maskValue(
                          h.quantity.toLocaleString('en-IN', { maximumFractionDigits: 9 })
                        )}
                      </TableCell>
                      <TableCell>{price(h.average_price)}</TableCell>
                      <TableCell>{price(h.closing_price)}</TableCell>
                      <TableCell>{money(h.invested_value)}</TableCell>
                      <TableCell>{money(h.current_value)}</TableCell>
                      <TableCell
                        className={
                          h.unrealized_return !== null && h.unrealized_return < 0
                            ? 'text-destructive'
                            : ''
                        }
                      >
                        {money(h.unrealized_return)}
                      </TableCell>
                      <TableCell>{percent(h.return_percent)}</TableCell>
                    </TableRow>
                  ))}
                  {!holdings.length && (
                    <TableRow>
                      <TableCell colSpan={8} className="text-center">
                        No holdings match your filters.
                      </TableCell>
                    </TableRow>
                  )}
                </TableBody>
              </Table>
            </CardContent>
          </Card>
          <Card>
            <CardHeader className="flex flex-row items-center justify-between">
              <CardTitle>Statement details</CardTitle>
              <Button variant="outline" onClick={() => setShowSource(!showSource)}>
                {showSource ? 'Hide details' : 'Show all fields'}
              </Button>
            </CardHeader>
            {showSource && (
              <CardContent className="flex flex-col gap-3">
                <p className="text-xs text-muted-foreground">
                  Original statement fields are preserved here. Repeated worksheets are shown
                  without adding them to portfolio totals.
                </p>
                <h3 className="text-sm font-semibold">Provider fields by holding</h3>
                {holdings.map((holding) => (
                  <section
                    key={holding.isin || holding.symbol}
                    aria-label={`Provider fields for ${holding.symbol}`}
                    className="flex flex-col gap-2"
                  >
                    <h4 className="text-sm font-medium">{holding.symbol}</h4>
                    {Object.keys(holding.fields ?? {}).length ? (
                      <Table>
                        <TableHeader>
                          <TableRow>
                            <TableHead>Field</TableHead>
                            <TableHead>Value</TableHead>
                          </TableRow>
                        </TableHeader>
                        <TableBody>
                          {Object.entries(holding.fields).map(([field, value]) => (
                            <TableRow key={field}>
                              <TableCell>{field}</TableCell>
                              <TableCell>{maskValue(value)}</TableCell>
                            </TableRow>
                          ))}
                        </TableBody>
                      </Table>
                    ) : (
                      <p className="text-xs text-muted-foreground">
                        No additional provider fields.
                      </p>
                    )}
                  </section>
                ))}
                <h3 className="text-sm font-semibold">Original worksheets</h3>
                <Select
                  value={sheet?.name ?? ''}
                  onValueChange={(value) => setSourceSheet(value ?? '')}
                >
                  <SelectTrigger className="w-[200px]" aria-label="Statement worksheet">
                    <SelectValue>{sheet?.name}</SelectValue>
                  </SelectTrigger>
                  <SelectContent>
                    <SelectGroup>
                      {snapshot.sheets.map((s) => (
                        <SelectItem key={s.name} value={s.name}>
                          {s.name}
                        </SelectItem>
                      ))}
                    </SelectGroup>
                  </SelectContent>
                </Select>
                <Table>
                  <TableBody>
                    {sourceRows.map((row, i) => (
                      <TableRow key={i}>
                        {row.map((value, j) => (
                          <TableCell key={j} className="whitespace-nowrap">
                            {maskValue(value)}
                          </TableCell>
                        ))}
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </CardContent>
            )}
          </Card>
        </>
      )}
    </div>
  )
}
