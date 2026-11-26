import { Link, useNavigate, useSearch } from '@tanstack/react-router'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'
import { ForecastView } from './ForecastView'
import { GoalsView } from './GoalsView'

export function PlanningView() {
  const search = useSearch({ strict: false }) as { tab?: 'goals' }
  const navigate = useNavigate()
  const tab = search.tab ?? 'forecast'
  return (
    <div className="flex flex-col gap-6">
      <div>
        <h1 className="text-2xl font-bold tracking-tight">Savings & planning</h1>
        <p className="mt-1 text-sm text-muted-foreground">
          Turn recorded finances into a plan. Everything stays on your machine.
        </p>
      </div>
      <Tabs
        value={tab}
        onValueChange={(value) =>
          void navigate({ to: '/planning', search: value === 'goals' ? { tab: 'goals' } : {} })
        }
      >
        <TabsList>
          <TabsTrigger value="forecast">Savings forecast</TabsTrigger>
          <TabsTrigger value="goals">Savings goals</TabsTrigger>
        </TabsList>
        <TabsContent value="forecast" className="mt-6">
          <ForecastView />
        </TabsContent>
        <TabsContent value="goals" className="mt-6">
          <GoalsView />
        </TabsContent>
      </Tabs>
      <p className="text-sm text-muted-foreground">
        Good plans start with recent statements.{' '}
        <Link to="/data-health" className="underline underline-offset-4">
          Review data health
        </Link>
        .
      </p>
    </div>
  )
}
