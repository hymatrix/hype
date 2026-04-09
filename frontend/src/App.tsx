import { useEffect, useMemo, useRef, useState } from 'react'
import type { ChangeEvent, FormEvent } from 'react'
import { fetchCatalog, fetchHealth, loadEnvFile, runCommand } from './api'
import { firstEnvValue, parseEnvContent } from './env'
import type {
  CatalogCommand,
  CatalogField,
  CatalogRoot,
  CommandResponse,
  HealthResponse,
  ImportedEnvState,
  RunRequest,
} from './types'

const importedEnvPreviewKeys = [
  'VMDOCKER_PRIVATE_KEY',
  'VMDOCKER_MODULE_ID',
  'VMDOCKER_SCHEDULER',
  'RUNTIME_BACKEND',
  'OPENCLAW_MODULE_ID',
  'OPENCLAW_MODEL',
  'OPENCLAW_PROVIDER',
  'OPENCLAW_API_KEY',
  'OPENCLAW_GATEWAY_TOKEN',
  'OPENCLAW_TELEGRAM_BOT_TOKEN',
  'ANTHROPIC_API_KEY',
  'ANTHROPIC_BASE_URL',
  'ANTHROPIC_MODEL',
  'CLAUDE_CODE_FLAGS',
  'REDIS_URL',
] as const

type CommandValuesMap = Record<string, Record<string, string | boolean>>
type EnvAppliedFieldsMap = Record<string, Record<string, true>>

export function App() {
  const [catalog, setCatalog] = useState<CatalogRoot[]>([])
  const [selectedRoot, setSelectedRoot] = useState('')
  const [selectedCommand, setSelectedCommand] = useState('')
  const [commandValues, setCommandValues] = useState<CommandValuesMap>({})
  const [importedEnv, setImportedEnv] = useState<ImportedEnvState | null>(null)
  const [envPath, setEnvPath] = useState('./vmdocker/.env')
  const [envStatus, setEnvStatus] = useState('')
  const [envError, setEnvError] = useState('')
  const [showEnvViewer, setShowEnvViewer] = useState(false)
  const [health, setHealth] = useState<HealthResponse | null>(null)
  const [loading, setLoading] = useState(false)
  const [bootError, setBootError] = useState('')
  const [error, setError] = useState('')
  const [result, setResult] = useState<CommandResponse | null>(null)
  const [copyFeedback, setCopyFeedback] = useState('')
  const commandValuesRef = useRef<CommandValuesMap>({})
  const envAppliedFieldsRef = useRef<EnvAppliedFieldsMap>({})

  function setCommandValuesAndRef(updater: (current: CommandValuesMap) => CommandValuesMap) {
    setCommandValues((current) => {
      const next = updater(current)
      commandValuesRef.current = next
      return next
    })
  }

  useEffect(() => {
    Promise.all([fetchHealth(), fetchCatalog()])
      .then(([healthData, catalogData]) => {
        setHealth(healthData)
        setCatalog(catalogData.roots)
        setCommandValuesAndRef((current) => {
          const withDefaults = mergeCommandValues(catalogData.roots, current)
          return withHealthDefaults(catalogData.roots, withDefaults, healthData.vmdockerDir)
        })

        const firstRoot = catalogData.roots[0]
        if (firstRoot) {
          setSelectedRoot((current) => current || firstRoot.name)
          setSelectedCommand((current) => current || firstRoot.commands[0]?.name || '')
        }
      })
      .catch((err: Error) => setBootError(err.message))
  }, [])

  useEffect(() => {
    commandValuesRef.current = commandValues
  }, [commandValues])

  const activeRoot = useMemo(
    () => catalog.find((root) => root.name === selectedRoot) ?? catalog[0] ?? null,
    [catalog, selectedRoot],
  )
  const activeCommand = useMemo(
    () => activeRoot?.commands?.find((command) => command.name === selectedCommand) ?? activeRoot?.commands?.[0] ?? null,
    [activeRoot, selectedCommand],
  )
  const activeCommandKey = activeCommand ? pathKey(activeCommand.path) : ''
  const activeValues = activeCommand ? commandValues[activeCommandKey] ?? buildDefaultValues(activeCommand) : {}

  useEffect(() => {
    if (!activeRoot) {
      return
    }
    if (selectedRoot !== activeRoot.name) {
      setSelectedRoot(activeRoot.name)
      return
    }
    if (!activeRoot.commands?.some((command) => command.name === selectedCommand)) {
      setSelectedCommand(activeRoot.commands?.[0]?.name || '')
    }
  }, [activeRoot, selectedCommand, selectedRoot])

  useEffect(() => {
    if (catalog.length === 0 || !importedEnv) {
      return
    }

    const result = applyEnvMappings(catalog, commandValuesRef.current, importedEnv.values, envAppliedFieldsRef.current)
    envAppliedFieldsRef.current = result.appliedFields
    setCommandValuesAndRef(() => result.values)
  }, [catalog, importedEnv])

  const hasImportedEnv = Boolean(importedEnv?.content.trim())
  const hasImportedPrivateKey = Boolean(firstEnvValue(importedEnv?.values ?? {}, ['VMDOCKER_PRIVATE_KEY', 'HYPE_PRIVATE_KEY', 'PRV_KEY']))
  const spawnedPids = result?.spawnedPids ?? []
  const importedEnvKeys = useMemo(() => {
    if (!importedEnv) {
      return []
    }
    return importedEnvPreviewKeys.filter((key) => Boolean(importedEnv.values[key]))
  }, [importedEnv])
  const importedEnvEntries = useMemo(() => {
    if (!importedEnv) {
      return []
    }
    return Object.entries(importedEnv.values).sort(([left], [right]) => left.localeCompare(right))
  }, [importedEnv])
  const fieldGroups = useMemo(() => groupFields(activeCommand), [activeCommand])
  const runDisabled = loading || !activeCommand || !activeCommand.supported || (activeCommand?.importedEnvRequired && !hasImportedEnv)

  async function copyPID(pid: string) {
    try {
      await navigator.clipboard.writeText(pid)
      setCopyFeedback(`Copied: ${pid}`)
      setTimeout(() => setCopyFeedback(''), 1500)
    } catch {
      setCopyFeedback('Copy failed')
      setTimeout(() => setCopyFeedback(''), 1500)
    }
  }

  function applyImportedEnv(values: Record<string, string>, fileName: string, sourcePath?: string, content = '') {
    setImportedEnv({
      fileName,
      sourcePath,
      content,
      values,
    })
  }

  async function onEnvImport(e: ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0]
    e.target.value = ''
    if (!file) {
      return
    }

    setEnvError('')
    try {
      const content = await file.text()
      const values = parseEnvContent(content)
      applyImportedEnv(values, file.name, undefined, content)
      setEnvStatus(buildEnvStatus(values, file.name))
    } catch (err) {
      setImportedEnv(null)
      setEnvStatus('')
      setEnvError(err instanceof Error ? err.message : 'Failed to import .env file')
    }
  }

  async function onEnvPathImport() {
    setEnvError('')
    setEnvStatus('')

    try {
      const loaded = await loadEnvFile(envPath)
      const values = parseEnvContent(loaded.content)
      applyImportedEnv(values, loaded.fileName, loaded.path, loaded.content)
      setEnvStatus(buildEnvStatus(values, loaded.path))
    } catch (err) {
      setImportedEnv(null)
      setEnvError(err instanceof Error ? err.message : 'Failed to load env file path')
    }
  }

  function clearImportedEnv() {
    const clearedValues = clearEnvMappedValues(catalog, commandValuesRef.current, envAppliedFieldsRef.current)
    envAppliedFieldsRef.current = {}
    setCommandValuesAndRef(() => clearedValues)
    setImportedEnv(null)
    setEnvStatus('')
    setEnvError('')
    setShowEnvViewer(false)
  }

  function updateFieldValue(command: CatalogCommand, field: CatalogField, value: string | boolean) {
    const key = pathKey(command.path)
    envAppliedFieldsRef.current = removeEnvAppliedField(envAppliedFieldsRef.current, key, field.name)
    setCommandValuesAndRef((current) => ({
      ...current,
      [key]: {
        ...(current[key] ?? buildDefaultValues(command)),
        [field.name]: value,
      },
    }))
  }

  async function onSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault()
    setError('')

    if (!activeCommand) {
      setError('No command selected')
      return
    }
    if (!activeCommand.supported) {
      setError(activeCommand.disabledReason || 'This command cannot run from the embedded UI.')
      return
    }
    if (activeCommand.importedEnvRequired && !importedEnv?.content.trim()) {
      setError('Import a .env file before running this command.')
      return
    }

    const missing = activeCommand.fields.find((field) => field.required && isMissingValue(activeValues[field.name], field.kind))
    if (missing) {
      setError(`${missing.label} is required`)
      return
    }

    const payload: RunRequest = {
      path: activeCommand.path,
      values: serializeValues(activeCommand, activeValues),
    }
    if (activeCommand.importedEnvRequired && importedEnv) {
      payload.importedEnv = {
        fileName: importedEnv.fileName,
        content: importedEnv.content,
      }
    }

    setLoading(true)
    try {
      const data = await runCommand(payload)
      setResult(data)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'request failed')
      setResult(null)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="page">
      <header className="topbar">
        <div>
          <h1>HypeUI</h1>
          <p>
            hype version: {health?.hypeVersion || 'unknown'} | HymxVersion: {health?.hymxVersion || 'unknown'}
          </p>
        </div>
        <div className="health">
          <div>Status: {health?.ok ? 'online' : 'offline'}</div>
          <div>Listen: {health?.listen || '127.0.0.1:7788'}</div>
          <div className="binary">Binary: {health?.hypeBinary || 'detecting...'}</div>
        </div>
      </header>

      <section className="env-import">
        <div>
          <p className="section-kicker">.env Import</p>
          <h2>Load shared runtime values before you run UI commands</h2>
          <p className="env-import-copy">Imported values stay in memory only and prefill matching fields across commands. They are not injected into command processes.</p>
        </div>
        <div className="env-import-actions">
          <div className="env-import-inline">
            <label className="file-btn">
              Import .env
              <input type="file" accept=".env,text/plain" onChange={onEnvImport} />
            </label>
            <input
              className="env-path-input"
              value={envPath}
              onChange={(e) => setEnvPath(e.target.value)}
              placeholder="./vmdocker/.env"
            />
            <button type="button" className="secondary-btn" onClick={onEnvPathImport}>
              Load Path
            </button>
            <button type="button" className="secondary-btn" onClick={() => setShowEnvViewer(true)} disabled={!importedEnv}>
              View Env
            </button>
            <button type="button" className="secondary-btn" onClick={clearImportedEnv} disabled={!importedEnv}>
              Clear Import
            </button>
          </div>
        </div>
        <div className="env-import-status">
          <p>{importedEnv ? `File: ${importedEnv.fileName}` : 'No .env file imported yet.'}</p>
          {importedEnv?.sourcePath && <p>Path: {importedEnv.sourcePath}</p>}
          {envStatus && <p className={hasImportedPrivateKey ? 'env-status ok' : 'env-status warn'}>{envStatus}</p>}
          {envError && <p className="env-status error">{envError}</p>}
        </div>
      </section>

      <main className="layout">
        <aside className="sidebar" aria-label="Command navigation">
          <label>
            Root Command
            <select value={activeRoot?.name ?? ''} onChange={(e) => setSelectedRoot(e.target.value)}>
              {catalog.map((root) => (
                <option key={root.name} value={root.name}>
                  {root.name}
                </option>
              ))}
            </select>
          </label>

          {(activeRoot?.commands ?? []).map((command) => (
            <button
              key={command.name}
              type="button"
              className={command.name === activeCommand?.name ? 'nav-btn active' : 'nav-btn'}
              onClick={() => setSelectedCommand(command.name)}
            >
              {command.name}
            </button>
          ))}
        </aside>

        <section className="panel">
          <div className="form-column">
            <div className="command-head">
              <p className="section-kicker">Command</p>
              <h2>{activeCommand ? activeCommand.path.join(' / ') : 'Loading commands...'}</h2>
              <p>{activeCommand?.description || activeRoot?.description || 'Select a command to configure and run.'}</p>
            </div>

            {bootError && <p className="error">{bootError}</p>}

            {activeCommand && !activeCommand.supported ? (
              <section className="unsupported-card">
                <h3>Unavailable in Web UI</h3>
                <p>{activeCommand.disabledReason}</p>
              </section>
            ) : (
              <form onSubmit={onSubmit}>
                {fieldGroups.map(([group, fields]) => (
                  <fieldset key={group}>
                    <legend>{group === 'shared' ? 'Shared' : activeCommand?.title || 'Command'}</legend>
                    {(fields ?? []).map((field) => renderField(field, activeValues[field.name], activeCommand, updateFieldValue))}
                  </fieldset>
                ))}

                {activeCommand?.importedEnvRequired && (
                  <p className="field-hint">
                    This command uses the imported <code>.env</code> content. Import is required before execution.
                  </p>
                )}

                <button disabled={runDisabled} className="run-btn" type="submit">
                  {loading ? 'Running...' : activeCommand ? `Run ${activeCommand.path.join(' ')}` : 'Run'}
                </button>
              </form>
            )}
          </div>

          <div className="result-column">
            <section className="env-summary" aria-live="polite">
              <div className="result-head">
                <h2>Imported .env</h2>
                {importedEnv && <span className={hasImportedPrivateKey ? 'env-pill ok' : 'env-pill warn'}>{hasImportedPrivateKey ? 'Shared values ready' : 'Missing private key'}</span>}
              </div>
              {!importedEnv ? (
                <p>No imported .env yet.</p>
              ) : (
                <>
                  <p className="env-summary-meta">
                    {importedEnv.sourcePath ?? importedEnv.fileName} | {Object.keys(importedEnv.values).length} keys loaded
                  </p>
                  {importedEnvKeys.length > 0 ? (
                    <ul className="env-key-list">
                      {importedEnvKeys.map((key) => (
                        <li key={key}>
                          <code>{key}</code>
                        </li>
                      ))}
                    </ul>
                  ) : (
                    <p>No mapped keys detected yet.</p>
                  )}
                </>
              )}
            </section>

            {(activeCommandKey === 'openclaw/spawn' || activeCommandKey === 'claude/spawn') && (
              <section className="spawn-pids" aria-live="polite">
                <div className="result-head">
                  <h2>Spawned PIDs</h2>
                  {copyFeedback && <span className="copy-feedback">{copyFeedback}</span>}
                </div>
                {spawnedPids.length === 0 ? (
                  <p>No pid yet.</p>
                ) : (
                  <ul className="pid-list">
                    {spawnedPids.map((pid, index) => (
                      <li key={`${pid}-${index}`}>
                        <button type="button" className="pid-copy-btn" onClick={() => copyPID(pid)} title="Click to copy pid">
                          <code>{pid}</code>
                        </button>
                      </li>
                    ))}
                  </ul>
                )}
              </section>
            )}

            <section className="result" aria-live="polite">
              <h2>Result</h2>
              <div className="result-body">
                {error && <p className="error">{error}</p>}
                {!error && !result && <p>No execution yet.</p>}
                {result && (
                  <>
                    <p>
                      ok={String(result.ok)} | exitCode={result.exitCode} | duration={result.durationMs}ms
                    </p>
                    <p className="cmd-line">{result.command.join(' ')}</p>
                    {typeof result.parsedJson?.reply === 'string' && (
                      <div className="reply-box">
                        <strong>Reply:</strong>
                        <pre>{String(result.parsedJson.reply)}</pre>
                      </div>
                    )}
                    {result.parsedJson && (
                      <details open>
                        <summary>parsedJson</summary>
                        <pre>{JSON.stringify(result.parsedJson, null, 2)}</pre>
                      </details>
                    )}
                    <details open={!result.ok && Boolean(result.stdoutRaw)}>
                      <summary>stdoutRaw</summary>
                      <pre>{result.stdoutRaw || '<empty>'}</pre>
                    </details>
                    <details open={!result.ok && Boolean(result.stderrRaw)}>
                      <summary>stderrRaw</summary>
                      <pre>{result.stderrRaw || '<empty>'}</pre>
                    </details>
                  </>
                )}
              </div>
            </section>
          </div>
        </section>
      </main>

      {showEnvViewer && importedEnv && (
        <div className="env-viewer-backdrop" role="presentation" onClick={() => setShowEnvViewer(false)}>
          <section
            className="env-viewer"
            role="dialog"
            aria-modal="true"
            aria-labelledby="env-viewer-title"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="env-viewer-head">
              <div>
                <p className="section-kicker">Imported Env</p>
                <h2 id="env-viewer-title">All Imported Variables</h2>
                <p className="env-viewer-meta">{importedEnv.sourcePath ?? importedEnv.fileName}</p>
              </div>
              <button type="button" className="secondary-btn" onClick={() => setShowEnvViewer(false)}>
                Close
              </button>
            </div>
            <div className="env-viewer-body">
              {importedEnvEntries.map(([key, value]) => (
                <div key={key} className="env-viewer-row">
                  <code className="env-viewer-key">{key}</code>
                  <pre className="env-viewer-value">{value}</pre>
                </div>
              ))}
            </div>
          </section>
        </div>
      )}
    </div>
  )
}

function pathKey(path: string[]): string {
  return path.join('/')
}

function buildDefaultValues(command: CatalogCommand): Record<string, string | boolean> {
  return (command.fields ?? []).reduce<Record<string, string | boolean>>((acc, field) => {
    acc[field.name] = field.kind === 'bool' ? field.defaultValue === 'true' : field.defaultValue ?? ''
    return acc
  }, {})
}

function mergeCommandValues(catalog: CatalogRoot[], current: CommandValuesMap): CommandValuesMap {
  const next: CommandValuesMap = {}

  for (const root of catalog) {
    for (const command of root.commands ?? []) {
      const key = pathKey(command.path)
      const defaults = buildDefaultValues(command)
      next[key] = {
        ...defaults,
        ...(current[key] ?? {}),
      }
    }
  }

  return next
}

function withHealthDefaults(catalog: CatalogRoot[], current: CommandValuesMap, vmdockerDir?: string): CommandValuesMap {
  if (!vmdockerDir) {
    return current
  }

  const next = { ...current }
  for (const root of catalog) {
    for (const command of root.commands ?? []) {
      if (!command.path[0].startsWith('vmdocker')) {
        continue
      }
      const key = pathKey(command.path)
      const values = { ...(next[key] ?? {}) }
      if (values.dir === '' || values.dir === './vmdocker') {
        values.dir = vmdockerDir
      }
      next[key] = values
    }
  }
  return next
}

function applyEnvMappings(
  catalog: CatalogRoot[],
  current: CommandValuesMap,
  envValues: Record<string, string>,
  previousAppliedFields: EnvAppliedFieldsMap,
): { values: CommandValuesMap; appliedFields: EnvAppliedFieldsMap } {
  const next = mergeCommandValues(catalog, clearEnvMappedValues(catalog, current, previousAppliedFields))
  const appliedFields: EnvAppliedFieldsMap = {}

  for (const root of catalog) {
    for (const command of root.commands ?? []) {
      const key = pathKey(command.path)
      const values = { ...(next[key] ?? {}) }
      for (const field of command.fields ?? []) {
        const envValue = findFirstEnvValue(envValues, field.envKeys)
        if (!envValue) {
          continue
        }
        values[field.name] = field.kind === 'bool' ? envValue === 'true' : envValue
        appliedFields[key] = {
          ...(appliedFields[key] ?? {}),
          [field.name]: true,
        }
      }
      next[key] = values
    }
  }

  return { values: next, appliedFields }
}

function clearEnvMappedValues(catalog: CatalogRoot[], current: CommandValuesMap, appliedFields: EnvAppliedFieldsMap): CommandValuesMap {
  const commandIndex = indexCatalogCommands(catalog)
  const next: CommandValuesMap = { ...current }

  for (const [key, fields] of Object.entries(appliedFields)) {
    const command = commandIndex[key]
    if (!command) {
      continue
    }

    const defaults = buildDefaultValues(command)
    const values = {
      ...defaults,
      ...(next[key] ?? {}),
    }
    for (const fieldName of Object.keys(fields)) {
      values[fieldName] = defaults[fieldName] ?? ''
    }
    next[key] = values
  }

  return next
}

function indexCatalogCommands(catalog: CatalogRoot[]): Record<string, CatalogCommand> {
  return catalog.reduce<Record<string, CatalogCommand>>((acc, root) => {
    for (const command of root.commands ?? []) {
      acc[pathKey(command.path)] = command
    }
    return acc
  }, {})
}

function removeEnvAppliedField(appliedFields: EnvAppliedFieldsMap, key: string, fieldName: string): EnvAppliedFieldsMap {
  if (!appliedFields[key]?.[fieldName]) {
    return appliedFields
  }

  const next = { ...appliedFields }
  const nextFields = { ...(next[key] ?? {}) }
  delete nextFields[fieldName]

  if (Object.keys(nextFields).length === 0) {
    delete next[key]
    return next
  }

  next[key] = nextFields
  return next
}

function findFirstEnvValue(values: Record<string, string>, keys?: string[]): string {
  if (!keys?.length) {
    return ''
  }
  return firstEnvValue(values, keys)
}

function buildEnvStatus(values: Record<string, string>, source: string): string {
  return `Loaded ${Object.keys(values).length} keys from ${source}. ${
    firstEnvValue(values, ['VMDOCKER_PRIVATE_KEY', 'HYPE_PRIVATE_KEY', 'PRV_KEY']) ? 'Private key detected.' : 'Private key missing.'
  }`
}

function groupFields(command: CatalogCommand | null): Array<[string, CatalogField[]]> {
  if (!command) {
    return []
  }

  const grouped = new Map<string, CatalogField[]>()
  for (const field of command.fields ?? []) {
    const group = field.group || 'command'
    grouped.set(group, [...(grouped.get(group) ?? []), field])
  }

  return Array.from(grouped.entries()).sort(([left], [right]) => {
    if (left === right) {
      return 0
    }
    if (left === 'shared') {
      return -1
    }
    if (right === 'shared') {
      return 1
    }
    return left.localeCompare(right)
  })
}

function renderField(
  field: CatalogField,
  value: string | boolean | undefined,
  command: CatalogCommand | null,
  updateFieldValue: (command: CatalogCommand, field: CatalogField, value: string | boolean) => void,
) {
  if (!command) {
    return null
  }

  const hint = field.description ? `${field.description}${field.required ? ' Required.' : ''}` : field.required ? 'Required.' : ''

  if (field.kind === 'bool') {
    return (
      <label key={field.name} className="checkbox-field">
        <span>{field.label}</span>
        <input
          type="checkbox"
          checked={Boolean(value)}
          onChange={(e) => updateFieldValue(command, field, e.target.checked)}
        />
        {hint && <span className="field-hint">{hint}</span>}
      </label>
    )
  }

  if (field.kind === 'enum') {
    return (
      <label key={field.name}>
        {field.label}
        <select value={typeof value === 'string' ? value : ''} onChange={(e) => updateFieldValue(command, field, e.target.value)}>
          {(field.options ?? []).map((option) => (
            <option key={`${field.name}-${option.value || 'empty'}`} value={option.value}>
              {option.label}
            </option>
          ))}
        </select>
        {hint && <span className="field-hint">{hint}</span>}
      </label>
    )
  }

  if (field.kind === 'multiline') {
    return (
      <label key={field.name}>
        {field.label}
        <textarea value={typeof value === 'string' ? value : ''} onChange={(e) => updateFieldValue(command, field, e.target.value)} rows={5} />
        {hint && <span className="field-hint">{hint}</span>}
      </label>
    )
  }

  return (
    <label key={field.name}>
      {field.label}
      <input
        type={field.kind === 'secret' ? 'password' : field.kind === 'int' || field.kind === 'int64' ? 'number' : 'text'}
        value={typeof value === 'string' ? value : ''}
        onChange={(e) => updateFieldValue(command, field, e.target.value)}
        placeholder={field.defaultValue || ''}
      />
      {hint && <span className="field-hint">{hint}</span>}
    </label>
  )
}

function isMissingValue(value: string | boolean | undefined, kind: CatalogField['kind']): boolean {
  if (kind === 'bool') {
    return false
  }
  return typeof value !== 'string' || value.trim() === ''
}

function serializeValues(command: CatalogCommand, values: Record<string, string | boolean>): Record<string, string | boolean> {
  const serialized: Record<string, string | boolean> = {}
  for (const field of command.fields ?? []) {
    const value = values[field.name]
    if (typeof value === 'boolean') {
      serialized[field.name] = value
      continue
    }
    serialized[field.name] = value ?? ''
  }
  return serialized
}
