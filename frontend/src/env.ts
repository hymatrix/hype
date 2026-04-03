export type ImportedEnv = {
  fileName: string
  content: string
  values: Record<string, string>
}

export function parseEnvContent(content: string): Record<string, string> {
  const values: Record<string, string> = {}

  for (const rawLine of content.split(/\r?\n/)) {
    const line = rawLine.trim()
    if (!line || line.startsWith('#')) {
      continue
    }

    const separator = line.indexOf('=')
    if (separator === -1) {
      continue
    }

    const key = line.slice(0, separator).trim()
    const value = line
      .slice(separator + 1)
      .trim()
      .replace(/^['"]+|['"]+$/g, '')

    if (!key || !value) {
      continue
    }

    values[key] = value
  }

  return values
}

export function firstEnvValue(values: Record<string, string>, keys: string[]): string {
  for (const key of keys) {
    const value = values[key]?.trim()
    if (value) {
      return value
    }
  }
  return ''
}
