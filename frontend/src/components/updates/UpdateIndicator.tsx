import React from 'react'
import { Button } from '@/components/ui/button'
import { Sparkles } from 'lucide-react'
import { UpdateDialog } from './UpdateDialog'
import { useSystemUpdate } from '@/hooks/use-system-update'

export const UpdateIndicator: React.FC = () => {
  const { versionInfo, dialogOpen, setDialogOpen, refetch } = useSystemUpdate()

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
