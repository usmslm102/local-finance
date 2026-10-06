import React, { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { fetchSystemVersion } from '@/lib/api'
import { Button } from '@/components/ui/button'
import { Sparkles } from 'lucide-react'
import { UpdateDialog } from './UpdateDialog'

export const UpdateIndicator: React.FC = () => {
  const [dialogOpen, setDialogOpen] = useState(false)

  const { data: versionInfo, refetch } = useQuery({
    queryKey: ['system-version'],
    queryFn: () => fetchSystemVersion(false),
    staleTime: 1000 * 60 * 30, // 30 minutes
    refetchOnWindowFocus: false,
  })

  if (!versionInfo?.update_available) {
    return null
  }

  return (
    <>
      <Button
        size="sm"
        variant="outline"
        onClick={() => setDialogOpen(true)}
        className="h-8 gap-1.5 text-xs font-semibold bg-primary/10 hover:bg-primary/20 text-primary border-primary/30 shadow-xs transition-all active:scale-[0.98] px-2.5"
        title={`New release ${versionInfo.latest_version} is available. Click to view and update.`}
      >
        <Sparkles className="h-3.5 w-3.5 text-primary shrink-0" />
        <span className="hidden sm:inline">Update</span>
        <span className="font-mono">{versionInfo.latest_version}</span>
      </Button>

      <UpdateDialog
        open={dialogOpen}
        onOpenChange={setDialogOpen}
        versionInfo={versionInfo}
        onUpdateSuccess={() => refetch()}
      />
    </>
  )
}
