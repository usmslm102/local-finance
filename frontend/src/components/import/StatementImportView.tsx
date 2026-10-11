import { StatementUploader } from './StatementUploader'
import { InvestmentStatementUploader } from './InvestmentStatementUploader'
import { SplitwiseImporter } from './SplitwiseImporter'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'

export function StatementImportView({ initialTab = 'bank' }: { initialTab?: string }) {
  return <div className="space-y-6">
    <div><h1 className="text-2xl font-bold tracking-tight">Statement Import</h1><p className="text-xs text-muted-foreground mt-1">Attach statements, detect their format, preview records, and import without duplicates.</p></div>
    <Tabs defaultValue={initialTab}>
      <TabsList className="w-full group-data-horizontal/tabs:h-auto sm:w-fit sm:group-data-horizontal/tabs:h-8"><TabsTrigger value="bank" className="h-auto min-w-0 whitespace-normal text-xs leading-tight sm:h-[calc(100%-1px)] sm:whitespace-nowrap sm:text-sm">Bank &amp; credit cards</TabsTrigger><TabsTrigger value="investments" className="h-auto min-w-0 text-xs sm:h-[calc(100%-1px)] sm:text-sm">Investments</TabsTrigger><TabsTrigger value="splitwise" className="h-auto min-w-0 text-xs sm:h-[calc(100%-1px)] sm:text-sm">Splitwise</TabsTrigger></TabsList>
      <TabsContent value="bank" keepMounted><StatementUploader /></TabsContent>
      <TabsContent value="investments" keepMounted><InvestmentStatementUploader /></TabsContent>
      <TabsContent value="splitwise" keepMounted><SplitwiseImporter /></TabsContent>
    </Tabs>
  </div>
}
