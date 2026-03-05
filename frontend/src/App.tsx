import { useEffect, useMemo, useState } from 'react'
import { fetchHealth, runCommand } from './api'
import { chatSchema, confTGSchema, pairTGSchema, sharedSchema, spawnSchema } from './schemas'
import type {
  ChatForm,
  CommandKey,
  ConfTGForm,
  HealthResponse,
  OpenclawResponse,
  PairTGForm,
  SharedForm,
  SpawnForm,
} from './types'

const rootCommands = ['openclaw', 'vmm', 'run', 'new', 'mount', 'module', 'get', 'repl', 'db-import', 'db-export'] as const
type RootCommand = (typeof rootCommands)[number]

const commandLabels: Record<CommandKey, string> = {
  spawn: 'spawn',
  'conf-tg': 'conf-tg',
  'pair-tg': 'pair-tg',
  chat: 'chat',
}

const initialShared: SharedForm = {
  nodeUrl: 'http://127.0.0.1:8080',
  privateKey: '',
}

const initialSpawn: SpawnForm = {
  moduleId: '',
  scheduler: '',
  model: '',
  timeoutMs: '180000',
  apiKey: '',
  gatewayToken: '',
}

const initialConfTG: ConfTGForm = {
  pid: '',
  botToken: '',
  defaultAccount: 'main',
  dmPolicy: 'pairing',
  allowFrom: '',
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

export function App() {
  const [rootCommand, setRootCommand] = useState<RootCommand>('openclaw')
  const [active, setActive] = useState<CommandKey>('spawn')
  const [shared, setShared] = useState<SharedForm>(initialShared)
  const [spawn, setSpawn] = useState<SpawnForm>(initialSpawn)
  const [confTG, setConfTG] = useState<ConfTGForm>(initialConfTG)
  const [pairTG, setPairTG] = useState<PairTGForm>(initialPairTG)
  const [chat, setChat] = useState<ChatForm>(initialChat)
  const [health, setHealth] = useState<HealthResponse | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [result, setResult] = useState<OpenclawResponse | null>(null)
  const [copyFeedback, setCopyFeedback] = useState('')

  useEffect(() => {
    fetchHealth().then(setHealth).catch((err: Error) => setError(err.message))
  }, [])

  const payload = useMemo(() => {
    if (active === 'spawn') return { ...shared, ...spawn }
    if (active === 'conf-tg') return { ...shared, ...confTG }
    if (active === 'pair-tg') return { ...shared, ...pairTG }
    return { ...shared, ...chat }
  }, [active, shared, spawn, confTG, pairTG, chat])

  const openclawAvailable = rootCommand === 'openclaw'
  const spawnedPids = result?.spawnedPids ?? []

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

  async function onSubmit(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault()
    if (!openclawAvailable) {
      setError(`"${rootCommand}" is not supported yet. Please switch back to "openclaw".`)
      return
    }
    setError('')

    const sharedCheck = sharedSchema.safeParse(shared)
    if (!sharedCheck.success) {
      setError(sharedCheck.error.issues[0]?.message || 'shared fields invalid')
      return
    }

    let valid = true
    if (active === 'spawn') valid = spawnSchema.safeParse(spawn).success
    if (active === 'conf-tg') valid = confTGSchema.safeParse(confTG).success
    if (active === 'pair-tg') valid = pairTGSchema.safeParse(pairTG).success
    if (active === 'chat') valid = chatSchema.safeParse(chat).success
    if (!valid) {
      setError('current command form has invalid or missing fields')
      return
    }

    setLoading(true)
    try {
      const data = await runCommand(active, payload)
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

          {openclawAvailable ? (
            Object.keys(commandLabels).map((key) => {
              const cmd = key as CommandKey
              return (
                <button
                  key={cmd}
                  type="button"
                  className={cmd === active ? 'nav-btn active' : 'nav-btn'}
                  onClick={() => setActive(cmd)}
                >
                  {commandLabels[cmd]}
                </button>
              )
            })
          ) : (
            <p className="unsupported-msg">
              Current scope only supports <code>openclaw</code>. Select <code>openclaw</code> to run commands.
            </p>
          )}
        </aside>

        <section className="panel">
          <form onSubmit={onSubmit}>
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
                Private Key (optional if HYPE_PRIVATE_KEY/PRV_KEY provided)
                <input
                  value={shared.privateKey}
                  onChange={(e) => setShared({ ...shared, privateKey: e.target.value })}
                  placeholder="0x..."
                />
              </label>
            </fieldset>

            {openclawAvailable && active === 'spawn' && (
              <fieldset>
                <legend>spawn</legend>
                <label>Module ID<input value={spawn.moduleId} onChange={(e) => setSpawn({ ...spawn, moduleId: e.target.value })} /></label>
                <label>Scheduler<input value={spawn.scheduler} onChange={(e) => setSpawn({ ...spawn, scheduler: e.target.value })} /></label>
                <label>Model<input value={spawn.model} onChange={(e) => setSpawn({ ...spawn, model: e.target.value })} /></label>
                <label>Timeout (ms)<input value={spawn.timeoutMs} onChange={(e) => setSpawn({ ...spawn, timeoutMs: e.target.value })} /></label>
                <label>API Key<input value={spawn.apiKey} onChange={(e) => setSpawn({ ...spawn, apiKey: e.target.value })} /></label>
                <label>Gateway Token<input value={spawn.gatewayToken} onChange={(e) => setSpawn({ ...spawn, gatewayToken: e.target.value })} /></label>
              </fieldset>
            )}

            {openclawAvailable && active === 'conf-tg' && (
              <fieldset>
                <legend>conf-tg</legend>
                <label>PID<input value={confTG.pid} onChange={(e) => setConfTG({ ...confTG, pid: e.target.value })} /></label>
                <label>Bot Token<input value={confTG.botToken} onChange={(e) => setConfTG({ ...confTG, botToken: e.target.value })} /></label>
                <label>Default Account<input value={confTG.defaultAccount} onChange={(e) => setConfTG({ ...confTG, defaultAccount: e.target.value })} /></label>
                <label>DM Policy<input value={confTG.dmPolicy} onChange={(e) => setConfTG({ ...confTG, dmPolicy: e.target.value })} /></label>
                <label>Allow From (optional)<input value={confTG.allowFrom} onChange={(e) => setConfTG({ ...confTG, allowFrom: e.target.value })} /></label>
              </fieldset>
            )}

            {openclawAvailable && active === 'pair-tg' && (
              <fieldset>
                <legend>pair-tg</legend>
                <label>PID<input value={pairTG.pid} onChange={(e) => setPairTG({ ...pairTG, pid: e.target.value })} /></label>
                <label>Code<input value={pairTG.code} onChange={(e) => setPairTG({ ...pairTG, code: e.target.value })} /></label>
                <label>Channel<input value={pairTG.channel} onChange={(e) => setPairTG({ ...pairTG, channel: e.target.value })} /></label>
                <label>DM Policy<input value={pairTG.dmPolicy} onChange={(e) => setPairTG({ ...pairTG, dmPolicy: e.target.value })} /></label>
              </fieldset>
            )}

            {openclawAvailable && active === 'chat' && (
              <fieldset>
                <legend>chat</legend>
                <label>PID<input value={chat.pid} onChange={(e) => setChat({ ...chat, pid: e.target.value })} /></label>
                <label>Command<textarea value={chat.command} onChange={(e) => setChat({ ...chat, command: e.target.value })} rows={4} /></label>
              </fieldset>
            )}

            <button disabled={loading || !openclawAvailable} className="run-btn" type="submit">
              {loading ? 'Running...' : `Run ${active}`}
            </button>
          </form>

          <div className="result-column">
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
                    <details>
                      <summary>stdoutRaw</summary>
                      <pre>{result.stdoutRaw || '<empty>'}</pre>
                    </details>
                    <details>
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
    </div>
  )
}
