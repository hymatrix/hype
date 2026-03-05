import type { CommandKey, HealthResponse, OpenclawResponse } from './types'

export async function fetchHealth(): Promise<HealthResponse> {
  const response = await fetch('/api/health')
  if (!response.ok) {
    throw new Error(`health request failed: ${response.status}`)
  }
  return response.json()
}

export async function runCommand(command: CommandKey, body: Record<string, string>): Promise<OpenclawResponse> {
  const response = await fetch(`/api/openclaw/${command}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })

  const data = (await response.json()) as OpenclawResponse
  if (!response.ok) {
    throw new Error(data.error || `request failed: ${response.status}`)
  }
  return data
}
