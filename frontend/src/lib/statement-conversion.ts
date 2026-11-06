import skill from '@/content/localfinance-statement-csv/SKILL.md?raw'

// Keep the one-off prompt and reusable skill on the same CSV contract.
export const statementConversionPrompt = skill.replace(/^---\r?\n[\s\S]*?\r?\n---\r?\n/, '').trim()
