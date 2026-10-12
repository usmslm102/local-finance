import { fetchWithAuth } from './api'

async function checked(response: Response) {
  if (!response.ok) {
    const body = await response.json().catch(() => ({}))
    throw new Error(body.error || 'Unable to complete the desktop action')
  }
}

// Use the normal authenticated REST interface for narrow native OS actions.
// Browser builds keep their standard links and download behavior.
export async function installDesktopActions() {
  const response = await fetchWithAuth('/api/desktop').catch(() => null)
  if (!response?.ok || !(await response.json().catch(() => ({ enabled: false }))).enabled) return

  const openURL = async (url: string) => {
    await checked(await fetchWithAuth('/api/desktop/open-url', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ url }),
    }))
  }
  const download = async (anchor: HTMLAnchorElement) => {
    const source = new URL(anchor.href)
    // Never attach the UI credential to an external download URL.
    if (source.origin !== window.location.origin || !['http:', 'blob:'].includes(source.protocol)) {
      throw new Error('Only local application files can use the native save dialog')
    }
    const response = await fetchWithAuth(anchor.href)
    await checked(response)
    const content = await response.blob()
    const disposition = response.headers.get('Content-Disposition') || ''
    const filename = anchor.download || disposition.match(/filename="?([^";]+)"?/)?.[1] || 'localfinance-export'
    await checked(await fetchWithAuth(`/api/desktop/save?name=${encodeURIComponent(filename)}`, {
      method: 'POST', body: content,
    }))
  }
  window.open = (url) => {
    if (url) void openURL(String(url)).catch(error => window.alert(error.message))
    return null
  }
  document.addEventListener('click', event => {
    const anchor = event.target instanceof Element ? event.target.closest('a') : null
    if (!anchor) return
    if (anchor.hasAttribute('download')) {
      event.preventDefault()
      void download(anchor).catch(error => window.alert(error.message))
    } else if (anchor.protocol === 'https:' || anchor.protocol === 'http:') {
      if (anchor.origin !== window.location.origin) {
        event.preventDefault()
        void openURL(anchor.href).catch(error => window.alert(error.message))
      }
    }
  }, true)
}
