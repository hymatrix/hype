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

export type CatalogOption = {
  label: string
  value: string
}

export type CatalogFieldKind = 'string' | 'multiline' | 'bool' | 'int' | 'int64' | 'enum' | 'path' | 'secret'

export type CatalogField = {
  name: string
  label: string
  description?: string
  kind: CatalogFieldKind
  defaultValue?: string
  required?: boolean
  options?: CatalogOption[]
  envKeys?: string[]
  group?: string
}

export type CatalogCommand = {
  name: string
  title: string
  path: string[]
  description?: string
  supported: boolean
  disabledReason?: string
  importedEnvRequired?: boolean
  fields: CatalogField[]
}

export type CatalogRoot = {
  name: string
  title: string
  description?: string
  commands: CatalogCommand[]
}

export type CatalogResponse = {
  ok: boolean
  roots: CatalogRoot[]
}

export type ImportedEnvState = {
  fileName: string
  sourcePath?: string
  content: string
  values: Record<string, string>
}

export type RunRequest = {
  path: string[]
  values: Record<string, string | boolean>
  importedEnv?: {
    fileName: string
    content: string
  }
}
