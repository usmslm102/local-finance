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
        aria-label={`New release ${versionInfo.latest_version} is available. Click to view and update.`}
        size="sm"
        variant="outline"
        onClick={() => setDialogOpen(true)}
        className="h-8 gap-1.5 transition-all active:scale-[0.98] px-2.5"
        title={`New release ${versionInfo.latest_version} is available. Click to view and update.`}
      >
        <Sparkles data-icon="inline-start" className="text-primary shrink-0" />
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
