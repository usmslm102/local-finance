import React, { useState, useRef, useCallback } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useSearch, useNavigate, useRouterState } from '@tanstack/react-router'
import {
  fetchAccounts,
  fetchAnalytics,
  fetchCategories,
  fetchDatabaseInfo,
  fetchParsers,
  fetchRules,
  fetchStatements,
  resetDatabase,
  restoreDatabase,
  createRule,
  updateRule,
  deleteRule,
  reapplyRules,
  createCategory,
  deleteCategory,
  setupAuth,
  changeAuthPassword,
  disableAuth,
  updateSecuritySettings,
} from '@/lib/api'
import { useSystemUpdate } from '@/hooks/use-system-update'
import { MCPSettingsPanel } from './MCPSettingsPanel'
import { Switch } from '@/components/ui/switch'
import { useAuth } from '@/components/auth/AuthProvider'
import { useTheme } from '@/components/theme-provider'
import type { SettingsSearchParams } from '@/types'
import { formatDate } from '@/lib/utils'
import { CategorizationRule, Category } from '@/types'
import {
  Sliders,
  Tags,
  ShieldCheck,
  ShieldAlert,
  Cpu,
  Trash2,
  Database,
  CheckCircle2,
  AlertCircle,
  Sun,
  Moon,
  Laptop,
  FileSpreadsheet,
  FileCode2,
  Lock,
  Unlock,
  KeyRound,
  Download,
  UploadCloud,
  Copy,
  Check,
  HardDriveDownload,
  FolderOpen,
  RefreshCw,
  Plus,
  Edit2,
  Volume2,
  VolumeX,
} from 'lucide-react'
import {
  isSoundEnabled,
  setSoundEnabled,
  getSoundVolume,
  setSoundVolume,
  playSuccessChime,
  playSoftClick,
  playPrivacyToggleSound,
} from '@/lib/audio'
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { UpdateDialog } from '@/components/updates/UpdateDialog'
import { ArrowUpCircle, ExternalLink, Sparkles } from 'lucide-react'
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

type SettingsTab = 'general' | 'security' | 'rules' | 'parsers' | 'database' | 'mcp'

interface SettingsViewProps {
  initialTab?: SettingsTab
}

function formatBytes(bytes: number): string {
  if (!bytes || bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(1))} ${sizes[i]}`
}

const PRESET_COLORS = [
  '#10B981', '#3B82F6', '#8B5CF6', '#EC4899',
  '#F59E0B', '#EF4444', '#06B6D4', '#6366F1',
  '#14B8A6', '#84CC16', '#64748B', '#D97706',
]

export const SettingsView: React.FC<SettingsViewProps> = ({ initialTab = 'general' }) => {
  const queryClient = useQueryClient()
  const { authStatus, refetchAuth, lockApp } = useAuth()
  const searchParams = (useSearch({ strict: false }) as SettingsSearchParams) || {}
  const navigate = useNavigate()
  const routerState = useRouterState()
  const currentPath = routerState.location.pathname

  const activeTab: SettingsTab =
    (searchParams.tab as SettingsTab) || (initialTab as SettingsTab) || 'general'

  const handleTabChange = useCallback(
    (tabId: SettingsTab) => {
      navigate({
        to: currentPath as any,
        search: (tabId !== 'general' ? { tab: tabId } : {}) as any,
        replace: false,
      })
    },
    [navigate, currentPath]
  )

  const [showConfirmReset, setShowConfirmReset] = useState(false)
  const [selectedRestoreFile, setSelectedRestoreFile] = useState<File | null>(null)
  const [showConfirmRestore, setShowConfirmRestore] = useState(false)
  const [copiedPath, setCopiedPath] = useState(false)
  const [statusMessage, setStatusMessage] = useState<{ type: 'success' | 'error'; title: string; text: string } | null>(null)

  // Software Update States via shared hook
  const {
    versionInfo,
    dialogOpen: showUpdateModal,
    setDialogOpen: setShowUpdateModal,
    isChecking: isCheckingUpdate,
    checkError: updateCheckError,
    autoCheckEnabled,
    setAutoCheckEnabled,
    checkNow: handleManualCheckUpdate,
    refetch: refetchVersion,
  } = useSystemUpdate()

  // Security Form States
  const [showSetupAuthModal, setShowSetupAuthModal] = useState(false)
  const [showChangePasswordModal, setShowChangePasswordModal] = useState(false)
  const [showDisableAuthModal, setShowDisableAuthModal] = useState(false)

  const [setupPassword, setSetupPassword] = useState('')
  const [setupConfirmPassword, setSetupConfirmPassword] = useState('')
  const [setupAutoLock, setSetupAutoLock] = useState(60)

  const [changeCurrentPassword, setChangeCurrentPassword] = useState('')
  const [changeNewPassword, setChangeNewPassword] = useState('')
  const [changeConfirmPassword, setChangeConfirmPassword] = useState('')

  const [disablePassword, setDisablePassword] = useState('')
  const [securityActionError, setSecurityActionError] = useState<string | null>(null)

  // Sound Preferences State
  const [soundEnabled, setSoundEnabledState] = useState(isSoundEnabled)
  const [soundVol, setSoundVolState] = useState(() => Math.round(getSoundVolume() * 100))

  // Rule Form State
  const [showAddRuleModal, setShowAddRuleModal] = useState(false)
  const [editingRule, setEditingRule] = useState<CategorizationRule | null>(null)
  const [showDeleteRuleModal, setShowDeleteRuleModal] = useState<CategorizationRule | null>(null)
  const [ruleFormData, setRuleFormData] = useState<{
    priority: number
    match_field: string
    match_type: string
    match_pattern: string
    exclude_pattern: string
    tx_type: 'ALL' | 'DEBIT' | 'CREDIT'
    target_category_id: string
    assign_tags: string
    is_active: boolean
  }>({
    priority: 50,
    match_field: 'cleaned_payee',
    match_type: 'CONTAINS',
    match_pattern: '',
    exclude_pattern: '',
    tx_type: 'ALL',
    target_category_id: '',
    assign_tags: '',
    is_active: true,
  })

  // Category Form State
  const [showAddCategoryModal, setShowAddCategoryModal] = useState(false)
  const [showDeleteCategoryModal, setShowDeleteCategoryModal] = useState<Category | null>(null)
  const [categoryFormData, setCategoryFormData] = useState<{
    name: string
    color_hex: string
    icon: string
  }>({
    name: '',
    color_hex: '#10B981',
    icon: 'tag',
  })

  const restoreFileInputRef = useRef<HTMLInputElement>(null)
  const { theme, setTheme } = useTheme()

  const { data: analytics } = useQuery({
    queryKey: ['analytics'],
    queryFn: fetchAnalytics,
  })

  const { data: accounts } = useQuery({
    queryKey: ['accounts'],
    queryFn: fetchAccounts,
  })

  const { data: categories } = useQuery({
    queryKey: ['categories'],
    queryFn: fetchCategories,
  })

  const { data: rules } = useQuery({
    queryKey: ['rules'],
    queryFn: fetchRules,
  })

  const { data: parsers } = useQuery({
    queryKey: ['parsers'],
    queryFn: fetchParsers,
  })

  const { data: statementsHistory } = useQuery({
    queryKey: ['statements'],
    queryFn: fetchStatements,
  })

  const { data: dbInfo, refetch: refetchDbInfo } = useQuery({
    queryKey: ['database-info'],
    queryFn: fetchDatabaseInfo,
  })

  // Rule Mutations
  const createRuleMutation = useMutation({
    mutationFn: (data: Partial<CategorizationRule>) => createRule(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['rules'] })
      setShowAddRuleModal(false)
      setStatusMessage({
        type: 'success',
        title: 'Rule Created',
        text: 'Custom auto-categorization rule added successfully!',
      })
      setTimeout(() => setStatusMessage(null), 4000)
    },
    onError: (err: any) => {
      setStatusMessage({
        type: 'error',
        title: 'Rule Creation Failed',
        text: err.message || 'Failed to create rule',
      })
    },
  })

  const updateRuleMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<CategorizationRule> }) =>
      updateRule(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['rules'] })
      setEditingRule(null)
      setStatusMessage({
        type: 'success',
        title: 'Rule Updated',
        text: 'Categorization rule changes saved successfully!',
      })
      setTimeout(() => setStatusMessage(null), 4000)
    },
    onError: (err: any) => {
      setStatusMessage({
        type: 'error',
        title: 'Rule Update Failed',
        text: err.message || 'Failed to update rule',
      })
    },
  })

  const deleteRuleMutation = useMutation({
    mutationFn: (id: string) => deleteRule(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['rules'] })
      setShowDeleteRuleModal(null)
      setStatusMessage({
        type: 'success',
        title: 'Rule Deleted',
        text: 'Categorization rule removed successfully!',
      })
      setTimeout(() => setStatusMessage(null), 4000)
    },
    onError: (err: any) => {
      setStatusMessage({
        type: 'error',
        title: 'Delete Failed',
        text: err.message || 'Failed to delete rule',
      })
    },
  })

  const reapplyRulesMutation = useMutation({
    mutationFn: reapplyRules,
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: ['transactions'] })
      queryClient.invalidateQueries({ queryKey: ['analytics'] })
      setStatusMessage({
        type: 'success',
        title: 'Rules Re-Applied',
        text: `Successfully re-categorized ${data.updated_count} transactions across your ledger!`,
      })
      setTimeout(() => setStatusMessage(null), 5000)
    },
    onError: (err: any) => {
      setStatusMessage({
        type: 'error',
        title: 'Re-Apply Failed',
        text: err.message || 'Failed to re-apply rules to transactions',
      })
    },
  })

  // Category Mutations
  const createCategoryMutation = useMutation({
    mutationFn: (data: Partial<Category>) => createCategory(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['categories'] })
      setShowAddCategoryModal(false)
      setStatusMessage({
        type: 'success',
        title: 'Category Created',
        text: 'Custom spending category added successfully!',
      })
      setTimeout(() => setStatusMessage(null), 4000)
    },
    onError: (err: any) => {
      setStatusMessage({
        type: 'error',
        title: 'Category Creation Failed',
        text: err.message || 'Failed to create category',
      })
    },
  })

  const deleteCategoryMutation = useMutation({
    mutationFn: (id: string) => deleteCategory(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['categories'] })
      queryClient.invalidateQueries({ queryKey: ['rules'] })
      queryClient.invalidateQueries({ queryKey: ['transactions'] })
      setShowDeleteCategoryModal(null)
      setStatusMessage({
        type: 'success',
        title: 'Category Deleted',
        text: 'Category deleted successfully!',
      })
      setTimeout(() => setStatusMessage(null), 4000)
    },
    onError: (err: any) => {
      setStatusMessage({
        type: 'error',
        title: 'Delete Failed',
        text: err.message || 'Failed to delete category',
      })
    },
  })

  const handleOpenAddRule = () => {
    setRuleFormData({
      priority: 50,
      match_field: 'cleaned_payee',
      match_type: 'CONTAINS',
      match_pattern: '',
      exclude_pattern: '',
      tx_type: 'ALL',
      target_category_id: categories?.[0]?.id || '',
      assign_tags: '',
      is_active: true,
    })
    setShowAddRuleModal(true)
  }

  const handleOpenEditRule = (r: CategorizationRule) => {
    setRuleFormData({
      priority: r.priority,
      match_field: r.match_field,
      match_type: r.match_type,
      match_pattern: r.match_pattern,
      exclude_pattern: r.exclude_pattern || '',
      tx_type: r.tx_type || 'ALL',
      target_category_id: r.target_category_id,
      assign_tags: r.assign_tags || '',
      is_active: r.is_active,
    })
    setEditingRule(r)
  }

  const handleOpenAddCategory = () => {
    setCategoryFormData({
      name: '',
      color_hex: '#10B981',
      icon: 'tag',
    })
    setShowAddCategoryModal(true)
  }

  const resetMutation = useMutation({
    mutationFn: resetDatabase,
    onSuccess: (data) => {
      setStatusMessage({
        type: 'success',
        title: 'Database Reset',
        text: data.message || 'Database reset successfully! All categories and custom rules have been preserved.',
      })
      setShowConfirmReset(false)
      queryClient.invalidateQueries()
      setTimeout(() => setStatusMessage(null), 6000)
    },
    onError: (err: any) => {
      setStatusMessage({
        type: 'error',
        title: 'Reset Failed',
        text: err.message || 'Failed to reset database',
      })
      setShowConfirmReset(false)
    },
  })

  const restoreMutation = useMutation({
    mutationFn: (file: File) => restoreDatabase(file),
    onSuccess: (data) => {
      setStatusMessage({
        type: 'success',
        title: 'Database Restored',
        text: data.message || 'Database restored successfully from backup!',
      })
      setShowConfirmRestore(false)
      setSelectedRestoreFile(null)
      if (restoreFileInputRef.current) restoreFileInputRef.current.value = ''
      queryClient.invalidateQueries()
      refetchDbInfo()
      setTimeout(() => setStatusMessage(null), 6000)
    },
    onError: (err: any) => {
      setStatusMessage({
        type: 'error',
        title: 'Restore Failed',
        text: err.message || 'Failed to restore database from backup file',
      })
      setShowConfirmRestore(false)
      setSelectedRestoreFile(null)
      if (restoreFileInputRef.current) restoreFileInputRef.current.value = ''
    },
  })

  // Security Mutations
  const setupAuthMutation = useMutation({
    mutationFn: () => {
      if (setupPassword !== setupConfirmPassword) {
        throw new Error('Passwords do not match')
      }
      return setupAuth(setupPassword, setupAutoLock)
    },
    onSuccess: () => {
      refetchAuth()
      setShowSetupAuthModal(false)
      setSetupPassword('')
      setSetupConfirmPassword('')
      setStatusMessage({
        type: 'success',
        title: 'Master Password Configured',
        text: 'Local authentication enabled! Your ledger is now password-protected.',
      })
      setTimeout(() => setStatusMessage(null), 5000)
    },
    onError: (err: any) => {
      setSecurityActionError(err.message || 'Failed to setup master password')
    },
  })

  const changePasswordMutation = useMutation({
    mutationFn: () => {
      if (changeNewPassword !== changeConfirmPassword) {
        throw new Error('New passwords do not match')
      }
      return changeAuthPassword(changeCurrentPassword, changeNewPassword)
    },
    onSuccess: () => {
      refetchAuth()
      setShowChangePasswordModal(false)
      setChangeCurrentPassword('')
      setChangeNewPassword('')
      setChangeConfirmPassword('')
      setStatusMessage({
        type: 'success',
        title: 'Password Updated',
        text: 'Your master password has been changed successfully.',
      })
      setTimeout(() => setStatusMessage(null), 5000)
    },
    onError: (err: any) => {
      setSecurityActionError(err.message || 'Failed to change password')
    },
  })

  const disableAuthMutation = useMutation({
    mutationFn: () => disableAuth(disablePassword),
    onSuccess: () => {
      refetchAuth()
      setShowDisableAuthModal(false)
      setDisablePassword('')
      setStatusMessage({
        type: 'success',
        title: 'Security Disabled',
        text: 'Local authentication turned off. Anyone on this device can view the app.',
      })
      setTimeout(() => setStatusMessage(null), 5000)
    },
    onError: (err: any) => {
      setSecurityActionError(err.message || 'Incorrect password')
    },
  })

  const updateTimeoutMutation = useMutation({
    mutationFn: (minutes: number) => updateSecuritySettings(minutes),
    onSuccess: () => {
      refetchAuth()
      setStatusMessage({
        type: 'success',
        title: 'Settings Saved',
        text: 'Auto-lock timeout updated successfully.',
      })
      setTimeout(() => setStatusMessage(null), 4000)
    },
  })

  const handleCopyPath = (path: string) => {
    navigator.clipboard.writeText(path)
    setCopiedPath(true)
    setTimeout(() => setCopiedPath(false), 2500)
  }

  const handleRestoreFileSelected = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (file) {
      setSelectedRestoreFile(file)
      setShowConfirmRestore(true)
    }
  }

  const navTabs: { id: SettingsTab; label: string; icon: React.FC<{ className?: string }>; desc: string }[] = [
    {
      id: 'general',
      label: 'General & Preferences',
      icon: Sliders,
      desc: 'Theme, regional formatting & audio',
    },
    {
      id: 'security',
      label: 'Security & App Lock',
      icon: ShieldCheck,
      desc: 'Master password, PIN & database permissions',
    },
    {
      id: 'rules',
      label: 'Categories & Rules',
      icon: Tags,
      desc: 'Classification taxonomy & regex rules',
    },
    {
      id: 'parsers',
      label: 'Bank Parsers',
      icon: Cpu,
      desc: 'Sniffing engines & statement plugins',
    },
    { id: 'mcp', label: 'AI / MCP', icon: ShieldCheck, desc: 'Read-only AI connections' },
    {
      id: 'database',
      label: 'Data & Storage',
      icon: Database,
      desc: 'DB location, backups, export & import',
    },
  ]

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-bold tracking-tight text-foreground sm:text-3xl">
          Settings & Configuration
        </h1>
        <p className="text-xs text-muted-foreground mt-1">
          Manage local database storage, backups, classification rules, parser modules, and interface preferences.
        </p>
      </div>

      {statusMessage && (
        <Alert
          variant={statusMessage.type === 'error' ? 'destructive' : 'default'}
          className={
            statusMessage.type === 'success'
              ? 'border-emerald-500/30 bg-emerald-500/10 text-emerald-400'
              : ''
          }
        >
          {statusMessage.type === 'success' ? (
            <CheckCircle2 className="h-4 w-4 text-emerald-400" />
          ) : (
            <AlertCircle className="h-4 w-4" />
          )}
          <AlertTitle>{statusMessage.title}</AlertTitle>
          <AlertDescription className="text-xs">{statusMessage.text}</AlertDescription>
        </Alert>
      )}

      {/* Main Settings Layout with Sidebar Navigation */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-8 items-start">
        {/* Left Side Tab Navigation */}
        <aside className="lg:col-span-3">
          <nav className="flex lg:flex-col gap-1.5 overflow-x-auto pb-2 lg:pb-0">
            {navTabs.map((tab) => {
              const Icon = tab.icon
              const isActive = activeTab === tab.id
              return (
                <button
                  key={tab.id}
                  onClick={() => handleTabChange(tab.id)}
                  className={`flex items-center gap-3 rounded-lg px-3.5 py-2.5 text-left text-xs transition-colors shrink-0 ${
                    isActive
                      ? 'bg-secondary text-secondary-foreground font-semibold shadow-xs'
                      : 'text-muted-foreground hover:bg-muted/60 hover:text-foreground'
                  }`}
                >
                  <Icon className={`h-4 w-4 ${isActive ? 'text-primary' : 'text-muted-foreground'}`} />
                  <div className="hidden sm:block truncate">
                    <div className="font-medium text-xs text-foreground">{tab.label}</div>
                    <div className="text-[10px] text-muted-foreground truncate">{tab.desc}</div>
                  </div>
                  <span className="sm:hidden font-medium">{tab.label}</span>
                </button>
              )
            })}
          </nav>
        </aside>

        {/* Right Tab Content */}
        <main className="lg:col-span-9 space-y-6">
          {activeTab === 'mcp' && <MCPSettingsPanel />}
          {/* TAB 1: GENERAL & PREFERENCES */}
          {activeTab === 'general' && (
            <div className="space-y-6">
              {/* Software Version & Updates Card */}
              <Card className="border-border/80 bg-card shadow-xs">
                <CardHeader>
                  <div className="flex items-center justify-between flex-wrap gap-2">
                    <div>
                      <CardTitle className="text-base font-semibold flex items-center gap-2">
                        <Sparkles className="h-4 w-4 text-primary" /> Software Version &amp; Updates
                      </CardTitle>
                      <CardDescription className="text-xs">
                        Check for new releases published on GitHub and update in place with 1-click
                      </CardDescription>
                    </div>
                    <div className="flex items-center gap-2">
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={handleManualCheckUpdate}
                        disabled={isCheckingUpdate}
                        className="h-8 text-xs font-semibold gap-1.5"
                      >
                        <RefreshCw className={`h-3.5 w-3.5 ${isCheckingUpdate ? 'animate-spin' : ''}`} />
                        <span>{isCheckingUpdate ? 'Checking GitHub...' : 'Check for Updates'}</span>
                      </Button>
                      {versionInfo?.update_available && (
                        <Button
                          size="sm"
                          variant="default"
                          onClick={() => setShowUpdateModal(true)}
                          className="h-8 text-xs font-semibold gap-1.5 bg-primary shadow-xs"
                        >
                          <ArrowUpCircle className="h-3.5 w-3.5" />
                          <span>Update Now</span>
                        </Button>
                      )}
                    </div>
                  </div>
                </CardHeader>
                <CardContent className="space-y-3 text-xs">
                  <div className="flex items-center justify-between border-b pb-3">
                    <div>
                      <p className="font-semibold text-foreground">Installed Version</p>
                      <p className="text-muted-foreground text-[11px]">Locally running binary</p>
                    </div>
                    <Badge variant="outline" className="font-mono text-xs px-2 py-0.5">
                      {versionInfo?.current_version || 'v1.2.0'}
                    </Badge>
                  </div>
                  <div className="flex items-center justify-between border-b pb-3">
                    <div>
                      <p className="font-semibold text-foreground">Automatic Update Checks</p>
                      <p className="text-muted-foreground text-[11px]">
                        Check GitHub Releases in the background on launch (cached for 4 hours). Disable to keep LocalFinance completely offline.
                      </p>
                    </div>
                    <Switch
                      checked={autoCheckEnabled}
                      onCheckedChange={setAutoCheckEnabled}
                    />
                  </div>
                  <div className="flex items-center justify-between border-b pb-3">
                    <div>
                      <p className="font-semibold text-foreground">Release Status</p>
                      <p className="text-muted-foreground text-[11px]">
                        {versionInfo?.update_available
                          ? `New version ${versionInfo.latest_version} is available!`
                          : updateCheckError || versionInfo?.auto_update_error
                          ? `Check failed: ${updateCheckError || versionInfo?.auto_update_error}`
                          : !autoCheckEnabled && !versionInfo?.checked_at
                          ? 'Automatic checks disabled (offline mode). You can still check manually at any time.'
                          : versionInfo?.checked_at
                          ? `Up to date (checked at ${new Date(versionInfo.checked_at).toLocaleTimeString()})`
                          : 'Checking GitHub Releases in background...'}
                      </p>
                    </div>
                    {versionInfo?.update_available ? (
                      <Badge variant="default" className="bg-primary text-primary-foreground font-semibold text-[11px]">
                        {versionInfo.latest_version} Available
                      </Badge>
                    ) : updateCheckError || versionInfo?.auto_update_error ? (
                      <Badge variant="destructive" className="text-[11px]">
                        Check Failed
                      </Badge>
                    ) : !autoCheckEnabled && !versionInfo?.checked_at ? (
                      <Badge variant="secondary" className="text-[11px]">
                        Offline Only
                      </Badge>
                    ) : versionInfo?.checked_at ? (
                      <Badge variant="outline" className="border-emerald-500/30 text-emerald-600 dark:text-emerald-400 flex items-center gap-1 text-[11px]">
                        <CheckCircle2 className="h-3 w-3" /> Up to Date
                      </Badge>
                    ) : (
                      <Badge variant="secondary" className="text-[11px]">
                        Checking...
                      </Badge>
                    )}
                  </div>
                  <div className="flex items-center justify-between pt-1">
                    <div>
                      <p className="font-semibold text-foreground">GitHub Repository</p>
                      <p className="text-muted-foreground text-[11px]">Official releases and open-source changelog</p>
                    </div>
                    <a
                      href={versionInfo?.release_url || 'https://github.com/usmslm102/local-finance/releases'}
                      target="_blank"
                      rel="noreferrer"
                      className="inline-flex items-center gap-1 text-primary hover:underline font-medium text-xs"
                    >
                      View on GitHub <ExternalLink className="h-3 w-3" />
                    </a>
                  </div>
                </CardContent>
              </Card>

              {/* Appearance / Theme Card */}
              <Card className="border-border/80 bg-card shadow-xs">
                <CardHeader>
                  <CardTitle className="text-base font-semibold">Appearance & Theme</CardTitle>
                  <CardDescription className="text-xs">
                    Choose your preferred interface theme
                  </CardDescription>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div className="grid grid-cols-3 gap-3">
                    <button
                      onClick={() => setTheme('light')}
                      className={`flex flex-col items-center justify-center gap-2 rounded-xl border p-4 text-center transition-all ${
                        theme === 'light'
                          ? 'border-primary bg-primary/5 text-primary ring-1 ring-primary'
                          : 'border-border/70 bg-muted/20 text-muted-foreground hover:border-border hover:bg-muted/40'
                      }`}
                    >
                      <Sun className="h-5 w-5" />
                      <span className="text-xs font-semibold">Light</span>
                    </button>
                    <button
                      onClick={() => setTheme('dark')}
                      className={`flex flex-col items-center justify-center gap-2 rounded-xl border p-4 text-center transition-all ${
                        theme === 'dark'
                          ? 'border-primary bg-primary/5 text-primary ring-1 ring-primary'
                          : 'border-border/70 bg-muted/20 text-muted-foreground hover:border-border hover:bg-muted/40'
                      }`}
                    >
                      <Moon className="h-5 w-5" />
                      <span className="text-xs font-semibold">Dark</span>
                    </button>
                    <button
                      onClick={() => setTheme('system')}
                      className={`flex flex-col items-center justify-center gap-2 rounded-xl border p-4 text-center transition-all ${
                        theme === 'system'
                          ? 'border-primary bg-primary/5 text-primary ring-1 ring-primary'
                          : 'border-border/70 bg-muted/20 text-muted-foreground hover:border-border hover:bg-muted/40'
                      }`}
                    >
                      <Laptop className="h-5 w-5" />
                      <span className="text-xs font-semibold">System</span>
                    </button>
                  </div>
                </CardContent>
              </Card>

              {/* Audio & Tactile Sound Effects Card */}
              <Card className="border-border/80 bg-card shadow-xs">
                <CardHeader className="pb-3">
                  <div className="flex items-center justify-between flex-wrap gap-2">
                    <div>
                      <CardTitle className="text-base font-semibold flex items-center gap-2">
                        {soundEnabled ? (
                          <Volume2 className="h-4 w-4 text-primary" />
                        ) : (
                          <VolumeX className="h-4 w-4 text-muted-foreground" />
                        )}
                        Audio &amp; Tactile Sound Effects
                      </CardTitle>
                      <CardDescription className="text-xs">
                        Pure offline browser synthesizer cues for milestone interactions, copy actions, and discreet mode toggles
                      </CardDescription>
                    </div>
                    <Badge variant="outline" className="text-[10px] font-mono border-primary/30 text-primary">
                      100% Local Synthesizer
                    </Badge>
                  </div>
                </CardHeader>
                <CardContent className="space-y-4">
                  {/* Enable / Disable Toggle Row */}
                  <div className="flex items-center justify-between border-b pb-3 text-xs">
                    <div>
                      <p className="font-semibold text-foreground">Sound Effects</p>
                      <p className="text-muted-foreground text-[11px]">
                        Play subtle harmonic chimes and tactile click cues
                      </p>
                    </div>
                    <Button
                      size="sm"
                      variant={soundEnabled ? 'default' : 'outline'}
                      onClick={() => {
                        const next = !soundEnabled
                        setSoundEnabled(next)
                        setSoundEnabledState(next)
                        if (next) playSuccessChime()
                      }}
                      className="h-7 text-xs font-semibold px-3 gap-1.5"
                    >
                      {soundEnabled ? (
                        <>
                          <Volume2 className="h-3.5 w-3.5" />
                          <span>Enabled</span>
                        </>
                      ) : (
                        <>
                          <VolumeX className="h-3.5 w-3.5" />
                          <span>Muted</span>
                        </>
                      )}
                    </Button>
                  </div>

                  {/* Volume Slider Row */}
                  <div className="space-y-1.5 border-b pb-3 text-xs">
                    <div className="flex items-center justify-between">
                      <span className="font-semibold text-foreground">Output Volume</span>
                      <span className="font-mono text-muted-foreground font-semibold">{soundVol}%</span>
                    </div>
                    <input
                      type="range"
                      min="5"
                      max="100"
                      step="5"
                      value={soundVol}
                      disabled={!soundEnabled}
                      onChange={(e) => {
                        const val = parseInt(e.target.value, 10)
                        setSoundVolState(val)
                        setSoundVolume(val / 100)
                        playSoftClick(600)
                      }}
                      className="w-full accent-primary h-1.5 bg-muted rounded-lg cursor-pointer disabled:opacity-40"
                    />
                  </div>

                  {/* Interactive Test Panel */}
                  <div className="space-y-2 pt-1">
                    <p className="text-[11px] font-bold uppercase tracking-wider text-muted-foreground">
                      Audition Sound Cues (Click to Test)
                    </p>
                    <div className="grid grid-cols-2 sm:grid-cols-4 gap-2">
                      <Button
                        size="sm"
                        variant="secondary"
                        onClick={() => playSuccessChime()}
                        disabled={!soundEnabled}
                        className="h-8 text-xs font-semibold gap-1.5"
                      >
                        <span>🔔 Success Chime</span>
                      </Button>
                      <Button
                        size="sm"
                        variant="secondary"
                        onClick={() => playPrivacyToggleSound(true)}
                        disabled={!soundEnabled}
                        className="h-8 text-xs font-semibold gap-1.5"
                      >
                        <span>🛡️ Discreet Mask</span>
                      </Button>
                      <Button
                        size="sm"
                        variant="secondary"
                        onClick={() => playPrivacyToggleSound(false)}
                        disabled={!soundEnabled}
                        className="h-8 text-xs font-semibold gap-1.5"
                      >
                        <span>👁️ Public Reveal</span>
                      </Button>
                      <Button
                        size="sm"
                        variant="secondary"
                        onClick={() => playSoftClick(650)}
                        disabled={!soundEnabled}
                        className="h-8 text-xs font-semibold gap-1.5"
                      >
                        <span>🔘 Interface Tap</span>
                      </Button>
                    </div>
                    <p className="text-[10px] text-muted-foreground pt-1">
                      💡 <strong>Note:</strong> Browsers (Chrome/Safari) require at least one user click on the webpage before audio output is permitted by their autoplay security policies.
                    </p>
                  </div>
                </CardContent>
              </Card>

              {/* Regional & Financial Formatting */}
              <Card className="border-border/80 bg-card shadow-xs">
                <CardHeader>
                  <CardTitle className="text-base font-semibold">Regional & Indian Ecosystem</CardTitle>
                  <CardDescription className="text-xs">
                    Formatting configurations tailored for Indian Banking & Cards
                  </CardDescription>
                </CardHeader>
                <CardContent className="space-y-3 text-xs">
                  <div className="flex items-center justify-between border-b pb-3">
                    <div>
                      <p className="font-semibold text-foreground">Currency Representation</p>
                      <p className="text-muted-foreground text-[11px]">Indian Rupee (INR / ₹) with Lakhs & Crores grouping</p>
                    </div>
                    <Badge variant="outline" className="font-mono">
                      ₹ Indian Rupee
                    </Badge>
                  </div>
                  <div className="flex items-center justify-between border-b pb-3">
                    <div>
                      <p className="font-semibold text-foreground">Payment Modes</p>
                      <p className="text-muted-foreground text-[11px]">Native sniffing for UPI VPAs, IMPS, NEFT, RTGS & POS terminals</p>
                    </div>
                    <Badge variant="secondary" className="font-mono text-[10px]">
                      Enabled (12 Modes)
                    </Badge>
                  </div>
                  <div className="flex items-center justify-between pt-1">
                    <div>
                      <p className="font-semibold text-foreground">Date Normalization</p>
                      <p className="text-muted-foreground text-[11px]">Formats parsed from DD/MM/YYYY, DD-MMM-YYYY, and Month DD, YYYY</p>
                    </div>
                    <Badge variant="outline" className="font-mono text-[10px]">
                      ISO 8601 Internal
                    </Badge>
                  </div>
                </CardContent>
              </Card>

              {/* Privacy Guarantee */}
              <Card className="border-border/80 bg-card shadow-xs">
                <CardHeader>
                  <CardTitle className="text-base font-semibold flex items-center gap-2">
                    <ShieldCheck className="h-4 w-4 text-emerald-400" /> Privacy & Local-Only Guarantee
                  </CardTitle>
                </CardHeader>
                <CardContent className="text-xs text-muted-foreground space-y-2">
                  <p>
                    <strong className="text-foreground">Zero Cloud Telemetry:</strong> All statement processing, parsing, OCR decryption, and categorization execute 100% locally on your device.
                  </p>
                  <p>
                    <strong className="text-foreground">Offline SQLite:</strong> Your financial data resides in a local SQLite file with no third-party account aggregators or SMS scrapers.
                  </p>
                </CardContent>
              </Card>
            </div>
          )}

          {/* TAB: SECURITY & APP LOCK */}
          {activeTab === 'security' && (
            <div className="space-y-6">
              {/* App Lock & Password Protection Card */}
              <Card className="border-border/80 bg-card shadow-xs">
                <CardHeader className="pb-3">
                  <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
                    <div>
                      <CardTitle className="text-base font-semibold flex items-center gap-2">
                        <Lock className="h-4 w-4 text-primary" /> Local Authentication &amp; Master Lock
                      </CardTitle>
                      <CardDescription className="text-xs">
                        Require a master password or PIN to unlock the application and view your offline financial ledger
                      </CardDescription>
                    </div>
                    <div>
                      {authStatus?.auth_enabled ? (
                        <Badge variant="outline" className="text-xs font-mono border-emerald-500/30 bg-emerald-500/10 text-emerald-400 gap-1.5 py-1 px-2.5">
                          <CheckCircle2 className="h-3.5 w-3.5" /> App Lock Active
                        </Badge>
                      ) : (
                        <Badge variant="secondary" className="text-xs font-mono gap-1.5 py-1 px-2.5">
                          <Unlock className="h-3.5 w-3.5 text-muted-foreground" /> Lock Disabled
                        </Badge>
                      )}
                    </div>
                  </div>
                </CardHeader>
                <CardContent className="space-y-4 text-xs">
                  {!authStatus?.auth_enabled ? (
                    <div className="rounded-lg border border-dashed border-border/80 bg-muted/20 p-5 text-center flex flex-col items-center justify-center gap-3">
                      <div className="flex h-12 w-12 items-center justify-center rounded-2xl bg-primary/10 text-primary border border-primary/20">
                        <Lock className="h-6 w-6" />
                      </div>
                      <div className="max-w-md space-y-1">
                        <p className="text-sm font-semibold text-foreground">
                          Protect Your Financial Privacy
                        </p>
                        <p className="text-xs text-muted-foreground">
                          Anyone with access to this browser or port 8080 on this computer can currently view your bank accounts, transactions, and balances.
                          Enable a master PIN or password to lock down access.
                        </p>
                      </div>
                      <Button
                        size="sm"
                        onClick={() => {
                          setSecurityActionError(null)
                          setSetupPassword('')
                          setSetupConfirmPassword('')
                          setShowSetupAuthModal(true)
                        }}
                        className="font-semibold gap-1.5 h-8.5 mt-1"
                      >
                        <KeyRound className="h-3.5 w-3.5" /> Set Master Password / PIN
                      </Button>
                    </div>
                  ) : (
                    <div className="space-y-4">
                      {/* Active Status Row */}
                      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 border-b pb-4">
                        <div>
                          <p className="font-semibold text-foreground flex items-center gap-1.5">
                            <ShieldCheck className="h-4 w-4 text-emerald-400" /> Master Password Protection
                          </p>
                          <p className="text-[11px] text-muted-foreground mt-0.5">
                            Ledger queries and sensitive financial APIs require an active authenticated session.
                          </p>
                        </div>
                        <div className="flex items-center gap-2">
                          <Button
                            size="sm"
                            variant="secondary"
                            onClick={() => lockApp()}
                            className="h-8 text-xs font-semibold gap-1.5"
                          >
                            <Lock className="h-3.5 w-3.5" /> Lock App Now
                          </Button>
                          <Button
                            size="sm"
                            variant="outline"
                            onClick={() => {
                              setSecurityActionError(null)
                              setChangeCurrentPassword('')
                              setChangeNewPassword('')
                              setChangeConfirmPassword('')
                              setShowChangePasswordModal(true)
                            }}
                            className="h-8 text-xs font-semibold gap-1.5"
                          >
                            <Edit2 className="h-3.5 w-3.5" /> Change Password
                          </Button>
                        </div>
                      </div>

                      {/* Auto-Lock Inactivity Timeout */}
                      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 border-b pb-4">
                        <div>
                          <p className="font-semibold text-foreground">Inactivity Auto-Lock Timeout</p>
                          <p className="text-[11px] text-muted-foreground mt-0.5">
                            Automatically locks the application after period of inactivity in the browser
                          </p>
                        </div>
                        <div className="flex items-center gap-2">
                          <Select
                            value={String(authStatus?.auto_lock_minutes || 60)}
                            onValueChange={(val) => val && updateTimeoutMutation.mutate(parseInt(val, 10))}
                            disabled={updateTimeoutMutation.isPending}
                          >
                            <SelectTrigger className="text-xs h-8 w-[160px]">
                              <SelectValue />
                            </SelectTrigger>
                            <SelectContent>
                              <SelectItem value="15">15 Minutes</SelectItem>
                              <SelectItem value="30">30 Minutes</SelectItem>
                              <SelectItem value="60">1 Hour</SelectItem>
                              <SelectItem value="240">4 Hours</SelectItem>
                            </SelectContent>
                          </Select>
                        </div>
                      </div>

                      {/* Disable Security */}
                      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 pt-1">
                        <div>
                          <p className="font-semibold text-destructive">Disable Master Password</p>
                          <p className="text-[11px] text-muted-foreground mt-0.5">
                            Turn off authentication. The application will be unlocked for anyone on this device.
                          </p>
                        </div>
                        <Button
                          size="sm"
                          variant="outline"
                          onClick={() => {
                            setSecurityActionError(null)
                            setDisablePassword('')
                            setShowDisableAuthModal(true)
                          }}
                          className="h-8 text-xs font-semibold text-destructive hover:text-destructive border-destructive/30 hover:bg-destructive/10"
                        >
                          <Unlock className="h-3.5 w-3.5 mr-1.5" /> Disable App Lock
                        </Button>
                      </div>
                    </div>
                  )}
                </CardContent>
              </Card>

              {/* SQLite Database Security & File Permissions Card */}
              <Card className="border-border/80 bg-card shadow-xs">
                <CardHeader className="pb-3">
                  <div className="flex items-center justify-between">
                    <div>
                      <CardTitle className="text-base font-semibold flex items-center gap-2">
                        <Database className="h-4 w-4 text-primary" /> SQLite Database Disk Protection
                      </CardTitle>
                      <CardDescription className="text-xs">
                        How your local SQLite database file is safeguarded on disk
                      </CardDescription>
                    </div>
                    <Badge variant="outline" className="font-mono text-xs border-emerald-500/30 text-emerald-400">
                      0600 File Mode
                    </Badge>
                  </div>
                </CardHeader>
                <CardContent className="space-y-4 text-xs">
                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                    <div className="rounded-lg border bg-muted/20 p-3 space-y-1">
                      <span className="text-muted-foreground text-[11px] font-semibold">Unix File Mode Permission</span>
                      <p className="font-mono font-bold text-foreground text-sm">
                        {authStatus?.file_permissions || '-rw------- (0600)'}
                      </p>
                      <p className="text-[11px] text-muted-foreground">
                        Restricted to owner only. Other local OS user accounts on this machine cannot read or copy this database file.
                      </p>
                    </div>

                    <div className="rounded-lg border bg-muted/20 p-3 space-y-1">
                      <span className="text-muted-foreground text-[11px] font-semibold">Password Storage Security</span>
                      <p className="font-mono font-bold text-foreground text-sm">
                        bcrypt Multi-Round Salted Hash
                      </p>
                      <p className="text-[11px] text-muted-foreground">
                        Passwords are never stored in plaintext. Hashed with adaptive multi-round bcrypt salt and verified locally.
                      </p>
                    </div>
                  </div>

                  <div className="rounded-lg border border-border/70 bg-muted/30 p-3.5 space-y-2 text-muted-foreground text-[11px] leading-relaxed">
                    <p className="font-semibold text-foreground text-xs flex items-center gap-1.5">
                      <ShieldCheck className="h-4 w-4 text-primary" /> Multi-Layer Offline Defense:
                    </p>
                    <ul className="list-disc list-inside space-y-1 pl-1">
                      <li>
                        <strong className="text-foreground">App Lock Layer:</strong> Prevents unauthorized viewing of dashboards, accounts, and transactions on your browser or local network.
                      </li>
                      <li>
                        <strong className="text-foreground">Filesystem Permissions (0600):</strong> Enforced at database initialization so standard system users without root cannot read the file.
                      </li>
                      <li>
                        <strong className="text-foreground">Hardware At-Rest Encryption:</strong> Because SQLite lives entirely on your disk, combining LocalFinance with macOS FileVault, Windows BitLocker, or Linux LUKS provides complete encryption at rest even if the physical drive is extracted.
                      </li>
                    </ul>
                  </div>
                </CardContent>
              </Card>
            </div>
          )}

          {/* TAB 2: CATEGORIES & AUTO-TAGGING RULES */}
          {activeTab === 'rules' && (
            <div className="space-y-6">
              {/* Categories */}
              <Card className="border-border/80 bg-card shadow-xs">
                <CardHeader className="pb-3">
                  <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
                    <div>
                      <CardTitle className="text-base font-semibold flex items-center gap-2">
                        <Tags className="h-4 w-4 text-primary" /> Spending Taxonomy & Categories
                      </CardTitle>
                      <CardDescription className="text-xs">
                        System and custom categories for budgeting, ledger analytics, and spending breakdown
                      </CardDescription>
                    </div>
                    <div className="flex items-center gap-2">
                      <Button
                        size="sm"
                        onClick={handleOpenAddCategory}
                        className="text-xs font-semibold gap-1.5 h-8"
                      >
                        <Plus className="h-3.5 w-3.5" /> Add Category
                      </Button>
                      <Badge variant="secondary" className="text-xs font-mono">
                        {categories?.length || 0} Categories
                      </Badge>
                    </div>
                  </div>
                </CardHeader>
                <CardContent>
                  <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 gap-3">
                    {categories?.map((cat) => (
                      <div
                        key={cat.id}
                        className="group flex items-center justify-between rounded-lg border border-border/70 bg-muted/30 p-3 transition-colors hover:border-border"
                      >
                        <div className="flex items-center gap-2.5 truncate min-w-0">
                          <div
                            className="h-3 w-3 rounded-full flex-shrink-0"
                            style={{ backgroundColor: cat.color_hex }}
                          />
                          <div className="truncate min-w-0">
                            <p className="text-xs font-semibold text-foreground truncate">{cat.name}</p>
                            <p className="text-[10px] text-muted-foreground font-medium">
                              {cat.is_system ? 'System Default' : 'Custom Category'}
                            </p>
                          </div>
                        </div>

                        {!cat.is_system && (
                          <button
                            onClick={() => setShowDeleteCategoryModal(cat)}
                            className="opacity-0 group-hover:opacity-100 p-1 text-muted-foreground hover:text-destructive transition-opacity"
                            title="Delete category"
                          >
                            <Trash2 className="h-3.5 w-3.5" />
                          </button>
                        )}
                      </div>
                    ))}
                  </div>
                </CardContent>
              </Card>

              {/* Rules Table */}
              <Card className="border-border/80 bg-card shadow-xs">
                <CardHeader className="pb-3 border-b">
                  <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
                    <div>
                      <CardTitle className="text-base font-semibold flex items-center gap-2">
                        <Sliders className="h-4 w-4 text-primary" /> Auto-Categorization & Match Rules
                      </CardTitle>
                      <CardDescription className="text-xs">
                        Prioritized rules that map UPI VPAs, POS merchant keywords, and narrations to categories
                      </CardDescription>
                    </div>

                    <div className="flex items-center gap-2">
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => reapplyRulesMutation.mutate()}
                        disabled={reapplyRulesMutation.isPending}
                        className="text-xs font-semibold gap-1.5 h-8"
                        title="Re-run all rules against existing uncategorized ledger transactions"
                      >
                        <RefreshCw className={`h-3.5 w-3.5 ${reapplyRulesMutation.isPending ? 'animate-spin' : ''}`} />
                        {reapplyRulesMutation.isPending ? 'Applying...' : 'Re-Apply to Ledger'}
                      </Button>

                      <Button
                        size="sm"
                        onClick={handleOpenAddRule}
                        className="text-xs font-semibold gap-1.5 h-8"
                      >
                        <Plus className="h-3.5 w-3.5" /> Add Rule
                      </Button>

                      <Badge variant="secondary" className="text-xs font-mono">
                        {rules?.length || 0} Rules
                      </Badge>
                    </div>
                  </div>
                </CardHeader>
                <CardContent className="p-0">
                  <div className="overflow-x-auto">
                    <Table>
                      <TableHeader>
                        <TableRow className="hover:bg-transparent">
                          <TableHead className="text-xs font-semibold uppercase">Priority</TableHead>
                          <TableHead className="text-xs font-semibold uppercase">Match Field</TableHead>
                          <TableHead className="text-xs font-semibold uppercase">Type</TableHead>
                          <TableHead className="text-xs font-semibold uppercase">Pattern / Keyword</TableHead>
                          <TableHead className="text-xs font-semibold uppercase">Assigned Category</TableHead>
                          <TableHead className="text-xs font-semibold uppercase">Status</TableHead>
                          <TableHead className="text-xs font-semibold uppercase text-right">Actions</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {rules?.map((r) => (
                          <TableRow key={r.id} className="hover:bg-muted/40 text-xs">
                            <TableCell className="font-mono font-bold text-muted-foreground">{r.priority}</TableCell>
                            <TableCell className="font-mono text-foreground">{r.match_field}</TableCell>
                            <TableCell>
                              <div className="flex flex-col gap-1 items-start">
                                <Badge variant="outline" className="font-mono text-[10px] uppercase">
                                  {r.match_type}
                                </Badge>
                                {r.tx_type && r.tx_type !== 'ALL' && (
                                  <Badge
                                    variant="secondary"
                                    className={`text-[9px] font-semibold px-1.5 py-0 ${
                                      r.tx_type === 'CREDIT'
                                        ? 'text-emerald-400 bg-emerald-500/10 border-emerald-500/20'
                                        : 'text-rose-400 bg-rose-500/10 border-rose-500/20'
                                    }`}
                                  >
                                    {r.tx_type === 'CREDIT' ? 'Credits Only' : 'Debits Only'}
                                  </Badge>
                                )}
                              </div>
                            </TableCell>
                            <TableCell className="font-mono text-xs max-w-[240px]">
                              <div className="flex flex-col gap-0.5">
                                <span className="font-medium text-emerald-400 truncate">
                                  {r.match_pattern}
                                </span>
                                {r.exclude_pattern && (
                                  <span
                                    className="text-[10px] text-amber-400/90 truncate flex items-center gap-1"
                                    title={`Except: ${r.exclude_pattern}`}
                                  >
                                    <span className="font-semibold text-[9px] uppercase px-1 py-0 rounded-xs bg-amber-500/15 border border-amber-500/20 text-amber-400">
                                      Except
                                    </span>
                                    <span className="truncate">{r.exclude_pattern}</span>
                                  </span>
                                )}
                              </div>
                            </TableCell>
                            <TableCell className="font-semibold text-foreground">
                              {r.target_category || r.target_category_id}
                            </TableCell>
                            <TableCell>
                              <Badge
                                className={`text-[10px] font-semibold ${
                                  r.is_active
                                    ? 'bg-emerald-500/10 text-emerald-400 border-emerald-500/20'
                                    : 'bg-muted text-muted-foreground'
                                }`}
                              >
                                {r.is_active ? 'Active' : 'Disabled'}
                              </Badge>
                            </TableCell>
                            <TableCell className="text-right">
                              <div className="flex items-center justify-end gap-1">
                                <Button
                                  variant="ghost"
                                  size="sm"
                                  onClick={() => handleOpenEditRule(r)}
                                  className="h-7 w-7 p-0 text-muted-foreground hover:text-foreground"
                                  title="Edit Rule"
                                >
                                  <Edit2 className="h-3.5 w-3.5" />
                                </Button>
                                <Button
                                  variant="ghost"
                                  size="sm"
                                  onClick={() => setShowDeleteRuleModal(r)}
                                  className="h-7 w-7 p-0 text-muted-foreground hover:text-destructive"
                                  title="Delete Rule"
                                >
                                  <Trash2 className="h-3.5 w-3.5" />
                                </Button>
                              </div>
                            </TableCell>
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  </div>
                </CardContent>
              </Card>
            </div>
          )}

          {/* TAB 3: BANK PARSER PLUGINS */}
          {activeTab === 'parsers' && (
            <div className="space-y-6">
              <Card className="border-border/80 bg-card shadow-xs">
                <CardHeader className="pb-3">
                  <div className="flex items-center justify-between">
                    <div>
                      <CardTitle className="text-base font-semibold flex items-center gap-2">
                        <Cpu className="h-4 w-4 text-primary" /> Registered Statement Parsers
                      </CardTitle>
                      <CardDescription className="text-xs">
                        Modular format extractors equipped with automatic format sniffing and confidence scoring
                      </CardDescription>
                    </div>
                    <Badge variant="secondary" className="text-xs font-mono">
                      {parsers?.length || 0} Registered
                    </Badge>
                  </div>
                </CardHeader>
                <CardContent className="pt-0">
                  <div className="space-y-3">
                    {parsers?.map((p) => (
                      <div
                        key={p.id}
                        className="rounded-lg border border-border/70 bg-muted/30 p-4 transition-colors hover:border-border flex flex-col sm:flex-row sm:items-center justify-between gap-3"
                      >
                        <div className="flex items-start gap-3">
                          <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-background border text-primary">
                            {p.supported_types.some((t) => t.includes('pdf')) ? (
                              <FileCode2 className="h-4 w-4" />
                            ) : (
                              <FileSpreadsheet className="h-4 w-4" />
                            )}
                          </div>
                          <div>
                            <p className="text-xs font-semibold text-foreground">{p.name}</p>
                            <p className="text-[11px] font-mono text-muted-foreground mt-0.5">
                              ID: {p.id}
                            </p>
                          </div>
                        </div>

                        <div className="flex items-center gap-2 flex-wrap self-end sm:self-center">
                          {p.supported_types.map((type) => (
                            <Badge key={type} variant="outline" className="text-[10px] font-mono uppercase">
                              {type.replace('TYPE_', '')}
                            </Badge>
                          ))}
                          <Badge className="bg-emerald-500/10 text-emerald-400 border-emerald-500/20 text-[10px]">
                            Ready
                          </Badge>
                        </div>
                      </div>
                    ))}
                  </div>
                </CardContent>
              </Card>

              {/* Password Decryption Guide */}
              <Card className="border-border/80 bg-card shadow-xs">
                <CardHeader>
                  <CardTitle className="text-base font-semibold flex items-center gap-2">
                    <Lock className="h-4 w-4 text-amber-400" /> Encrypted Statement Decryption
                  </CardTitle>
                </CardHeader>
                <CardContent className="text-xs text-muted-foreground space-y-2">
                  <p>
                    <strong className="text-foreground">Password decrypting:</strong> Handled entirely locally in memory using standard PDF AES-128/256 & RC4 algorithms. Statement passwords are never logged, cached, or written to disk.
                  </p>
                  <div className="grid grid-cols-1 sm:grid-cols-3 gap-2 mt-3 font-mono text-[11px]">
                    <div className="border rounded-md p-2.5 bg-muted/40">
                      <span className="text-foreground font-semibold">HDFC Bank:</span>
                      <p className="text-muted-foreground mt-0.5">8-digit Customer ID</p>
                    </div>
                    <div className="border rounded-md p-2.5 bg-muted/40">
                      <span className="text-foreground font-semibold">ICICI Bank:</span>
                      <p className="text-muted-foreground mt-0.5">DOB (DDMM) + Last 4 of Mobile</p>
                    </div>
                    <div className="border rounded-md p-2.5 bg-muted/40">
                      <span className="text-foreground font-semibold">Axis Bank:</span>
                      <p className="text-muted-foreground mt-0.5">Name (4 chars) + DOB (DDMM)</p>
                    </div>
                  </div>
                </CardContent>
              </Card>
            </div>
          )}

          {/* TAB 4: DATABASE & STORAGE MANAGEMENT */}
          {activeTab === 'database' && (
            <div className="space-y-6">
              {/* Active Database Location Card */}
              <Card className="border-border/80 bg-card shadow-xs">
                <CardHeader className="pb-3">
                  <div className="flex items-center justify-between">
                    <CardTitle className="text-base font-semibold flex items-center gap-2">
                      <FolderOpen className="h-4 w-4 text-primary" /> Active Database Location
                    </CardTitle>
                    <Badge variant="outline" className="font-mono text-xs text-emerald-400 border-emerald-500/20 bg-emerald-500/10">
                      WAL Mode Active
                    </Badge>
                  </div>
                  <CardDescription className="text-xs">
                    Absolute file system path where your encrypted local ledger lives
                  </CardDescription>
                </CardHeader>
                <CardContent className="space-y-3">
                  <div className="flex items-center justify-between gap-3 rounded-lg border bg-muted/40 p-3">
                    <div className="min-w-0 flex-1">
                      <p className="text-[11px] text-muted-foreground font-medium uppercase tracking-wider">File Path</p>
                      <p className="text-xs font-mono font-semibold text-foreground truncate mt-0.5" title={dbInfo?.path}>
                        {dbInfo?.path || 'Loading path...'}
                      </p>
                    </div>
                    {dbInfo?.path && (
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => handleCopyPath(dbInfo.path)}
                        className="h-8 shrink-0 text-xs gap-1.5"
                      >
                        {copiedPath ? (
                          <>
                            <Check className="h-3.5 w-3.5 text-emerald-400" />
                            <span className="text-emerald-400 font-semibold">Copied</span>
                          </>
                        ) : (
                          <>
                            <Copy className="h-3.5 w-3.5" />
                            <span>Copy Path</span>
                          </>
                        )}
                      </Button>
                    )}
                  </div>

                  <div className="grid grid-cols-2 sm:grid-cols-3 gap-3 text-xs pt-1">
                    <div className="rounded-lg border bg-muted/20 p-2.5">
                      <span className="text-muted-foreground text-[11px]">Database Size</span>
                      <p className="font-mono font-bold text-foreground mt-0.5">
                        {dbInfo?.file_size ? formatBytes(dbInfo.file_size) : '0 B'}
                      </p>
                    </div>
                    <div className="rounded-lg border bg-muted/20 p-2.5">
                      <span className="text-muted-foreground text-[11px]">Last Modified</span>
                      <p className="font-mono font-bold text-foreground mt-0.5">
                        {dbInfo?.last_modified ? formatDate(dbInfo.last_modified) : 'Just now'}
                      </p>
                    </div>
                    <div className="rounded-lg border bg-muted/20 p-2.5 col-span-2 sm:col-span-1">
                      <span className="text-muted-foreground text-[11px]">Driver Engine</span>
                      <p className="font-mono font-bold text-foreground mt-0.5">
                        Pure Go SQLite
                      </p>
                    </div>
                  </div>
                </CardContent>
              </Card>

              {/* Database Backup & Export Center */}
              <Card className="border-border/80 bg-card shadow-xs">
                <CardHeader className="pb-3">
                  <CardTitle className="text-base font-semibold flex items-center gap-2">
                    <HardDriveDownload className="h-4 w-4 text-primary" /> Backup & Data Export
                  </CardTitle>
                  <CardDescription className="text-xs">
                    Create standalone SQLite snapshots or export data as CSV/JSON
                  </CardDescription>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
                    {/* Direct Download SQLite Database Backup */}
                    <a href="/api/database/backup" download className="block">
                      <div className="rounded-lg border border-border/80 bg-muted/30 p-4 transition-all hover:border-primary hover:bg-muted/50 h-full flex flex-col justify-between">
                        <div>
                          <div className="flex items-center gap-2 text-primary font-semibold text-xs">
                            <Database className="h-4 w-4" /> SQLite Backup (.db)
                          </div>
                          <p className="text-[11px] text-muted-foreground mt-1">
                            Point-in-time standalone snapshot with WAL flush.
                          </p>
                        </div>
                        <Button size="sm" variant="outline" className="w-full mt-3 text-xs gap-1.5 h-8">
                          <Download className="h-3.5 w-3.5" /> Download .db
                        </Button>
                      </div>
                    </a>

                    {/* Export Transactions CSV */}
                    <a href="/api/database/export/csv" download className="block">
                      <div className="rounded-lg border border-border/80 bg-muted/30 p-4 transition-all hover:border-primary hover:bg-muted/50 h-full flex flex-col justify-between">
                        <div>
                          <div className="flex items-center gap-2 text-emerald-400 font-semibold text-xs">
                            <FileSpreadsheet className="h-4 w-4" /> Export CSV Ledger
                          </div>
                          <p className="text-[11px] text-muted-foreground mt-1">
                            Clean tabular export of all transactions and accounts.
                          </p>
                        </div>
                        <Button size="sm" variant="outline" className="w-full mt-3 text-xs gap-1.5 h-8">
                          <Download className="h-3.5 w-3.5" /> Download .csv
                        </Button>
                      </div>
                    </a>

                    {/* Export Full JSON */}
                    <a href="/api/database/export/json" download className="block">
                      <div className="rounded-lg border border-border/80 bg-muted/30 p-4 transition-all hover:border-primary hover:bg-muted/50 h-full flex flex-col justify-between">
                        <div>
                          <div className="flex items-center gap-2 text-sky-400 font-semibold text-xs">
                            <FileCode2 className="h-4 w-4" /> Full JSON Dump
                          </div>
                          <p className="text-[11px] text-muted-foreground mt-1">
                            Complete dump of accounts, rules, bills & transactions.
                          </p>
                        </div>
                        <Button size="sm" variant="outline" className="w-full mt-3 text-xs gap-1.5 h-8">
                          <Download className="h-3.5 w-3.5" /> Download .json
                        </Button>
                      </div>
                    </a>
                  </div>
                </CardContent>
              </Card>

              {/* Import / Restore Database Backup */}
              <Card className="border-border/80 bg-card shadow-xs">
                <CardHeader className="pb-3">
                  <CardTitle className="text-base font-semibold flex items-center gap-2">
                    <UploadCloud className="h-4 w-4 text-primary" /> Import & Restore Database
                  </CardTitle>
                  <CardDescription className="text-xs">
                    Restore your financial ledger from a previously saved <code className="font-mono text-foreground">.db</code> or <code className="font-mono text-foreground">.sqlite</code> backup file
                  </CardDescription>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div className="rounded-lg border border-dashed border-border/80 bg-muted/20 p-5 text-center flex flex-col items-center justify-center gap-3">
                    <div className="flex h-10 w-10 items-center justify-center rounded-full bg-primary/10 text-primary">
                      <Database className="h-5 w-5" />
                    </div>
                    <div>
                      <p className="text-xs font-semibold text-foreground">
                        Select a SQLite Backup File (.db / .sqlite)
                      </p>
                      <p className="text-[11px] text-muted-foreground mt-0.5">
                        Restoring replaces the current database. An automated backup (<code className="font-mono">.bak</code>) of the existing file will be saved.
                      </p>
                    </div>

                    <input
                      ref={restoreFileInputRef}
                      type="file"
                      accept=".db,.sqlite,.sqlite3"
                      className="hidden"
                      onChange={handleRestoreFileSelected}
                    />

                    <Button
                      size="sm"
                      onClick={() => restoreFileInputRef.current?.click()}
                      className="text-xs font-semibold gap-1.5 h-8"
                    >
                      <FolderOpen className="h-3.5 w-3.5" /> Browse Backup File
                    </Button>
                  </div>
                </CardContent>
              </Card>

              {/* SQLite DB Statistics */}
              <Card className="border-border/80 bg-card shadow-xs">
                <CardHeader className="pb-3">
                  <div className="flex items-center justify-between">
                    <div>
                      <CardTitle className="text-base font-semibold flex items-center gap-2">
                        <Database className="h-4 w-4 text-primary" /> SQLite Database Statistics
                      </CardTitle>
                      <CardDescription className="text-xs">
                        Current storage footprint and records managed in local database
                      </CardDescription>
                    </div>
                    <Badge variant="outline" className="font-mono text-xs">
                      WAL Mode Active
                    </Badge>
                  </div>
                </CardHeader>
                <CardContent>
                  <div className="grid grid-cols-2 sm:grid-cols-4 gap-3 text-xs">
                    <div className="rounded-lg bg-muted/40 border p-3">
                      <span className="text-muted-foreground font-medium">Transactions</span>
                      <p className="font-bold font-mono text-foreground text-lg mt-0.5">
                        {dbInfo?.total_transactions ?? analytics?.total_transactions ?? 0}
                      </p>
                    </div>
                    <div className="rounded-lg bg-muted/40 border p-3">
                      <span className="text-muted-foreground font-medium">Linked Accounts</span>
                      <p className="font-bold font-mono text-foreground text-lg mt-0.5">
                        {dbInfo?.total_accounts ?? accounts?.length ?? 0}
                      </p>
                    </div>
                    <div className="rounded-lg bg-muted/40 border p-3">
                      <span className="text-muted-foreground font-medium">Import Logs</span>
                      <p className="font-bold font-mono text-foreground text-lg mt-0.5">
                        {dbInfo?.total_statements ?? statementsHistory?.length ?? 0}
                      </p>
                    </div>
                    <div className="rounded-lg bg-muted/40 border p-3">
                      <span className="text-muted-foreground font-medium">Rule Matchers</span>
                      <p className="font-bold font-mono text-foreground text-lg mt-0.5">
                        {dbInfo?.total_rules ?? rules?.length ?? 0}
                      </p>
                    </div>
                  </div>
                </CardContent>
              </Card>

              {/* Danger Zone: Reset Database */}
              <Card className="border-destructive/40 bg-card shadow-xs">
                <CardHeader className="pb-3">
                  <CardTitle className="text-base font-semibold text-destructive flex items-center gap-2">
                    <Trash2 className="h-4 w-4" /> Danger Zone • Database Management
                  </CardTitle>
                  <CardDescription className="text-xs">
                    Actions that modify or reset existing transaction ledger data
                  </CardDescription>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div className="rounded-lg border border-destructive/30 bg-destructive/5 p-4 flex flex-col sm:flex-row sm:items-center justify-between gap-4">
                    <div>
                      <h5 className="text-xs font-semibold text-destructive">
                        Clear Transactions & Statement Logs
                      </h5>
                      <p className="text-[11px] text-muted-foreground mt-0.5">
                        Clears all imported transactions, accounts, and statement logs to restart testing. Categories and custom rules will be preserved.
                      </p>
                    </div>
                    <Button
                      variant="destructive"
                      size="sm"
                      onClick={() => setShowConfirmReset(true)}
                      className="shrink-0 text-xs font-semibold h-8"
                    >
                      <Trash2 className="h-3.5 w-3.5 mr-1.5" /> Reset Database
                    </Button>
                  </div>
                </CardContent>
              </Card>
            </div>
          )}
        </main>
      </div>

      {/* Confirmation Modal for Resetting Database */}
      <Dialog open={showConfirmReset} onOpenChange={setShowConfirmReset}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle className="text-base font-semibold text-destructive flex items-center gap-2">
              <Trash2 className="h-5 w-5" /> Reset Local Database
            </DialogTitle>
            <DialogDescription className="text-xs">
              Are you sure you want to clear all transactions, accounts, credit card bills, and statement logs?
              Your custom rules and categories will be safely preserved.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter className="gap-2 pt-2">
            <Button
              variant="outline"
              size="sm"
              onClick={() => setShowConfirmReset(false)}
            >
              Cancel
            </Button>
            <Button
              variant="destructive"
              size="sm"
              onClick={() => resetMutation.mutate()}
              disabled={resetMutation.isPending}
              className="gap-1.5"
            >
              {resetMutation.isPending ? 'Clearing...' : 'Confirm Reset'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Confirmation Modal for Restoring Database Backup */}
      <Dialog open={showConfirmRestore} onOpenChange={setShowConfirmRestore}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle className="text-base font-semibold text-primary flex items-center gap-2">
              <UploadCloud className="h-5 w-5" /> Restore Database Backup
            </DialogTitle>
            <DialogDescription className="text-xs">
              Are you sure you want to restore the database from{' '}
              <strong className="text-foreground">{selectedRestoreFile?.name}</strong>?
              Your current database will be archived as <code className="font-mono">.bak</code> before replacement.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter className="gap-2 pt-2">
            <Button
              variant="outline"
              size="sm"
              onClick={() => {
                setShowConfirmRestore(false)
                setSelectedRestoreFile(null)
              }}
            >
              Cancel
            </Button>
            <Button
              size="sm"
              onClick={() => {
                if (selectedRestoreFile) {
                  restoreMutation.mutate(selectedRestoreFile)
                }
              }}
              disabled={restoreMutation.isPending}
              className="gap-1.5 font-semibold"
            >
              {restoreMutation.isPending ? (
                <>
                  <RefreshCw className="h-3.5 w-3.5 animate-spin" />
                  Restoring...
                </>
              ) : (
                'Confirm & Restore'
              )}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Add / Edit Categorization Rule Dialog */}
      <Dialog
        open={showAddRuleModal || !!editingRule}
        onOpenChange={(open) => {
          if (!open) {
            setShowAddRuleModal(false)
            setEditingRule(null)
          }
        }}
      >
        <DialogContent className="sm:max-w-lg">
          <DialogHeader>
            <DialogTitle className="text-base font-semibold flex items-center gap-2">
              {editingRule ? <Edit2 className="h-4 w-4 text-primary" /> : <Plus className="h-4 w-4 text-primary" />}
              {editingRule ? 'Edit Auto-Categorization Rule' : 'Add Auto-Categorization Rule'}
            </DialogTitle>
            <DialogDescription className="text-xs">
              Match incoming transactions by merchant keyword, UPI VPA, or narration pattern
            </DialogDescription>
          </DialogHeader>

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3.5 py-2 text-xs">
            {/* Match Field */}
            <div className="space-y-1">
              <label className="font-semibold text-foreground">Target Field *</label>
              <Select
                value={ruleFormData.match_field}
                onValueChange={(val) => setRuleFormData({ ...ruleFormData, match_field: val || 'cleaned_payee' })}
              >
                <SelectTrigger className="text-xs h-8.5">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="cleaned_payee">Cleaned Payee Name</SelectItem>
                  <SelectItem value="raw_narration">Raw Statement Narration</SelectItem>
                  <SelectItem value="upi_vpa">UPI VPA Handle</SelectItem>
                  <SelectItem value="reference_number">Reference / UTR Number</SelectItem>
                </SelectContent>
              </Select>
            </div>

            {/* Match Type */}
            <div className="space-y-1">
              <label className="font-semibold text-foreground">Match Type *</label>
              <Select
                value={ruleFormData.match_type}
                onValueChange={(val) => setRuleFormData({ ...ruleFormData, match_type: val || 'CONTAINS' })}
              >
                <SelectTrigger className="text-xs h-8.5">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="CONTAINS">Contains Keyword</SelectItem>
                  <SelectItem value="EXACT">Exact Match</SelectItem>
                  <SelectItem value="STARTS_WITH">Starts With</SelectItem>
                  <SelectItem value="REGEX">Regular Expression (Regex)</SelectItem>
                </SelectContent>
              </Select>
            </div>

            {/* Pattern / Keyword */}
            <div className="sm:col-span-2 space-y-1">
              <label className="font-semibold text-foreground">Pattern / Match Keyword *</label>
              <Input
                placeholder="e.g. SWIGGY, ZOMATO, (?i)amazon|flipkart, @okhdfcbank"
                value={ruleFormData.match_pattern}
                onChange={(e) => setRuleFormData({ ...ruleFormData, match_pattern: e.target.value })}
                className="text-xs font-mono h-8.5"
              />
              <p className="text-[10px] text-muted-foreground">
                Matches are case-insensitive by default.
              </p>
            </div>

            {/* Exception / Exclude Keywords */}
            <div className="sm:col-span-2 space-y-1">
              <label className="font-semibold text-foreground">
                Exception / Exclude Keywords (Optional)
              </label>
              <Input
                placeholder="e.g. maid, driver, cook, advance, helper, staff, aws"
                value={ruleFormData.exclude_pattern}
                onChange={(e) => setRuleFormData({ ...ruleFormData, exclude_pattern: e.target.value })}
                className="text-xs font-mono h-8.5"
              />
              <p className="text-[10px] text-muted-foreground">
                Skip this rule if narration or payee contains any of these comma-separated keywords (or regex).
              </p>
            </div>

            {/* Target Category */}
            <div className="space-y-1">
              <label className="font-semibold text-foreground">Assign Category *</label>
              <Select
                value={ruleFormData.target_category_id}
                onValueChange={(val) => setRuleFormData({ ...ruleFormData, target_category_id: val || '' })}
              >
                <SelectTrigger className="text-xs h-8.5">
                  <SelectValue placeholder="Select Category" />
                </SelectTrigger>
                <SelectContent>
                  {categories?.map((c) => (
                    <SelectItem key={c.id} value={c.id}>
                      {c.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            {/* Transaction Direction */}
            <div className="space-y-1">
              <label className="font-semibold text-foreground">Transaction Direction</label>
              <Select
                value={ruleFormData.tx_type}
                onValueChange={(val) =>
                  setRuleFormData({ ...ruleFormData, tx_type: (val as 'ALL' | 'DEBIT' | 'CREDIT') || 'ALL' })
                }
              >
                <SelectTrigger className="text-xs h-8.5">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="ALL">All (Debits & Credits)</SelectItem>
                  <SelectItem value="CREDIT">Credits Only (Income / Deposits)</SelectItem>
                  <SelectItem value="DEBIT">Debits Only (Expenses / Payments)</SelectItem>
                </SelectContent>
              </Select>
            </div>

            {/* Priority */}
            <div className="space-y-1">
              <label className="font-semibold text-foreground">Priority (Higher runs first)</label>
              <Input
                type="number"
                min="1"
                max="1000"
                value={ruleFormData.priority || ''}
                onChange={(e) => setRuleFormData({ ...ruleFormData, priority: parseInt(e.target.value) || 50 })}
                className="text-xs font-mono h-8.5"
              />
            </div>

            {/* Assign Tags */}
            <div className="space-y-1">
              <label className="font-semibold text-foreground">Auto-Assign Tags (Optional)</label>
              <Input
                placeholder="e.g. food, delivery, online"
                value={ruleFormData.assign_tags}
                onChange={(e) => setRuleFormData({ ...ruleFormData, assign_tags: e.target.value })}
                className="text-xs h-8.5"
              />
            </div>
          </div>

          <DialogFooter className="gap-2 pt-2">
            <Button
              variant="outline"
              size="sm"
              onClick={() => {
                setShowAddRuleModal(false)
                setEditingRule(null)
              }}
            >
              Cancel
            </Button>
            <Button
              size="sm"
              onClick={() => {
                if (editingRule) {
                  updateRuleMutation.mutate({ id: editingRule.id, data: ruleFormData })
                } else {
                  createRuleMutation.mutate(ruleFormData)
                }
              }}
              disabled={
                createRuleMutation.isPending ||
                updateRuleMutation.isPending ||
                !ruleFormData.match_pattern ||
                !ruleFormData.target_category_id
              }
              className="font-semibold"
            >
              {editingRule ? 'Save Changes' : 'Create Rule'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete Rule Confirmation Dialog */}
      <Dialog open={!!showDeleteRuleModal} onOpenChange={(open) => !open && setShowDeleteRuleModal(null)}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle className="text-base font-semibold text-destructive flex items-center gap-2">
              <Trash2 className="h-5 w-5" /> Delete Categorization Rule
            </DialogTitle>
            <DialogDescription className="text-xs">
              Are you sure you want to delete the rule matching{' '}
              <code className="font-mono text-foreground font-semibold">"{showDeleteRuleModal?.match_pattern}"</code>?
              Existing categorized transactions will not be modified.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter className="gap-2 pt-2">
            <Button variant="outline" size="sm" onClick={() => setShowDeleteRuleModal(null)}>
              Cancel
            </Button>
            <Button
              variant="destructive"
              size="sm"
              onClick={() => {
                if (showDeleteRuleModal) {
                  deleteRuleMutation.mutate(showDeleteRuleModal.id)
                }
              }}
              disabled={deleteRuleMutation.isPending}
            >
              {deleteRuleMutation.isPending ? 'Deleting...' : 'Confirm Delete'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Add Custom Category Dialog */}
      <Dialog open={showAddCategoryModal} onOpenChange={setShowAddCategoryModal}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle className="text-base font-semibold flex items-center gap-2">
              <Plus className="h-4 w-4 text-primary" /> Add Custom Spending Category
            </DialogTitle>
            <DialogDescription className="text-xs">
              Create a custom category for specialized expense tracking and budgeting
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-3.5 py-2 text-xs">
            <div className="space-y-1">
              <label className="font-semibold text-foreground">Category Name *</label>
              <Input
                placeholder="e.g. Gaming & In-App, Pet Care, Crypto..."
                value={categoryFormData.name}
                onChange={(e) => setCategoryFormData({ ...categoryFormData, name: e.target.value })}
                className="text-xs h-8.5"
              />
            </div>

            <div className="space-y-1.5">
              <label className="font-semibold text-foreground">Category Color Accent</label>
              <div className="flex items-center gap-2">
                <div
                  className="h-7 w-7 rounded-lg border flex-shrink-0"
                  style={{ backgroundColor: categoryFormData.color_hex }}
                />
                <Input
                  type="text"
                  value={categoryFormData.color_hex}
                  onChange={(e) => setCategoryFormData({ ...categoryFormData, color_hex: e.target.value })}
                  className="text-xs font-mono h-8.5 max-w-[120px]"
                />
              </div>

              {/* Color Preset Palette */}
              <div className="flex flex-wrap gap-1.5 pt-1">
                {PRESET_COLORS.map((color) => (
                  <button
                    key={color}
                    type="button"
                    onClick={() => setCategoryFormData({ ...categoryFormData, color_hex: color })}
                    className={`h-5 w-5 rounded-full border transition-transform ${
                      categoryFormData.color_hex === color ? 'scale-125 ring-2 ring-primary ring-offset-1' : 'hover:scale-110'
                    }`}
                    style={{ backgroundColor: color }}
                  />
                ))}
              </div>
            </div>
          </div>

          <DialogFooter className="gap-2 pt-2">
            <Button variant="outline" size="sm" onClick={() => setShowAddCategoryModal(false)}>
              Cancel
            </Button>
            <Button
              size="sm"
              onClick={() => createCategoryMutation.mutate(categoryFormData)}
              disabled={createCategoryMutation.isPending || !categoryFormData.name.trim()}
              className="font-semibold"
            >
              {createCategoryMutation.isPending ? 'Creating...' : 'Create Category'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete Category Confirmation Dialog */}
      <Dialog open={!!showDeleteCategoryModal} onOpenChange={(open) => !open && setShowDeleteCategoryModal(null)}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle className="text-base font-semibold text-destructive flex items-center gap-2">
              <Trash2 className="h-5 w-5" /> Delete Category
            </DialogTitle>
            <DialogDescription className="text-xs">
              Are you sure you want to delete <strong className="text-foreground">{showDeleteCategoryModal?.name}</strong>?
              Any transactions or rules currently mapped to this category will be unassigned.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter className="gap-2 pt-2">
            <Button variant="outline" size="sm" onClick={() => setShowDeleteCategoryModal(null)}>
              Cancel
            </Button>
            <Button
              variant="destructive"
              size="sm"
              onClick={() => {
                if (showDeleteCategoryModal) {
                  deleteCategoryMutation.mutate(showDeleteCategoryModal.id)
                }
              }}
              disabled={deleteCategoryMutation.isPending}
            >
              {deleteCategoryMutation.isPending ? 'Deleting...' : 'Confirm Delete'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <UpdateDialog
        open={showUpdateModal}
        onOpenChange={setShowUpdateModal}
        versionInfo={versionInfo || null}
        onUpdateSuccess={() => refetchVersion()}
      />

      {/* Setup Master Password Dialog */}
      <Dialog open={showSetupAuthModal} onOpenChange={setShowSetupAuthModal}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle className="text-base font-semibold flex items-center gap-2">
              <KeyRound className="h-5 w-5 text-primary" /> Setup Master Password / PIN
            </DialogTitle>
            <DialogDescription className="text-xs">
              Configure a password or PIN to encrypt and restrict access to your local finance ledger
            </DialogDescription>
          </DialogHeader>

          <form
            onSubmit={(e) => {
              e.preventDefault()
              setSecurityActionError(null)
              if (setupPassword.length < 4) {
                setSecurityActionError('Password or PIN must be at least 4 characters')
                return
              }
              if (setupPassword !== setupConfirmPassword) {
                setSecurityActionError('Passwords do not match')
                return
              }
              setupAuthMutation.mutate()
            }}
            className="space-y-3.5 py-2 text-xs"
          >
            <div className="space-y-1">
              <label className="font-semibold text-foreground">Master Password or PIN *</label>
              <Input
                type="password"
                placeholder="Enter master password or PIN (min 4 chars)"
                value={setupPassword}
                onChange={(e) => {
                  setSetupPassword(e.target.value)
                  if (securityActionError) setSecurityActionError(null)
                }}
                className="text-xs font-mono h-8.5"
                autoFocus
              />
            </div>

            <div className="space-y-1">
              <label className="font-semibold text-foreground">Confirm Password or PIN *</label>
              <Input
                type="password"
                placeholder="Re-enter password or PIN"
                value={setupConfirmPassword}
                onChange={(e) => {
                  setSetupConfirmPassword(e.target.value)
                  if (securityActionError) setSecurityActionError(null)
                }}
                className="text-xs font-mono h-8.5"
              />
            </div>

            <div className="space-y-1">
              <label className="font-semibold text-foreground">Auto-Lock Inactivity Duration</label>
              <Select
                value={String(setupAutoLock)}
                onValueChange={(val) => val && setSetupAutoLock(parseInt(val, 10))}
              >
                <SelectTrigger className="text-xs h-8.5">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="15">15 Minutes</SelectItem>
                  <SelectItem value="30">30 Minutes</SelectItem>
                  <SelectItem value="60">1 Hour (Recommended)</SelectItem>
                  <SelectItem value="240">4 Hours</SelectItem>
                </SelectContent>
              </Select>
            </div>

            {securityActionError && (
              <div className="flex items-center gap-1.5 text-xs text-destructive font-medium pt-1">
                <AlertCircle className="h-3.5 w-3.5 shrink-0" />
                <span>{securityActionError}</span>
              </div>
            )}

            <DialogFooter className="gap-2 pt-3">
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={() => setShowSetupAuthModal(false)}
              >
                Cancel
              </Button>
              <Button
                type="submit"
                size="sm"
                disabled={setupAuthMutation.isPending || !setupPassword || !setupConfirmPassword}
                className="font-semibold"
              >
                {setupAuthMutation.isPending ? 'Enabling...' : 'Enable App Lock'}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      {/* Change Password Dialog */}
      <Dialog open={showChangePasswordModal} onOpenChange={setShowChangePasswordModal}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle className="text-base font-semibold flex items-center gap-2">
              <Edit2 className="h-5 w-5 text-primary" /> Change Master Password
            </DialogTitle>
            <DialogDescription className="text-xs">
              Verify your current master password to set a new password or PIN
            </DialogDescription>
          </DialogHeader>

          <form
            onSubmit={(e) => {
              e.preventDefault()
              setSecurityActionError(null)
              if (changeNewPassword.length < 4) {
                setSecurityActionError('New password must be at least 4 characters')
                return
              }
              if (changeNewPassword !== changeConfirmPassword) {
                setSecurityActionError('New passwords do not match')
                return
              }
              changePasswordMutation.mutate()
            }}
            className="space-y-3.5 py-2 text-xs"
          >
            <div className="space-y-1">
              <label className="font-semibold text-foreground">Current Password *</label>
              <Input
                type="password"
                placeholder="Enter current password"
                value={changeCurrentPassword}
                onChange={(e) => {
                  setChangeCurrentPassword(e.target.value)
                  if (securityActionError) setSecurityActionError(null)
                }}
                className="text-xs font-mono h-8.5"
                autoFocus
              />
            </div>

            <div className="space-y-1">
              <label className="font-semibold text-foreground">New Password *</label>
              <Input
                type="password"
                placeholder="Enter new password (min 4 chars)"
                value={changeNewPassword}
                onChange={(e) => {
                  setChangeNewPassword(e.target.value)
                  if (securityActionError) setSecurityActionError(null)
                }}
                className="text-xs font-mono h-8.5"
              />
            </div>

            <div className="space-y-1">
              <label className="font-semibold text-foreground">Confirm New Password *</label>
              <Input
                type="password"
                placeholder="Re-enter new password"
                value={changeConfirmPassword}
                onChange={(e) => {
                  setChangeConfirmPassword(e.target.value)
                  if (securityActionError) setSecurityActionError(null)
                }}
                className="text-xs font-mono h-8.5"
              />
            </div>

            {securityActionError && (
              <div className="flex items-center gap-1.5 text-xs text-destructive font-medium pt-1">
                <AlertCircle className="h-3.5 w-3.5 shrink-0" />
                <span>{securityActionError}</span>
              </div>
            )}

            <DialogFooter className="gap-2 pt-3">
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={() => setShowChangePasswordModal(false)}
              >
                Cancel
              </Button>
              <Button
                type="submit"
                size="sm"
                disabled={changePasswordMutation.isPending || !changeCurrentPassword || !changeNewPassword || !changeConfirmPassword}
                className="font-semibold"
              >
                {changePasswordMutation.isPending ? 'Updating...' : 'Update Password'}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      {/* Disable Auth Dialog */}
      <Dialog open={showDisableAuthModal} onOpenChange={setShowDisableAuthModal}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle className="text-base font-semibold text-destructive flex items-center gap-2">
              <ShieldAlert className="h-5 w-5" /> Disable App Lock &amp; Password
            </DialogTitle>
            <DialogDescription className="text-xs">
              Disabling app lock removes password protection. Any local user on this computer will be able to view your financial accounts and transactions.
            </DialogDescription>
          </DialogHeader>

          <form
            onSubmit={(e) => {
              e.preventDefault()
              if (!disablePassword) return
              disableAuthMutation.mutate()
            }}
            className="space-y-3.5 py-2 text-xs"
          >
            <div className="space-y-1">
              <label className="font-semibold text-foreground">Confirm Current Master Password *</label>
              <Input
                type="password"
                placeholder="Enter current password to confirm"
                value={disablePassword}
                onChange={(e) => {
                  setDisablePassword(e.target.value)
                  if (securityActionError) setSecurityActionError(null)
                }}
                className="text-xs font-mono h-8.5"
                autoFocus
              />
            </div>

            {securityActionError && (
              <div className="flex items-center gap-1.5 text-xs text-destructive font-medium pt-1">
                <AlertCircle className="h-3.5 w-3.5 shrink-0" />
                <span>{securityActionError}</span>
              </div>
            )}

            <DialogFooter className="gap-2 pt-3">
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={() => setShowDisableAuthModal(false)}
              >
                Cancel
              </Button>
              <Button
                type="submit"
                variant="destructive"
                size="sm"
                disabled={disableAuthMutation.isPending || !disablePassword}
                className="font-semibold"
              >
                {disableAuthMutation.isPending ? 'Disabling...' : 'Confirm & Disable'}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </div>
  )
}
