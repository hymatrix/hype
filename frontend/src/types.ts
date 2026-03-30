export type CommandKey = 'spawn' | 'conf-tg' | 'pair-tg' | 'chat'

export type HealthResponse = {
  ok: boolean
  hypeBinary: string
  listen: string
  hypeVersion?: string
  hymxVersion?: string
}

export type OpenclawResponse = {
  ok: boolean
  exitCode: number
  command: string[]
  stdoutRaw: string
  stderrRaw: string
  parsedJson?: Record<string, unknown>
  spawnPid?: string
  spawnedPids?: string[]
  error?: string
  durationMs: number
}

export type SharedForm = {
  nodeUrl: string
  privateKey: string
}

export type SpawnForm = {
  moduleId: string
  scheduler: string
  model: string
  provider: string
  apiKey: string
  gatewayToken: string
  runtimeBackend: string
  botToken: string
  defaultAccount: string
  dmPolicy: string
  allowFrom: string
}

export type ConfTGForm = {
  pid: string
  botToken: string
  defaultAccount: string
  dmPolicy: string
  allowFrom: string
}

export type PairTGForm = {
  pid: string
  code: string
  channel: string
  dmPolicy: string
}

export type ChatForm = {
  pid: string
  command: string
}
