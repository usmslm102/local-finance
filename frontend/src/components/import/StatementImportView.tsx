import { StatementUploader } from './StatementUploader'
import { InvestmentStatementUploader } from './InvestmentStatementUploader'
import { SplitwiseImporter } from './SplitwiseImporter'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'

export function StatementImportView({ initialTab = 'bank' }: { initialTab?: string }) {
  return <div className="space-y-6">
    <div><h1 className="text-2xl font-bold tracking-tight">Statement Import</h1><p className="text-xs text-muted-foreground mt-1">Attach statements, detect their format, preview records, and import without duplicates.</p></div>
    <Tabs defaultValue={initialTab}>
      <TabsList><TabsTrigger value="bank">Bank &amp; credit cards</TabsTrigger><TabsTrigger value="investments">Investments</TabsTrigger><TabsTrigger value="splitwise">Splitwise</TabsTrigger></TabsList>
      <TabsContent value="bank" keepMounted><StatementUploader /></TabsContent>
      <TabsContent value="investments" keepMounted><InvestmentStatementUploader /></TabsContent>
      <TabsContent value="splitwise" keepMounted><SplitwiseImporter /></TabsContent>
    </Tabs>
  </div>
}
