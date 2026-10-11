import { CheckCircle2, Landmark, TrendingUp, Users } from 'lucide-react'
import { StatementUploader } from './StatementUploader'
import { InvestmentStatementUploader } from './InvestmentStatementUploader'
import { SplitwiseImporter } from './SplitwiseImporter'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'

const importOptions = [
  { value: 'bank', title: 'Bank & credit card statements', description: 'Import account transactions, card spending and balances.', icon: Landmark },
  { value: 'investments', title: 'Investment statements', description: 'Import your holdings and portfolio values.', icon: TrendingUp },
  { value: 'splitwise', title: 'Splitwise group history', description: 'Import shared expenses and match repayments to your bank.', icon: Users },
]

export function StatementImportView({ initialTab = 'bank' }: { initialTab?: string }) {
  return <div className="space-y-6">
    <div><h1 className="text-2xl font-bold tracking-tight">Statement Import</h1><p className="text-sm text-muted-foreground mt-1">Choose what you want to import, then upload your files below.</p></div>
    <Tabs defaultValue={initialTab} className="gap-6">
      <div className="space-y-3">
        <h2 id="import-type-heading" className="text-base font-semibold">What would you like to import?</h2>
        <TabsList aria-labelledby="import-type-heading" className="grid w-full grid-cols-1 items-stretch gap-3 bg-transparent p-0 group-data-horizontal/tabs:h-auto md:grid-cols-3">
          {importOptions.map(({ value, title, description, icon: Icon }) => <TabsTrigger key={value} value={value} className="group/import-choice h-auto min-w-0 items-start justify-start gap-4 whitespace-normal rounded-xl border border-border bg-card p-4 text-left text-foreground shadow-sm transition-colors hover:border-primary/50 hover:bg-accent/50 data-active:border-primary data-active:bg-primary/5 data-active:ring-1 data-active:ring-primary dark:data-active:border-primary dark:data-active:bg-primary/10 md:min-h-36 md:p-5">
            <span className="flex size-10 shrink-0 items-center justify-center rounded-lg bg-muted text-muted-foreground group-data-active/import-choice:bg-primary/10 group-data-active/import-choice:text-primary"><Icon className="size-5" aria-hidden="true" /></span>
            <span className="min-w-0 flex-1 space-y-1.5"><span className="block text-sm font-semibold leading-snug md:text-base">{title}</span><span className="block text-xs font-normal leading-relaxed text-muted-foreground md:text-sm">{description}</span></span>
            <CheckCircle2 aria-hidden="true" className="mt-0.5 size-4 shrink-0 text-primary opacity-0 group-data-active/import-choice:opacity-100" />
          </TabsTrigger>)}
        </TabsList>
      </div>
      <TabsContent value="bank" keepMounted><StatementUploader /></TabsContent>
      <TabsContent value="investments" keepMounted><InvestmentStatementUploader /></TabsContent>
      <TabsContent value="splitwise" keepMounted><SplitwiseImporter /></TabsContent>
    </Tabs>
  </div>
}
