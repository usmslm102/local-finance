import { useId, useRef, useState } from 'react'
import { Link } from '@tanstack/react-router'
import { Check, Copy, Download, FileText, ShieldCheck } from 'lucide-react'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button, buttonVariants } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Field, FieldDescription, FieldLabel } from '@/components/ui/field'
import { Textarea } from '@/components/ui/textarea'
import skillUrl from '@/content/localfinance-statement-csv/SKILL.md?url'
import { statementConversionPrompt } from '@/lib/statement-conversion'

export function StatementConversionHelp({
  open,
  onOpenChange,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
}) {
  const promptId = useId()
  const promptRef = useRef<HTMLTextAreaElement>(null)
  const [copyStatus, setCopyStatus] = useState<'idle' | 'copied' | 'error'>('idle')

  async function copyPrompt() {
    try {
      await navigator.clipboard.writeText(statementConversionPrompt)
      setCopyStatus('copied')
    } catch {
      setCopyStatus('error')
      promptRef.current?.focus()
      promptRef.current?.select()
    }
  }

  return (
    <Dialog open={open} onOpenChange={(nextOpen) => {
      setCopyStatus('idle')
      onOpenChange(nextOpen)
    }}>
      <DialogContent className="max-w-2xl">
        <DialogHeader className="pr-6">
          <DialogTitle>Convert an unsupported statement</DialogTitle>
          <DialogDescription>
            Give your agent the prompt and your PDF. It can create a standard CSV for
            LocalFinance, even when your bank or statement format isn’t supported.
          </DialogDescription>
        </DialogHeader>

        <ol className="flex list-decimal flex-col gap-3 pl-5">
          <li>Copy the prompt below, or download the reusable skill, and give it to your agent with the PDF.</li>
          <li>Check its CSV and verification summary against your original statement. This format supports INR transactions.</li>
          <li>
            Upload the CSV here using Auto-Detect Bank &amp; Format and review Preview Records.
            Or ask a connected agent to upload the original PDF or its verified CSV through MCP. Enable{' '}
            <strong>Allow statement and investment uploads</strong> in{' '}
            <Link to="/settings" search={{ tab: 'mcp' }} className="underline underline-offset-4">
              Settings → AI / MCP → Advanced settings &amp; permissions
            </Link>.
          </li>
        </ol>

        <Alert>
          <ShieldCheck />
          <AlertTitle>You choose where your PDF goes</AlertTitle>
          <AlertDescription>
            LocalFinance does not send statements to AI services. Your chosen agent may share
            them with its model provider. Use local tools and a local model to keep conversion on your computer.
          </AlertDescription>
        </Alert>

        <Field>
          <FieldLabel htmlFor={promptId}>Prompt for your agent</FieldLabel>
          <Textarea
            ref={promptRef}
            id={promptId}
            value={statementConversionPrompt}
            readOnly
            spellCheck={false}
            className="h-48 resize-y field-sizing-fixed"
            aria-describedby={`${promptId}-help`}
          />
          <FieldDescription id={`${promptId}-help`}>
            Includes the CSV schema, verification steps, and both upload options. For repeat
            use, save the downloaded file as localfinance-statement-csv/SKILL.md in your agent’s skills folder.
            Card billing details such as due dates are not included in this CSV format.
          </FieldDescription>
        </Field>

        <p role="status" className="text-sm text-muted-foreground">
          {copyStatus === 'copied'
            ? 'Prompt copied. Paste it into your agent’s conversation and attach the PDF.'
            : copyStatus === 'error'
              ? 'Clipboard unavailable. The prompt is selected; copy it manually.'
              : 'The prompt and skill are bundled with LocalFinance and work offline.'}
        </p>

        <DialogFooter>
          <a href={skillUrl} download="SKILL.md" data-slot="button" className={buttonVariants({ variant: 'outline' })}>
            <Download data-icon="inline-start" /> Download agent skill
          </a>
          <Button onClick={copyPrompt}>
            {copyStatus === 'copied' ? <Check data-icon="inline-start" /> : <Copy data-icon="inline-start" />}
            {copyStatus === 'copied' ? 'Prompt copied' : 'Copy prompt'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

export function StatementConversionNotice({ onOpen }: { onOpen: () => void }) {
  return (
    <Alert>
      <FileText />
      <AlertTitle>Bank or statement format not supported?</AlertTitle>
      <AlertDescription>
        <p>Use your agent to turn a PDF into a CSV you can import here.</p>
        <Button variant="outline" onClick={onOpen}>Convert with your agent</Button>
      </AlertDescription>
    </Alert>
  )
}
