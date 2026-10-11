import React, { useState } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import {
  fetchAccounts,
  fetchParsers,
  fetchStatements,
  uploadStatement,
  previewStatement,
  fetchSampleStatements,
  loadSampleFile,
  SampleStatementItem,
} from '@/lib/api'
import { ImportResult, StatementPreviewResult, PreviewTransactionItem } from '@/types'
import { formatDate, formatINR } from '@/lib/utils'
import {
  UploadCloud,
  CheckCircle2,
  AlertCircle,
  RefreshCw,
  History,
  Unlock,
  KeyRound,
  FileText,
  Eye,
  EyeOff,
  Layers,
  Search,
  Calendar,
  X,
  Sliders,
  FlaskConical,
  Sparkles,
} from 'lucide-react'
import { MerchantAvatar } from '@/components/ui/merchant-avatar'
import { CopyButton } from '@/components/ui/copy-button'
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Link } from '@tanstack/react-router'

interface QueuedFile {
  id: string
  file: File
  password?: string
  status: 'previewing' | 'ready' | 'password_required' | 'error' | 'importing' | 'imported'
  preview?: StatementPreviewResult
  error?: string
  result?: ImportResult
}

export const StatementUploader: React.FC = () => {
  const queryClient = useQueryClient()
  const [queuedFiles, setQueuedFiles] = useState<QueuedFile[]>([])
  const [parserId, setParserId] = useState('AUTO')
  const [accountId, setAccountId] = useState('AUTO')
  const [activePreviewFile, setActivePreviewFile] = useState<QueuedFile | null>(null)
  const [previewSearch, setPreviewSearch] = useState('')
  const [isDragging, setIsDragging] = useState(false)
  const [isBatchImporting, setIsBatchImporting] = useState(false)
  const [bulkPassword, setBulkPassword] = useState('')
  const [showBulkPassword, setShowBulkPassword] = useState(false)
  const [isBulkUnlocking, setIsBulkUnlocking] = useState(false)
  const [rememberedPassword, setRememberedPassword] = useState('')

  const { data: parsers } = useQuery({
    queryKey: ['parsers'],
    queryFn: fetchParsers,
  })

  const { data: accounts } = useQuery({
    queryKey: ['accounts'],
    queryFn: fetchAccounts,
  })

  const { data: statementsHistory } = useQuery({
    queryKey: ['statements'],
    queryFn: fetchStatements,
  })

  const { data: sampleStatements } = useQuery({
    queryKey: ['sample-statements'],
    queryFn: fetchSampleStatements,
  })

  const [loadingSamplePath, setLoadingSamplePath] = useState<string | null>(null)

  const handleLoadSample = async (sample: SampleStatementItem) => {
    try {
      setLoadingSamplePath(sample.path)
      const sampleFile = await loadSampleFile(sample.path, sample.filename)
      handleFilesAdded([sampleFile])
    } catch (err: any) {
      console.error('Failed to load sample statement:', err)
    } finally {
      setLoadingSamplePath(null)
    }
  }

  // Process and generate preview for a single file
  const processFilePreview = async (qFile: QueuedFile, customPassword?: string) => {
    setQueuedFiles((prev) =>
      prev.map((f) =>
        f.id === qFile.id ? { ...f, status: 'previewing', error: undefined } : f
      )
    )

    try {
      const formData = new FormData()
      formData.append('file', qFile.file)
      if (parserId && parserId !== 'AUTO') formData.append('parser_id', parserId)
      if (accountId && accountId !== 'AUTO') formData.append('account_id', accountId)
      const pwd = customPassword !== undefined ? customPassword : qFile.password || ''
      if (pwd) formData.append('password', pwd)

      const previewRes = await previewStatement(formData)

      setQueuedFiles((prev) =>
        prev.map((f) => {
          if (f.id !== qFile.id) return f
          if (previewRes.requires_password) {
            return {
              ...f,
              status: 'password_required',
              preview: previewRes,
              password: pwd,
              error: pwd ? 'Incorrect password. Please try again.' : 'Password required to open encrypted statement PDF',
            }
          }
          if (previewRes.error) {
            return {
              ...f,
              status: 'error',
              preview: previewRes,
              error: previewRes.error,
            }
          }
          if (pwd) {
            setRememberedPassword(pwd)
          }
          return {
            ...f,
            status: 'ready',
            preview: previewRes,
            password: pwd,
            error: undefined,
          }
        })
      )
    } catch (err: any) {
      setQueuedFiles((prev) =>
        prev.map((f) =>
          f.id === qFile.id
            ? { ...f, status: 'error', error: err.message || 'Failed to preview file' }
            : f
        )
      )
    }
  }

  const handleFilesAdded = (files: FileList | File[]) => {
    const newQueue: QueuedFile[] = []
    for (let i = 0; i < files.length; i++) {
      const file = files[i]
      const id = `${file.name}-${file.size}-${Date.now()}-${i}`
      const qFile: QueuedFile = {
        id,
        file,
        status: 'previewing',
      }
      newQueue.push(qFile)
    }

    setQueuedFiles((prev) => [...prev, ...newQueue])

    // Immediately request preview for all added files concurrently
    newQueue.forEach((qFile) => {
      processFilePreview(qFile, rememberedPassword || undefined)
    })
  }

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.files && e.target.files.length > 0) {
      handleFilesAdded(e.target.files)
      e.target.value = ''
    }
  }

  const handleDragOver = (e: React.DragEvent) => {
    e.preventDefault()
    setIsDragging(true)
  }

  const handleDragLeave = () => {
    setIsDragging(false)
  }

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault()
    setIsDragging(false)
    if (e.dataTransfer.files && e.dataTransfer.files.length > 0) {
      handleFilesAdded(e.dataTransfer.files)
    }
  }

  const removeFileFromQueue = (id: string) => {
    setQueuedFiles((prev) => prev.filter((f) => f.id !== id))
    if (activePreviewFile?.id === id) {
      setActivePreviewFile(null)
    }
  }

  const clearQueue = () => {
    setQueuedFiles([])
    setActivePreviewFile(null)
  }

  const handlePasswordSubmit = (qFile: QueuedFile, pwd: string) => {
    processFilePreview(qFile, pwd)
  }

  const handleBulkPasswordSubmit = async (customPwd?: string) => {
    const pwdToUse = (customPwd !== undefined ? customPwd : bulkPassword).trim()
    if (!pwdToUse) return

    setIsBulkUnlocking(true)
    const lockedFiles = queuedFiles.filter((f) => f.status === 'password_required')
    try {
      await Promise.all(
        lockedFiles.map((qFile) => processFilePreview(qFile, pwdToUse).catch(() => null))
      )
      setRememberedPassword(pwdToUse)
      setBulkPassword('')
    } finally {
      setIsBulkUnlocking(false)
    }
  }

  const importSingleFile = async (qFile: QueuedFile) => {
    setQueuedFiles((prev) =>
      prev.map((f) => (f.id === qFile.id ? { ...f, status: 'importing' } : f))
    )

    try {
      const formData = new FormData()
      formData.append('file', qFile.file)
      if (parserId && parserId !== 'AUTO') formData.append('parser_id', parserId)
      if (accountId && accountId !== 'AUTO') formData.append('account_id', accountId)
      if (qFile.password) formData.append('password', qFile.password)

      const result = await uploadStatement(formData)

      setQueuedFiles((prev) =>
        prev.map((f) =>
          f.id === qFile.id ? { ...f, status: 'imported', result } : f
        )
      )

      queryClient.invalidateQueries({ queryKey: ['analytics'] })
      queryClient.invalidateQueries({ queryKey: ['transactions'] })
      queryClient.invalidateQueries({ queryKey: ['accounts'] })
      queryClient.invalidateQueries({ queryKey: ['statements'] })
      queryClient.invalidateQueries({ queryKey: ['credit-card-bills'] })
    } catch (err: any) {
      setQueuedFiles((prev) =>
        prev.map((f) =>
          f.id === qFile.id
            ? { ...f, status: 'error', error: err.message || 'Import failed' }
            : f
        )
      )
    }
  }

  const handleImportAllReady = async () => {
    const readyFiles = queuedFiles.filter((f) => f.status === 'ready')
    if (readyFiles.length === 0) return

    setIsBatchImporting(true)
    for (const qFile of readyFiles) {
      await importSingleFile(qFile)
    }
    setIsBatchImporting(false)
  }

  const readyCount = queuedFiles.filter((f) => f.status === 'ready').length
  const lockedCount = queuedFiles.filter((f) => f.status === 'password_required').length
  const totalTxsInReady = queuedFiles
    .filter((f) => f.status === 'ready')
    .reduce((acc, f) => acc + (f.preview?.total_transactions || 0), 0)

  // Filter preview transactions if search query present
  const filteredPreviewTransactions: PreviewTransactionItem[] =
    activePreviewFile?.preview?.transactions?.filter((tx) => {
      if (!previewSearch) return true
      const q = previewSearch.toLowerCase()
      return (
        tx.cleaned_payee.toLowerCase().includes(q) ||
        tx.raw_narration.toLowerCase().includes(q) ||
        tx.reference_number.toLowerCase().includes(q) ||
        tx.date.includes(q) ||
        tx.payment_mode.toLowerCase().includes(q) ||
        (tx.category_name && tx.category_name.toLowerCase().includes(q))
      )
    }) || []

  return (
    <div className="space-y-8">
      {/* Upload Zone & Settings */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
        {/* Dropzone Card */}
        <Card className="lg:col-span-7 border-border/80 bg-card shadow-xs">
          <CardHeader>
            <CardTitle className="text-base font-semibold flex items-center gap-2">
              <UploadCloud className="h-4 w-4 text-primary" /> Import Statements
            </CardTitle>
            <CardDescription className="text-xs">
              Drop one or multiple PDF, Excel (.xls/.xlsx), and CSV statements. Inspect parsed records before saving.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            {/* Drag & Drop Box with multiple support */}
            <div
              onDragOver={handleDragOver}
              onDragLeave={handleDragLeave}
              onDrop={handleDrop}
              className={`relative flex flex-col items-center justify-center rounded-xl border-2 border-dashed p-8 text-center transition-colors ${
                isDragging
                  ? 'border-primary bg-primary/5'
                  : 'border-border/80 bg-muted/20 hover:border-border hover:bg-muted/40'
              }`}
            >
              <input
                type="file"
                multiple
                accept=".csv,.xls,.xlsx,.pdf"
                onChange={handleFileChange}
                className="absolute inset-0 cursor-pointer opacity-0"
              />
              <div className="flex h-12 w-12 items-center justify-center rounded-full bg-muted border text-muted-foreground mb-3">
                <UploadCloud className="h-6 w-6" />
              </div>
              <div>
                <p className="text-sm font-medium text-foreground">
                  Click to select or drag and drop multiple statement files
                </p>
                <p className="text-xs text-muted-foreground mt-1">
                  HDFC, ICICI, Axis, SBI Bank & Credit Cards (PDF, XLS, XLSX, CSV)
                </p>
              </div>
            </div>

            {/* 1-Click Sample Statement Loader */}
            {sampleStatements && sampleStatements.length > 0 && (
              <div className="pt-2 border-t border-border/70 space-y-2">
                <div className="flex items-center justify-between">
                  <span className="text-[11px] font-semibold text-foreground flex items-center gap-1.5">
                    <FlaskConical className="h-3.5 w-3.5 text-amber-400" /> Test with a masked sample statement:
                  </span>
                  <span className="text-[10px] text-muted-foreground font-mono">1-Click Sniff &amp; Preview</span>
                </div>
                <div className="flex flex-wrap gap-1.5">
                  {sampleStatements.map((s) => {
                    const isLoadingThis = loadingSamplePath === s.path
                    return (
                      <button
                        key={s.path}
                        type="button"
                        disabled={isLoadingThis}
                        onClick={() => handleLoadSample(s)}
                        className="inline-flex items-center gap-1.5 rounded-md border border-border/70 bg-muted/40 hover:bg-muted hover:border-primary/40 px-2 py-1 text-[11px] font-medium text-foreground transition-all duration-150 active:scale-95 cursor-pointer disabled:opacity-50"
                      >
                        {isLoadingThis ? (
                          <RefreshCw className="h-3 w-3 animate-spin text-primary" />
                        ) : (
                          <Sparkles className="h-3 w-3 text-amber-400" />
                        )}
                        <span>{s.title}</span>
                      </button>
                    )
                  })}
                </div>
              </div>
            )}
          </CardContent>
        </Card>

        {/* Options & Reset */}
        <Card className="lg:col-span-5 border-border/80 bg-card shadow-xs flex flex-col justify-between">
          <div>
            <CardHeader>
              <CardTitle className="text-base font-semibold">Format & Account Settings</CardTitle>
              <CardDescription className="text-xs">
                Auto-sniffing detects bank, account mask & credit cards automatically
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div>
                <label className="text-xs font-medium text-muted-foreground mb-1.5 block">
                  Bank Parser Engine
                </label>
                <Select value={parserId} onValueChange={(val) => setParserId(val || 'AUTO')}>
                  <SelectTrigger className="text-xs w-full">
                    <SelectValue placeholder="Auto-Detect Bank & Format" />
                  </SelectTrigger>
                  <SelectContent className="min-w-[var(--anchor-width)] w-max max-w-sm sm:max-w-md">
                    <SelectItem value="AUTO">✨ Auto-Detect Bank & Format (Recommended)</SelectItem>
                    {parsers?.map((p) => (
                      <SelectItem key={p.id} value={p.id}>
                        {p.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>

              <div>
                <label className="text-xs font-medium text-muted-foreground mb-1.5 block">
                  Target Account
                </label>
                <Select value={accountId} onValueChange={(val) => setAccountId(val || 'AUTO')}>
                  <SelectTrigger className="text-xs w-full">
                    <SelectValue placeholder="Auto-Create / Resolve from File" />
                  </SelectTrigger>
                  <SelectContent className="min-w-[var(--anchor-width)] w-max max-w-sm sm:max-w-md">
                    <SelectItem value="AUTO">🏦 Auto-Resolve / Create Account</SelectItem>
                    {accounts?.map((a) => (
                      <SelectItem key={a.id} value={a.id}>
                        {a.bank_name} ({a.account_type}) {a.account_number ? `• A/C ${a.account_number}` : a.account_number_mask ? `• ${a.account_number_mask}` : ''}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            </CardContent>
          </div>

          <CardContent className="pt-0">
            <div className="rounded-lg border border-border/70 bg-muted/30 p-3.5 flex items-center justify-between gap-3">
              <div>
                <h5 className="text-xs font-semibold text-foreground flex items-center gap-1.5">
                  <Sliders className="h-3.5 w-3.5 text-primary" /> Settings & Configuration
                </h5>
                <p className="text-[11px] text-muted-foreground mt-0.5">
                  Manage categories, auto-tagging rules & local database storage.
                </p>
              </div>
              <Link to="/settings">
                <Button variant="outline" size="sm" className="h-8 text-xs font-semibold shrink-0">
                  Settings
                </Button>
              </Link>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Batch Files Processing & Preview Queue */}
      {queuedFiles.length > 0 && (
        <Card className="border-border/80 bg-card shadow-xs">
          <CardHeader className="pb-3">
            <div className="flex flex-wrap items-center justify-between gap-4">
              <div>
                <CardTitle className="text-base font-semibold flex items-center gap-2">
                  <Layers className="h-4 w-4 text-primary" /> Staged Statements ({queuedFiles.length})
                </CardTitle>
                <CardDescription className="text-xs">
                  Review parsed summary, preview transaction rows, and submit batch import
                </CardDescription>
              </div>
              <div className="flex items-center gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={clearQueue}
                  disabled={isBatchImporting}
                  className="h-8 text-xs"
                >
                  Clear Queue
                </Button>
                <Button
                  size="sm"
                  onClick={handleImportAllReady}
                  disabled={readyCount === 0 || isBatchImporting}
                  className="h-8 text-xs font-semibold gap-1.5"
                >
                  {isBatchImporting ? (
                    <>
                      <RefreshCw className="h-3.5 w-3.5 animate-spin" />
                      Importing Batch...
                    </>
                  ) : (
                    <>
                      <UploadCloud className="h-3.5 w-3.5" />
                      Import {readyCount} {readyCount === 1 ? 'Statement' : 'Statements'} ({totalTxsInReady} Txs)
                    </>
                  )}
                </Button>
              </div>
            </div>
          </CardHeader>
          <CardContent className="pt-0">
            {lockedCount > 0 && (
              <div className="mb-4 rounded-lg border border-amber-500/30 bg-amber-500/10 p-3.5 flex flex-col md:flex-row items-start md:items-center justify-between gap-3 shadow-xs">
                <div className="flex items-center gap-2.5">
                  <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-md bg-amber-500/20 text-amber-600 dark:text-amber-400">
                    <KeyRound className="h-4 w-4" />
                  </div>
                  <div>
                    <div className="flex items-center gap-2">
                      <p className="text-xs font-semibold text-foreground">
                        {lockedCount === 1
                          ? '1 Encrypted PDF Statement'
                          : `${lockedCount} Encrypted PDF Statements`}
                      </p>
                      <Badge
                        variant="outline"
                        className="text-[10px] bg-amber-500/15 text-amber-700 dark:text-amber-300 border-amber-500/30 font-medium"
                      >
                        Password Required
                      </Badge>
                    </div>
                    <p className="text-[11px] text-muted-foreground mt-0.5">
                      {lockedCount === 1
                        ? 'Enter password to unlock this encrypted statement.'
                        : 'Enter master password once to unlock all locked PDF statements at once.'}
                    </p>
                  </div>
                </div>

                <form
                  onSubmit={(e) => {
                    e.preventDefault()
                    handleBulkPasswordSubmit()
                  }}
                  className="flex items-center gap-2 w-full md:w-auto"
                >
                  <div className="relative flex-1 md:w-56">
                    <Input
                      type={showBulkPassword ? 'text' : 'password'}
                      placeholder="Enter PDF password"
                      className="h-8 text-xs pr-8 bg-background"
                      value={bulkPassword}
                      onChange={(e) => setBulkPassword(e.target.value)}
                      disabled={isBulkUnlocking}
                    />
                    <button
                      type="button"
                      onClick={() => setShowBulkPassword(!showBulkPassword)}
                      className="absolute right-2 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground focus:outline-none"
                      tabIndex={-1}
                      title={showBulkPassword ? 'Hide password' : 'Show password'}
                    >
                      {showBulkPassword ? <EyeOff className="h-3.5 w-3.5" /> : <Eye className="h-3.5 w-3.5" />}
                    </button>
                  </div>
                  <Button
                    type="submit"
                    size="sm"
                    disabled={!bulkPassword.trim() || isBulkUnlocking}
                    className="h-8 text-xs shrink-0 gap-1.5 font-semibold"
                  >
                    {isBulkUnlocking ? (
                      <>
                        <RefreshCw className="h-3.5 w-3.5 animate-spin" />
                        Unlocking...
                      </>
                    ) : (
                      <>
                        <Unlock className="h-3.5 w-3.5" />
                        {lockedCount === 1 ? 'Unlock' : `Unlock All (${lockedCount})`}
                      </>
                    )}
                  </Button>
                </form>
              </div>
            )}

            <div className="space-y-3">
              {queuedFiles.map((qFile) => (
                <div
                  key={qFile.id}
                  className="rounded-lg border border-border/70 bg-muted/30 p-4 transition-colors hover:border-border"
                >
                  <div className="flex flex-col md:flex-row md:items-center justify-between gap-3">
                    {/* Left File Info */}
                    <div className="flex items-start gap-3 min-w-0">
                      <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-background border text-primary">
                        <FileText className="h-5 w-5" />
                      </div>
                      <div className="min-w-0">
                        <div className="flex items-center gap-2 flex-wrap">
                          <p className="text-xs font-semibold text-foreground truncate max-w-sm">
                            {qFile.file.name}
                          </p>
                          <span className="text-[10px] font-mono text-muted-foreground">
                            {(qFile.file.size / 1024).toFixed(1)} KB
                          </span>
                          {qFile.preview?.account_type && (
                            <Badge variant="outline" className="text-[10px] uppercase font-semibold">
                              {qFile.preview.bank_name} • {qFile.preview.card_variant || qFile.preview.account_type}
                            </Badge>
                          )}
                          {(qFile.preview?.account_number || qFile.preview?.account_number_mask) && (
                            <Badge variant="secondary" className="text-[10px] font-mono">
                              {qFile.preview?.account_number ? `A/C ${qFile.preview.account_number}` : qFile.preview?.account_number_mask}
                            </Badge>
                          )}
                          {qFile.preview?.account_holder_name && (
                            <Badge variant="outline" className="text-[10px] font-semibold border-primary/30 text-primary">
                              {qFile.preview.account_holder_name}
                            </Badge>
                          )}
                        </div>

                        {qFile.preview && qFile.status === 'ready' && (
                          <div className="flex items-center gap-4 mt-1 text-[11px] text-muted-foreground flex-wrap">
                            <span>
                              <strong className="text-foreground font-mono">{qFile.preview.total_transactions}</strong> transactions
                              {' '}(<span className="text-emerald-400 font-mono font-semibold">{qFile.preview.new_txs_count} new</span>
                              {qFile.preview.existing_txs_count > 0 && `, ${qFile.preview.existing_txs_count} existing`})
                            </span>
                            {qFile.preview.start_date && qFile.preview.end_date && (
                              <span className="flex items-center gap-1 font-mono">
                                <Calendar className="h-3 w-3" /> {formatDate(qFile.preview.start_date)} – {formatDate(qFile.preview.end_date)}
                              </span>
                            )}
                            {qFile.preview.total_due_amount > 0 && (
                              <span className="font-mono">
                                Total Due: <strong className="text-foreground">{formatINR(qFile.preview.total_due_amount)}</strong>
                              </span>
                            )}
                          </div>
                        )}

                        {qFile.status === 'error' && qFile.error && (
                          <p className="text-xs text-destructive mt-1 flex items-center gap-1">
                            <AlertCircle className="h-3.5 w-3.5 shrink-0" /> {qFile.error}
                          </p>
                        )}
                      </div>
                    </div>

                    {/* Right Actions & Status */}
                    <div className="flex items-center gap-2 self-end md:self-center shrink-0">
                      {qFile.status === 'previewing' && (
                        <div className="flex items-center gap-1.5 text-xs text-muted-foreground font-medium">
                          <RefreshCw className="h-3.5 w-3.5 animate-spin text-primary" /> Sniffing format...
                        </div>
                      )}

                      {qFile.status === 'password_required' && (
                        <div className="flex items-center gap-1.5 flex-wrap">
                          <Input
                            type="password"
                            placeholder="Enter PDF password"
                            className="h-8 text-xs w-36 sm:w-44 bg-background"
                            defaultValue={qFile.password || ''}
                            onKeyDown={(e) => {
                              if (e.key === 'Enter') {
                                handlePasswordSubmit(qFile, (e.target as HTMLInputElement).value)
                              }
                            }}
                            id={`pwd-${qFile.id}`}
                          />
                          <Button
                            size="sm"
                            variant="secondary"
                            className="h-8 text-xs gap-1"
                            onClick={() => {
                              const input = document.getElementById(`pwd-${qFile.id}`) as HTMLInputElement
                              if (input) handlePasswordSubmit(qFile, input.value)
                            }}
                          >
                            <Unlock className="h-3 w-3" /> Unlock
                          </Button>
                          {lockedCount > 1 && (
                            <Button
                              size="sm"
                              variant="outline"
                              className="h-8 text-xs gap-1"
                              title="Apply this password to all locked statements"
                              onClick={() => {
                                const input = document.getElementById(`pwd-${qFile.id}`) as HTMLInputElement
                                if (input && input.value) {
                                  handleBulkPasswordSubmit(input.value)
                                }
                              }}
                            >
                              Apply to All ({lockedCount})
                            </Button>
                          )}
                        </div>
                      )}

                      {qFile.status === 'ready' && (
                        <>
                          <Button
                            size="sm"
                            variant="outline"
                            className="h-8 text-xs gap-1.5 font-medium border-border/80 hover:bg-background"
                            onClick={() => {
                              setActivePreviewFile(qFile)
                              setPreviewSearch('')
                            }}
                          >
                            <Eye className="h-3.5 w-3.5 text-primary" />
                            <span>Preview Records</span>
                          </Button>
                          <Button
                            size="sm"
                            className="h-8 text-xs font-semibold"
                            onClick={() => importSingleFile(qFile)}
                          >
                            Import
                          </Button>
                        </>
                      )}

                      {qFile.status === 'importing' && (
                        <Badge variant="secondary" className="gap-1 text-xs">
                          <RefreshCw className="h-3 w-3 animate-spin text-primary" /> Importing...
                        </Badge>
                      )}

                      {qFile.status === 'imported' && (
                        <Badge className="bg-emerald-500/10 text-emerald-400 border-emerald-500/20 gap-1 text-xs">
                          <CheckCircle2 className="h-3 w-3 text-emerald-400" /> Imported ({qFile.result?.inserted_count} new)
                        </Badge>
                      )}

                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => removeFileFromQueue(qFile.id)}
                        className="h-8 w-8 p-0 text-muted-foreground hover:text-destructive"
                      >
                        <X className="h-4 w-4" />
                      </Button>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      )}

      {/* Audit History Table */}
      <Card className="border-border/80 bg-card shadow-xs">
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle className="text-base font-semibold flex items-center gap-2">
                <History className="h-4 w-4 text-primary" /> Statement Audit History
              </CardTitle>
              <CardDescription className="text-xs">
                Log of all uploaded statement files, checksums, and totals
              </CardDescription>
            </div>
            <Badge variant="secondary" className="font-mono text-xs">
              {statementsHistory?.length || 0} Imported
            </Badge>
          </div>
        </CardHeader>
        <CardContent className="pt-0">
          <div className="overflow-x-auto">
            <Table>
              <TableHeader>
                <TableRow className="hover:bg-transparent">
                  <TableHead className="text-xs font-semibold uppercase">Filename</TableHead>
                  <TableHead className="text-xs font-semibold uppercase">Account</TableHead>
                  <TableHead className="text-xs font-semibold uppercase">Parser</TableHead>
                  <TableHead className="text-xs font-semibold uppercase">Period</TableHead>
                  <TableHead className="text-xs font-semibold uppercase text-right">Transactions</TableHead>
                  <TableHead className="text-xs font-semibold uppercase">Imported At</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {statementsHistory && statementsHistory.length > 0 ? (
                  statementsHistory.map((s) => (
                    <TableRow key={s.id} className="hover:bg-muted/40 text-xs">
                      <TableCell className="font-medium text-foreground max-w-xs">
                        <Tooltip>
                          <TooltipTrigger
                            render={<span tabIndex={0} />}
                            className="block truncate rounded-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                          >
                            {s.filename}
                          </TooltipTrigger>
                          <TooltipContent className="max-w-[min(32rem,calc(100vw-2rem))] whitespace-normal break-all">
                            {s.filename}
                          </TooltipContent>
                        </Tooltip>
                      </TableCell>
                      <TableCell className="text-muted-foreground">{s.bank_name || 'Bank Account'}</TableCell>
                      <TableCell>
                        <Badge variant="outline" className="text-[10px] font-mono">
                          {s.parser_used}
                        </Badge>
                      </TableCell>
                      <TableCell className="font-mono text-muted-foreground">
                        {s.start_date && s.end_date
                          ? `${formatDate(s.start_date)} - ${formatDate(s.end_date)}`
                          : '-'}
                      </TableCell>
                      <TableCell className="font-mono font-bold text-right text-foreground">
                        {s.total_transactions}
                      </TableCell>
                      <TableCell className="font-mono text-muted-foreground">
                        {formatDate(s.imported_at)}
                      </TableCell>
                    </TableRow>
                  ))
                ) : (
                  <TableRow>
                    <TableCell colSpan={6} className="h-24 text-center text-xs text-muted-foreground">
                      No statements uploaded yet.
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          </div>
        </CardContent>
      </Card>

      {/* FULL PARSED DATA PREVIEW DIALOG MODAL */}
      <Dialog open={!!activePreviewFile} onOpenChange={(open) => !open && setActivePreviewFile(null)}>
        <DialogContent className="sm:max-w-5xl md:max-w-6xl w-[95vw] max-h-[90vh] flex flex-col p-6 overflow-hidden">
          <DialogHeader className="pb-3 border-b">
            <div className="flex items-center justify-between gap-4">
              <div>
                <DialogTitle className="text-lg font-bold flex items-center gap-2">
                  <Eye className="h-5 w-5 text-primary" /> Parsed Statement Preview
                </DialogTitle>
                <DialogDescription className="text-xs mt-0.5">
                  Inspect extracted transactions, account summary, and metadata before committing to database
                </DialogDescription>
              </div>
              {activePreviewFile?.preview && (
                <div className="flex items-center gap-2">
                  <Badge variant="outline" className="font-mono text-xs">
                    {activePreviewFile.preview.bank_name}
                  </Badge>
                  <Badge variant="secondary" className="font-mono text-xs">
                    {activePreviewFile.preview.total_transactions} Records
                  </Badge>
                </div>
              )}
            </div>
          </DialogHeader>

          {activePreviewFile?.preview && (
            <div className="flex-1 overflow-y-auto space-y-4 py-2 pr-1">
              {/* Statement Summary Ribbon */}
              <div className="grid grid-cols-2 sm:grid-cols-4 gap-3 text-xs">
                <div className="rounded-lg bg-muted/40 border p-3">
                  <span className="text-[11px] text-muted-foreground uppercase font-medium">Bank & Account</span>
                  <p className="font-semibold text-foreground mt-0.5 text-sm">{activePreviewFile.preview.bank_name}</p>
                  <p className="text-xs text-muted-foreground font-mono mt-0.5">
                    {activePreviewFile.preview.card_variant || activePreviewFile.preview.account_type}{' '}
                    {activePreviewFile.preview.account_number
                      ? `• A/C ${activePreviewFile.preview.account_number}`
                      : activePreviewFile.preview.account_number_mask
                      ? `• ${activePreviewFile.preview.account_number_mask}`
                      : ''}
                  </p>
                  {activePreviewFile.preview.account_holder_name && (
                    <p className="text-[11px] text-primary font-medium mt-1">
                      Holder: <strong className="text-foreground">{activePreviewFile.preview.account_holder_name}</strong>
                    </p>
                  )}
                </div>
                <div className="rounded-lg bg-muted/40 border p-3">
                  <span className="text-[11px] text-muted-foreground uppercase font-medium">
                    {activePreviewFile.preview.account_type === 'CREDIT_CARD' ? 'Billing Period & Due' : 'Statement Period'}
                  </span>
                  <p className="font-semibold font-mono text-foreground mt-0.5 text-xs">
                    {activePreviewFile.preview.start_date && activePreviewFile.preview.end_date
                      ? `${formatDate(activePreviewFile.preview.start_date)} – ${formatDate(activePreviewFile.preview.end_date)}`
                      : 'N/A'}
                  </p>
                  {activePreviewFile.preview.payment_due_date && (
                    <p className="text-xs font-mono text-primary font-medium mt-0.5">
                      Due Date: {formatDate(activePreviewFile.preview.payment_due_date)}
                    </p>
                  )}
                </div>
                <div className="rounded-lg bg-muted/40 border p-3">
                  <span className="text-[11px] text-muted-foreground uppercase font-medium">
                    {activePreviewFile.preview.account_type === 'CREDIT_CARD' ? 'Total Due Amount' : 'Closing Balance'}
                  </span>
                  <p className="font-bold font-mono text-foreground text-base mt-0.5">
                    {activePreviewFile.preview.account_type === 'CREDIT_CARD'
                      ? formatINR(activePreviewFile.preview.total_due_amount)
                      : formatINR(activePreviewFile.preview.closing_balance)}
                  </p>
                  {activePreviewFile.preview.minimum_due_amount > 0 && (
                    <p className="text-[11px] font-mono text-muted-foreground mt-0.5">
                      Min Due: {formatINR(activePreviewFile.preview.minimum_due_amount)}
                    </p>
                  )}
                  {activePreviewFile.preview.credit_limit > 0 && (
                    <p className="text-[11px] font-mono text-muted-foreground mt-0.5">
                      Limit: {formatINR(activePreviewFile.preview.credit_limit)}
                    </p>
                  )}
                </div>
                <div className="rounded-lg bg-muted/40 border p-3">
                  <span className="text-[11px] text-muted-foreground uppercase font-medium">Idempotent Audit</span>
                  <p className="font-bold font-mono text-emerald-400 text-base mt-0.5">
                    {activePreviewFile.preview.new_txs_count} New
                  </p>
                  <p className="text-[11px] font-mono text-muted-foreground mt-0.5">
                    {activePreviewFile.preview.existing_txs_count} Existing in Database
                  </p>
                </div>
              </div>

              {/* Search Bar */}
              <div className="flex items-center justify-between gap-3">
                <div className="relative flex-1 max-w-sm">
                  <Search className="absolute left-2.5 top-2.5 h-3.5 w-3.5 text-muted-foreground" />
                  <Input
                    placeholder="Search payee, narration, ref, mode..."
                    value={previewSearch}
                    onChange={(e) => setPreviewSearch(e.target.value)}
                    className="pl-8 text-xs h-8"
                  />
                </div>
                <span className="text-xs text-muted-foreground font-mono">
                  Showing {filteredPreviewTransactions.length} of {activePreviewFile.preview.transactions.length} records
                </span>
              </div>

              {/* Parsed Transactions Table */}
              <div className="rounded-lg border overflow-hidden">
                <div className="max-h-[48vh] overflow-y-auto">
                  <Table className="min-w-[760px]">
                    <TableHeader className="sticky top-0 bg-muted/80 backdrop-blur-xs z-10">
                      <TableRow className="hover:bg-transparent">
                        <TableHead className="text-[11px] font-semibold uppercase w-24">Date</TableHead>
                        <TableHead className="text-[11px] font-semibold uppercase w-20">Type</TableHead>
                        <TableHead className="text-[11px] font-semibold uppercase text-right w-28">Amount</TableHead>
                        <TableHead className="text-[11px] font-semibold uppercase">Payee / Description</TableHead>
                        <TableHead className="text-[11px] font-semibold uppercase w-24">Mode</TableHead>
                        <TableHead className="text-[11px] font-semibold uppercase w-28">Category</TableHead>
                        <TableHead className="text-[11px] font-semibold uppercase w-28">Ref No</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {filteredPreviewTransactions.map((tx, idx) => {
                        const isCredit = tx.tx_type === 'CREDIT'
                        return (
                          <TableRow key={idx} className="hover:bg-muted/40 text-xs">
                            <TableCell className="font-mono text-muted-foreground whitespace-nowrap">
                              {formatDate(tx.date)}
                            </TableCell>
                            <TableCell>
                              <Badge
                                variant={isCredit ? 'secondary' : 'outline'}
                                className={`text-[10px] font-semibold uppercase px-1.5 py-0 ${
                                  isCredit
                                    ? 'bg-emerald-500/10 text-emerald-400 border-emerald-500/20'
                                    : 'text-muted-foreground'
                                }`}
                              >
                                {tx.tx_type}
                              </Badge>
                            </TableCell>
                            <TableCell className="text-right font-mono font-bold tabular-nums whitespace-nowrap">
                              <span className={isCredit ? 'text-emerald-400' : 'text-foreground'}>
                                {isCredit ? '+' : ''} {formatINR(tx.amount)}
                              </span>
                            </TableCell>
                            <TableCell className="max-w-xs">
                              <div className="flex items-center gap-2">
                                <MerchantAvatar payee={tx.cleaned_payee || tx.raw_narration} size="xs" />
                                <div className="min-w-0 flex-1">
                                  <div className="font-medium text-foreground truncate">{tx.cleaned_payee || tx.raw_narration}</div>
                                  <div className="text-[10px] text-muted-foreground truncate font-mono mt-0.5 flex items-center gap-1">
                                    <span className="truncate">{tx.raw_narration}</span>
                                    <CopyButton value={tx.raw_narration} label="Copy Narration" />
                                  </div>
                                </div>
                              </div>
                            </TableCell>
                            <TableCell>
                              <Badge variant="outline" className="text-[10px] font-mono">
                                {tx.payment_mode}
                              </Badge>
                            </TableCell>
                            <TableCell>
                              {tx.category_name ? (
                                <Badge variant="secondary" className="text-[10px]">
                                  {tx.category_name}
                                </Badge>
                              ) : (
                                <span className="text-muted-foreground text-[11px]">-</span>
                              )}
                            </TableCell>
                            <TableCell className="font-mono text-muted-foreground text-[11px]">
                              {tx.reference_number ? (
                                <div className="flex items-center gap-1">
                                  <span className="truncate max-w-[80px]">{tx.reference_number}</span>
                                  <CopyButton value={tx.reference_number} label="Copy Ref" />
                                </div>
                              ) : (
                                '-'
                              )}
                            </TableCell>
                          </TableRow>
                        )
                      })}
                    </TableBody>
                  </Table>
                </div>
              </div>
            </div>
          )}

          <DialogFooter className="border-t pt-3 flex items-center justify-between gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={() => setActivePreviewFile(null)}
              className="text-xs"
            >
              Close Preview
            </Button>
            {activePreviewFile?.status === 'ready' && (
              <Button
                size="sm"
                className="text-xs font-semibold gap-1.5"
                onClick={() => {
                  if (activePreviewFile) {
                    importSingleFile(activePreviewFile)
                    setActivePreviewFile(null)
                  }
                }}
              >
                <UploadCloud className="h-3.5 w-3.5" />
                Import This Statement Now
              </Button>
            )}
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
