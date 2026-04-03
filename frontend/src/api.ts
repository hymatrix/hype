import type { CommandKey, CommandResponse, HealthResponse, RootCommand } from './types'

export async function fetchHealth(): Promise<HealthResponse> {
  const response = await fetch('/api/health')
  if (!response.ok) {
    throw new Error(`health request failed: ${response.status}`)
  }
  return response.json()
}

export async function loadEnvFile(path: string): Promise<{ fileName: string; path: string; content: string }> {
  const response = await fetch('/api/env/load', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ path }),
  })

  const data = (await response.json()) as {
    ok: boolean
    fileName?: string
    path?: string
    content?: string
    error?: string
  }
  if (!response.ok || !data.ok || !data.content || !data.fileName || !data.path) {
    throw new Error(data.error || `request failed: ${response.status}`)
  }
  return {
    fileName: data.fileName,
    path: data.path,
    content: data.content,
  }
}

export async function runCommand(
  rootCommand: RootCommand,
  command: CommandKey,
  body: Record<string, string>,
): Promise<CommandResponse> {
  const response = await fetch(`/api/${rootCommand}/${command}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })

  const data = (await response.json()) as CommandResponse
  if (!response.ok) {
    throw new Error(data.error || `request failed: ${response.status}`)
  }
  return data
}
