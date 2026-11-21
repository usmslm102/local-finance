export interface StatementSettings {
  parserId: string
  accountId: string
  password?: string
}

// Preview and commit must receive the same staged settings. Defaults are copied
// when a file enters the queue, and changed only by an explicit re-preview.
export function statementForm(file: File, settings: StatementSettings): FormData {
  const form = new FormData()
  form.append('file', file)
  if (settings.parserId !== 'AUTO') form.append('parser_id', settings.parserId)
  if (settings.accountId !== 'AUTO') form.append('account_id', settings.accountId)
  if (settings.password) form.append('password', settings.password)
  return form
}
