import { useEffect, useMemo, useState } from 'react'
import type { ChangeEvent, FormEvent } from 'react'
import { fetchHealth, loadEnvFile, runCommand } from './api'
import { firstEnvValue, parseEnvContent } from './env'
import {
  chatSchema,
  confTGSchema,
  pairTGSchema,
  sharedSchema,
  spawnSchema,
  vmdockerGetSchema,
  vmdockerInitSchema,
} from './schemas'
import type {
  ChatForm,
  CommandResponse,
  ConfTGForm,
  HealthResponse,
  OpenclawCommandKey,
  PairTGForm,
  RootCommand,
  SharedForm,
  SpawnForm,
  VmdockerCommandKey,
  VmdockerGetForm,
  VmdockerInitForm,
} from './types'

const rootCommands: RootCommand[] = ['openclaw', 'vmdocker']
const openclawCommands: OpenclawCommandKey[] = ['spawn', 'conf-tg', 'pair-tg', 'chat']
const vmdockerCommands: VmdockerCommandKey[] = ['get', 'init']

const openclawCommandLabels: Record<OpenclawCommandKey, string> = {
  spawn: 'spawn',
  'conf-tg': 'conf-tg',
  'pair-tg': 'pair-tg',
  chat: 'chat',
}

const vmdockerCommandLabels: Record<VmdockerCommandKey, string> = {
  get: 'get',
  init: 'init',
}

const importedEnvPreviewKeys = [
  'VMDOCKER_PRIVATE_KEY',
  'OPENCLAW_MODULE_ID',
  'OPENCLAW_MODEL',
  'OPENCLAW_PROVIDER',
  'OPENCLAW_API_KEY',
  'OPENCLAW_GATEWAY_TOKEN',
  'OPENCLAW_TELEGRAM_BOT_TOKEN',
] as const

const initialShared: SharedForm = {
  nodeUrl: 'http://127.0.0.1:8080',
  privateKey: '',
}

const initialSpawn: SpawnForm = {
  moduleId: '',
  scheduler: '',
  model: '',
  provider: '',
  apiKey: '',
  gatewayToken: '',
  runtimeBackend: '',
  botToken: '',
  defaultAccount: 'main',
  dmPolicy: 'open',
  allowFrom: '*',
}

const initialConfTG: ConfTGForm = {
  pid: '',
  botToken: '',
  defaultAccount: 'main',
  dmPolicy: 'pairing',
  allowFrom: '*',
}

const initialPairTG: PairTGForm = {
  pid: '',
  code: '',
  channel: 'telegram',
  dmPolicy: 'pairing',
}

const initialChat: ChatForm = {
  pid: '',
  command: '',
}

const initialVmdockerGet: VmdockerGetForm = {
  dir: './vmdocker',
  version: '',
}

const initialVmdockerInit: VmdockerInitForm = {
  dir: './vmdocker',
}

type ImportedEnvState = {
  fileName: string
  sourcePath?: string
  content: string
  values: Record<string, string>
}

export function App() {
  const [rootCommand, setRootCommand] = useState<RootCommand>('openclaw')
  const [openclawCommand, setOpenclawCommand] = useState<OpenclawCommandKey>('spawn')
  const [vmdockerCommand, setVmdockerCommand] = useState<VmdockerCommandKey>('get')
  const [shared, setShared] = useState<SharedForm>(initialShared)
  const [spawn, setSpawn] = useState<SpawnForm>(initialSpawn)
  const [confTG, setConfTG] = useState<ConfTGForm>(initialConfTG)
  const [pairTG, setPairTG] = useState<PairTGForm>(initialPairTG)
  const [chat, setChat] = useState<ChatForm>(initialChat)
  const [vmdockerGet, setVmdockerGet] = useState<VmdockerGetForm>(initialVmdockerGet)
  const [vmdockerInit, setVmdockerInit] = useState<VmdockerInitForm>(initialVmdockerInit)
  const [importedEnv, setImportedEnv] = useState<ImportedEnvState | null>(null)
  const [envPath, setEnvPath] = useState('./vmdocker/.env')
  const [envStatus, setEnvStatus] = useState('')
  const [envError, setEnvError] = useState('')
  const [showEnvViewer, setShowEnvViewer] = useState(false)
  const [health, setHealth] = useState<HealthResponse | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [result, setResult] = useState<CommandResponse | null>(null)
  const [copyFeedback, setCopyFeedback] = useState('')

  useEffect(() => {
    fetchHealth()
      .then((data) => {
        setHealth(data)
        if (data.vmdockerDir) {
          setVmdockerGet((current) => ({
            ...current,
            dir: current.dir === './vmdocker' ? data.vmdockerDir : current.dir,
          }))
          setVmdockerInit((current) => ({
            ...current,
            dir: current.dir === './vmdocker' ? data.vmdockerDir : current.dir,
          }))
        }
      })
      .catch((err: Error) => setError(err.message))
  }, [])

  const payload = useMemo<Record<string, string>>(() => {
    if (rootCommand === 'openclaw') {
      if (openclawCommand === 'spawn') return { ...shared, ...spawn }
      if (openclawCommand === 'conf-tg') return { ...shared, ...confTG }
      if (openclawCommand === 'pair-tg') return { ...shared, ...pairTG }
      return { ...shared, ...chat }
    }
    if (vmdockerCommand === 'get') {
      return { ...vmdockerGet }
    }
    return {
      ...vmdockerInit,
      envFileName: importedEnv?.fileName ?? '',
      envFileContent: importedEnv?.content ?? '',
    }
  }, [rootCommand, openclawCommand, vmdockerCommand, shared, spawn, confTG, pairTG, chat, vmdockerGet, vmdockerInit, importedEnv])

  const activeCommand = rootCommand === 'openclaw' ? openclawCommand : vmdockerCommand
  const hasImportedEnv = Boolean(importedEnv?.content.trim())
  const hasImportedPrivateKey = Boolean(importedEnv?.values.VMDOCKER_PRIVATE_KEY)
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
  const runDisabled = loading || (rootCommand === 'vmdocker' && vmdockerCommand === 'init' && !hasImportedEnv)

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

  function applyImportedEnv(values: Record<string, string>) {
    const importedPrivateKey = firstEnvValue(values, ['HYPE_PRIVATE_KEY', 'PRV_KEY', 'VMDOCKER_PRIVATE_KEY'])
    if (importedPrivateKey) {
      setShared((current) => ({ ...current, privateKey: importedPrivateKey }))
    }

    setSpawn((current) => ({
      ...current,
      moduleId: values.OPENCLAW_MODULE_ID ?? current.moduleId,
      scheduler: values.VMDOCKER_SCHEDULER ?? current.scheduler,
      model: values.OPENCLAW_MODEL ?? current.model,
      provider: values.OPENCLAW_PROVIDER ?? current.provider,
      apiKey: values.OPENCLAW_API_KEY ?? current.apiKey,
      gatewayToken: values.OPENCLAW_GATEWAY_TOKEN ?? current.gatewayToken,
      botToken: values.OPENCLAW_TELEGRAM_BOT_TOKEN ?? current.botToken,
      defaultAccount: values.OPENCLAW_TELEGRAM_DEFAULT_ACCOUNT ?? current.defaultAccount,
      dmPolicy: values.OPENCLAW_TELEGRAM_DM_POLICY ?? current.dmPolicy,
      allowFrom: values.OPENCLAW_TELEGRAM_ALLOW_FROM ?? current.allowFrom,
    }))
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
      setImportedEnv({ fileName: file.name, content, values })
      setEnvStatus(
        `Loaded ${Object.keys(values).length} keys from ${file.name}. ${
          values.VMDOCKER_PRIVATE_KEY ? 'VMDOCKER_PRIVATE_KEY detected.' : 'VMDOCKER_PRIVATE_KEY missing.'
        }`,
      )
      applyImportedEnv(values)
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
      setImportedEnv({
        fileName: loaded.fileName,
        sourcePath: loaded.path,
        content: loaded.content,
        values,
      })
      setEnvStatus(
        `Loaded ${Object.keys(values).length} keys from ${loaded.path}. ${
          values.VMDOCKER_PRIVATE_KEY ? 'VMDOCKER_PRIVATE_KEY detected.' : 'VMDOCKER_PRIVATE_KEY missing.'
        }`,
      )
      applyImportedEnv(values)
    } catch (err) {
      setImportedEnv(null)
      setEnvError(err instanceof Error ? err.message : 'Failed to load env file path')
    }
  }

  function clearImportedEnv() {
    setImportedEnv(null)
    setEnvStatus('')
    setEnvError('')
    setShowEnvViewer(false)
  }

  async function onSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault()
    setError('')

    if (rootCommand === 'openclaw') {
      const sharedCheck = sharedSchema.safeParse(shared)
      if (!sharedCheck.success) {
        setError(sharedCheck.error.issues[0]?.message || 'shared fields invalid')
        return
      }

      let valid = true
      if (openclawCommand === 'spawn') valid = spawnSchema.safeParse(spawn).success
      if (openclawCommand === 'conf-tg') valid = confTGSchema.safeParse(confTG).success
      if (openclawCommand === 'pair-tg') valid = pairTGSchema.safeParse(pairTG).success
      if (openclawCommand === 'chat') valid = chatSchema.safeParse(chat).success
      if (!valid) {
        setError('current command form has invalid or missing fields')
        return
      }
    } else {
      const valid = vmdockerCommand === 'get' ? vmdockerGetSchema.safeParse(vmdockerGet).success : vmdockerInitSchema.safeParse(vmdockerInit).success
      if (!valid) {
        setError('current command form has invalid or missing fields')
        return
      }
      if (vmdockerCommand === 'init' && !hasImportedEnv) {
        setError('Import a .env file before running vmdocker init.')
        return
      }
    }

    setLoading(true)
    try {
      const data = await runCommand(rootCommand, activeCommand, payload)
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
          <h2>Load local runtime values before you run UI commands</h2>
          <p className="env-import-copy">Imported values stay in memory only and can prefill both Openclaw and VMDocker forms. Hidden files can be loaded by path.</p>
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
            <select value={rootCommand} onChange={(e) => setRootCommand(e.target.value as RootCommand)}>
              {rootCommands.map((cmd) => (
                <option key={cmd} value={cmd}>
                  {cmd}
                </option>
              ))}
            </select>
          </label>

          {rootCommand === 'openclaw' ? (
            openclawCommands.map((cmd) => {
              return (
                <button
                  key={cmd}
                  type="button"
                  className={cmd === openclawCommand ? 'nav-btn active' : 'nav-btn'}
                  onClick={() => setOpenclawCommand(cmd)}
                >
                  {openclawCommandLabels[cmd]}
                </button>
              )
            })
          ) : (
            vmdockerCommands.map((cmd) => (
              <button
                key={cmd}
                type="button"
                className={cmd === vmdockerCommand ? 'nav-btn active' : 'nav-btn'}
                onClick={() => setVmdockerCommand(cmd)}
              >
                {vmdockerCommandLabels[cmd]}
              </button>
            ))
          )}
        </aside>

        <section className="panel">
          <form onSubmit={onSubmit}>
            {rootCommand === 'openclaw' && (
              <fieldset>
                <legend>Shared</legend>
                <label>
                  Node URL
                  <input
                    value={shared.nodeUrl}
                    onChange={(e) => setShared({ ...shared, nodeUrl: e.target.value })}
                    placeholder="http://127.0.0.1:8080"
                  />
                </label>
                <label>
                  Private Key (optional if HYPE_PRIVATE_KEY/PRV_KEY/VMDOCKER_PRIVATE_KEY provided)
                  <input
                    value={shared.privateKey}
                    onChange={(e) => setShared({ ...shared, privateKey: e.target.value })}
                    placeholder="0x..."
                  />
                </label>
              </fieldset>
            )}

            {rootCommand === 'openclaw' && openclawCommand === 'spawn' && (
              <fieldset>
                <legend>spawn</legend>
                <label>Module ID<input value={spawn.moduleId} onChange={(e) => setSpawn({ ...spawn, moduleId: e.target.value })} /></label>
                <label>Scheduler<input value={spawn.scheduler} onChange={(e) => setSpawn({ ...spawn, scheduler: e.target.value })} /></label>
                <label>Model<input value={spawn.model} onChange={(e) => setSpawn({ ...spawn, model: e.target.value })} /></label>
                <label>Provider<input value={spawn.provider} onChange={(e) => setSpawn({ ...spawn, provider: e.target.value })} placeholder="zen" /></label>
                <label>API Key<input value={spawn.apiKey} onChange={(e) => setSpawn({ ...spawn, apiKey: e.target.value })} /></label>
                <label>Gateway Token<input value={spawn.gatewayToken} onChange={(e) => setSpawn({ ...spawn, gatewayToken: e.target.value })} /></label>
                <label>
                  Runtime Backend
                  <select value={spawn.runtimeBackend} onChange={(e) => setSpawn({ ...spawn, runtimeBackend: e.target.value })}>
                    <option value="">auto (let vmdocker choose)</option>
                    <option value="docker">docker</option>
                    <option value="sandbox">sandbox</option>
                  </select>
                </label>
                <label>Bot Token (optional auto conf-tg)<input value={spawn.botToken} onChange={(e) => setSpawn({ ...spawn, botToken: e.target.value })} /></label>
                <label>Default Account<input value={spawn.defaultAccount} onChange={(e) => setSpawn({ ...spawn, defaultAccount: e.target.value })} /></label>
                <label>DM Policy<input value={spawn.dmPolicy} onChange={(e) => setSpawn({ ...spawn, dmPolicy: e.target.value })} /></label>
                <label>Allow From<input value={spawn.allowFrom} onChange={(e) => setSpawn({ ...spawn, allowFrom: e.target.value })} /></label>
              </fieldset>
            )}

            {rootCommand === 'openclaw' && openclawCommand === 'conf-tg' && (
              <fieldset>
                <legend>conf-tg</legend>
                <label>PID<input value={confTG.pid} onChange={(e) => setConfTG({ ...confTG, pid: e.target.value })} /></label>
                <label>Bot Token<input value={confTG.botToken} onChange={(e) => setConfTG({ ...confTG, botToken: e.target.value })} /></label>
                <label>Default Account<input value={confTG.defaultAccount} onChange={(e) => setConfTG({ ...confTG, defaultAccount: e.target.value })} /></label>
                <label>DM Policy<input value={confTG.dmPolicy} onChange={(e) => setConfTG({ ...confTG, dmPolicy: e.target.value })} /></label>
                <label>Allow From<input value={confTG.allowFrom} onChange={(e) => setConfTG({ ...confTG, allowFrom: e.target.value })} placeholder="*" /></label>
              </fieldset>
            )}

            {rootCommand === 'openclaw' && openclawCommand === 'pair-tg' && (
              <fieldset>
                <legend>pair-tg</legend>
                <label>PID<input value={pairTG.pid} onChange={(e) => setPairTG({ ...pairTG, pid: e.target.value })} /></label>
                <label>Code<input value={pairTG.code} onChange={(e) => setPairTG({ ...pairTG, code: e.target.value })} /></label>
                <label>Channel<input value={pairTG.channel} onChange={(e) => setPairTG({ ...pairTG, channel: e.target.value })} /></label>
                <label>DM Policy<input value={pairTG.dmPolicy} onChange={(e) => setPairTG({ ...pairTG, dmPolicy: e.target.value })} /></label>
              </fieldset>
            )}

            {rootCommand === 'openclaw' && openclawCommand === 'chat' && (
              <fieldset>
                <legend>chat</legend>
                <label>PID<input value={chat.pid} onChange={(e) => setChat({ ...chat, pid: e.target.value })} /></label>
                <label>Command<textarea value={chat.command} onChange={(e) => setChat({ ...chat, command: e.target.value })} rows={4} /></label>
              </fieldset>
            )}

            {rootCommand === 'vmdocker' && vmdockerCommand === 'get' && (
              <fieldset>
                <legend>vmdocker get</legend>
                <label>
                  Checkout Directory
                  <input value={vmdockerGet.dir} onChange={(e) => setVmdockerGet({ ...vmdockerGet, dir: e.target.value })} />
                </label>
                <label>
                  Version (optional)
                  <input
                    value={vmdockerGet.version}
                    onChange={(e) => setVmdockerGet({ ...vmdockerGet, version: e.target.value })}
                    placeholder="latest semver if empty"
                  />
                </label>
              </fieldset>
            )}

            {rootCommand === 'vmdocker' && vmdockerCommand === 'init' && (
              <fieldset>
                <legend>vmdocker init</legend>
                <label>
                  Checkout Directory
                  <input value={vmdockerInit.dir} onChange={(e) => setVmdockerInit({ ...vmdockerInit, dir: e.target.value })} />
                </label>
                <p className="field-hint">
                  This command uses the imported <code>.env</code> content. Import is required before execution.
                </p>
              </fieldset>
            )}

            <button disabled={runDisabled} className="run-btn" type="submit">
              {loading ? 'Running...' : `Run ${rootCommand} ${activeCommand}`}
            </button>
          </form>

          <div className="result-column">
            {rootCommand === 'openclaw' && openclawCommand === 'spawn' && (
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

            {rootCommand === 'vmdocker' && (
              <section className="env-summary" aria-live="polite">
                <div className="result-head">
                  <h2>Imported .env</h2>
                  {importedEnv && <span className={hasImportedPrivateKey ? 'env-pill ok' : 'env-pill warn'}>{hasImportedPrivateKey ? 'Ready for init' : 'Missing private key'}</span>}
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
                      <p>No Openclaw/VMDocker keys detected yet.</p>
                    )}
                  </>
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
