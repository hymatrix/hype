export type RootCommand = 'openclaw' | 'vmdocker'

export type OpenclawCommandKey = 'spawn' | 'conf-tg' | 'pair-tg' | 'chat'

export type VmdockerCommandKey = 'get' | 'init'

export type CommandKey = OpenclawCommandKey | VmdockerCommandKey

export type HealthResponse = {
  ok: boolean
  hypeBinary: string
  listen: string
  workingDir: string
  vmdockerDir: string
  hypeVersion?: string
  hymxVersion?: string
}

export type CommandResponse = {
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

export type VmdockerGetForm = {
  dir: string
  version: string
}

export type VmdockerInitForm = {
  dir: string
}
