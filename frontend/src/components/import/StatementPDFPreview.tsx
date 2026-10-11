import { useEffect, useRef, useState } from 'react'
import { Link } from '@tanstack/react-router'
import type { PDFDocumentLoadingTask, PDFDocumentProxy } from 'pdfjs-dist'
import workerURL from 'pdfjs-dist/build/pdf.worker.min.mjs?url'
import { AlertCircle, ChevronLeft, ChevronRight, Download, RefreshCw, ZoomIn, ZoomOut } from 'lucide-react'
import { fetchStatementPDF } from '@/lib/api'
import type { StatementImport } from '@/types'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button, buttonVariants } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Field, FieldDescription, FieldLabel } from '@/components/ui/field'
import { Input } from '@/components/ui/input'

export function StatementPDFPreview({ statement, onClose }: { statement: StatementImport; onClose: () => void }) {
  const [url, setURL] = useState<string>()
  const [pdf, setPDF] = useState<PDFDocumentProxy>()
  const [error, setError] = useState<string>()
  const [attempt, setAttempt] = useState(0)
  const [pageNumber, setPageNumber] = useState(1)
  const [zoom, setZoom] = useState(1)
  const [width, setWidth] = useState(0)
  const [rendering, setRendering] = useState(true)
  const [passwordReason, setPasswordReason] = useState<number>()
  const [password, setPassword] = useState('')
  const passwordCallback = useRef<((password: string) => void) | undefined>(undefined)
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const viewportRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const controller = new AbortController()
    let objectURL: string | undefined
    let task: PDFDocumentLoadingTask | undefined
    setURL(undefined)
    setPDF(undefined)
    setError(undefined)
    setPasswordReason(undefined)
    setPassword('')
    setPageNumber(1)
    setZoom(1)
    setRendering(true)
    async function load() {
      try {
        const blob = await fetchStatementPDF(statement.id, controller.signal)
        const pdfjs = await import('pdfjs-dist')
        const data = new Uint8Array(await blob.arrayBuffer())
        if (controller.signal.aborted) return
        objectURL = URL.createObjectURL(blob)
        setURL(objectURL)
        pdfjs.GlobalWorkerOptions.workerSrc = workerURL
        task = pdfjs.getDocument({ data, useSystemFonts: true, useWasm: false })
        task.onPassword = (updatePassword: (value: string) => void, reason: number) => {
          if (controller.signal.aborted) return
          passwordCallback.current = updatePassword
          setPasswordReason(reason)
          setPassword('')
        }
        const document = await task.promise
        if (!controller.signal.aborted) {
          passwordCallback.current = undefined
          setPasswordReason(undefined)
          setPassword('')
          setPDF(document)
        }
      } catch (err) {
        if (!controller.signal.aborted) {
          setError(err instanceof Error ? err.message : 'Unable to load the statement PDF. Please try again.')
        }
      }
    }
    void load()
    return () => {
      controller.abort()
      passwordCallback.current = undefined
      if (task) void task.destroy()
      if (objectURL) URL.revokeObjectURL(objectURL)
    }
  }, [statement.id, attempt])

  useEffect(() => {
    const viewport = viewportRef.current
    if (!viewport) return
    const observer = new ResizeObserver(() => setWidth(Math.max(1, viewport.clientWidth - 32)))
    observer.observe(viewport)
    return () => observer.disconnect()
  }, [pdf])

  useEffect(() => {
    const canvas = canvasRef.current
    if (!pdf || !canvas || !width) return
    let cancelled = false
    let task: ReturnType<Awaited<ReturnType<PDFDocumentProxy['getPage']>>['render']> | undefined
    setRendering(true)
    async function render() {
      try {
        const page = await pdf!.getPage(pageNumber)
        if (cancelled) return
        const base = page.getViewport({ scale: 1 })
        const viewport = page.getViewport({ scale: width * zoom / base.width })
        const ratio = Math.min(window.devicePixelRatio || 1, 2)
        canvas!.width = Math.ceil(viewport.width * ratio)
        canvas!.height = Math.ceil(viewport.height * ratio)
        canvas!.style.width = `${viewport.width}px`
        canvas!.style.height = `${viewport.height}px`
        task = page.render({ canvas: canvas!, viewport, transform: [ratio, 0, 0, ratio, 0, 0] })
        await task.promise
        if (!cancelled) setRendering(false)
      } catch (err) {
        if (!cancelled) setError(err instanceof Error ? err.message : 'Unable to display this page. Try downloading the PDF.')
      }
    }
    void render()
    return () => {
      cancelled = true
      task?.cancel()
      // Release the backing bitmap on page/zoom changes and modal unmount.
      canvas.width = 0
      canvas.height = 0
    }
  }, [pdf, pageNumber, zoom, width])

  return (
    <Dialog open onOpenChange={(open) => { if (!open) onClose() }}>
      <DialogContent className="flex max-w-5xl flex-col overflow-hidden">
        <DialogHeader className="min-w-0 pr-8">
          <DialogTitle>Imported statement PDF</DialogTitle>
          <DialogDescription className="break-all">{statement.filename}</DialogDescription>
        </DialogHeader>
        {error ? (
          <Alert variant="destructive">
            <AlertCircle />
            <AlertTitle>PDF preview unavailable</AlertTitle>
            <AlertDescription>
              <p>{error}</p>
              <Button variant="outline" size="sm" onClick={() => setAttempt((value) => value + 1)}>Try again</Button>
            </AlertDescription>
          </Alert>
        ) : passwordReason ? (
          <form onSubmit={(event) => { event.preventDefault(); passwordCallback.current?.(password); setPassword('') }} className="flex flex-col gap-4">
            <Field data-invalid={passwordReason === 2}>
              <FieldLabel htmlFor="statement-preview-password">PDF password</FieldLabel>
              <Input id="statement-preview-password" type="password" autoComplete="off" value={password} onChange={(event) => setPassword(event.target.value)} aria-invalid={passwordReason === 2} autoFocus />
              <FieldDescription>{passwordReason === 2 ? 'Incorrect password. Please try again.' : 'Enter the original PDF password to preview this statement. It is not saved.'}</FieldDescription>
            </Field>
            <Button type="submit" disabled={!password}>Unlock preview</Button>
          </form>
        ) : pdf ? (
          <div className="flex min-h-0 flex-col gap-3">
            <div className="flex flex-wrap items-center justify-between gap-2">
              <div className="flex items-center gap-2">
                <Button variant="outline" size="icon-sm" aria-label="Previous page" disabled={pageNumber === 1} onClick={() => setPageNumber((value) => value - 1)}><ChevronLeft /></Button>
                <span className="text-sm tabular-nums" aria-live="polite">Page {pageNumber} of {pdf.numPages}</span>
                <Button variant="outline" size="icon-sm" aria-label="Next page" disabled={pageNumber === pdf.numPages} onClick={() => setPageNumber((value) => value + 1)}><ChevronRight /></Button>
              </div>
              <div className="flex items-center gap-2">
                <Button variant="outline" size="icon-sm" aria-label="Zoom out" disabled={zoom <= 1} onClick={() => setZoom((value) => value - 0.25)}><ZoomOut /></Button>
                <span className="text-sm tabular-nums">{Math.round(zoom * 100)}%</span>
                <Button variant="outline" size="icon-sm" aria-label="Zoom in" disabled={zoom >= 2} onClick={() => setZoom((value) => value + 0.25)}><ZoomIn /></Button>
              </div>
            </div>
            <div ref={viewportRef} className="h-[60dvh] min-h-0 overflow-auto rounded-lg border bg-muted p-4" aria-busy={rendering}>
              <canvas ref={canvasRef} role="img" aria-label={`Statement PDF, page ${pageNumber} of ${pdf.numPages}. Download the original PDF for text selection and accessibility tools.`} />
            </div>
          </div>
        ) : (
          <div role="status" className="flex h-[60dvh] items-center justify-center gap-2 text-sm text-muted-foreground">
            <RefreshCw className="size-4 animate-spin" aria-hidden="true" /> Loading PDF…
          </div>
        )}
        <p className="text-xs text-muted-foreground">The original PDF stays on this device, including its password protection.</p>
        <DialogFooter>
          <Button variant="outline" onClick={onClose}>Close</Button>
          <Link data-slot="button" to="/transactions" search={{ statement_import_id: statement.id }} onClick={onClose} className={buttonVariants({ variant: 'outline' })}>View imported records</Link>
          {url && <a data-slot="button" href={url} download={statement.filename} className={buttonVariants({ variant: 'outline' })}><Download data-icon="inline-start" />Download PDF</a>}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
