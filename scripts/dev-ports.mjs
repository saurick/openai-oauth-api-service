import { existsSync, readFileSync } from 'node:fs'
import path from 'node:path'
import process from 'node:process'
import { pathToFileURL } from 'node:url'

const portKeys = Object.freeze([
  'DEV_WEB_PORT',
  'DEV_HTTP_PORT',
  'DEV_GRPC_PORT',
  'DEV_STYLE_PORT',
  'DEV_AUX_PORT_START',
])
const auxPortRangeSize = 100

function parseEnvFile(filePath) {
  const values = {}
  for (const [index, rawLine] of readFileSync(filePath, 'utf8')
    .split(/\r?\n/u)
    .entries()) {
    const line = rawLine.trim()
    if (!line || line.startsWith('#')) continue
    const match = line.match(/^([A-Z][A-Z0-9_]*)=(.*)$/u)
    if (!match) throw new Error(`${filePath}:${index + 1}: expected KEY=value`)
    if (Object.hasOwn(values, match[1])) {
      throw new Error(`${filePath}:${index + 1}: duplicate key ${match[1]}`)
    }
    values[match[1]] = match[2].trim()
  }
  return values
}

function parsePort(key, value, sourceLabel) {
  if (!/^\d+$/u.test(String(value || ''))) {
    throw new Error(`${sourceLabel}: ${key} must be an integer port`)
  }
  const port = Number(value)
  if (!Number.isSafeInteger(port) || port < 1024 || port > 65535) {
    throw new Error(`${sourceLabel}: ${key} must be between 1024 and 65535`)
  }
  return port
}

export function loadDevPorts(projectRoot, env = process.env) {
  const manifestPath = path.join(projectRoot, 'config', 'dev-ports.env')
  const localPath = path.join(projectRoot, 'config', 'dev-ports.local.env')
  if (!existsSync(manifestPath)) {
    throw new Error(`development port manifest is missing: ${manifestPath}`)
  }
  const base = parseEnvFile(manifestPath)
  const local = existsSync(localPath) ? parseEnvFile(localPath) : {}
  if (Object.keys(local).length > 0) {
    const missing = portKeys.filter((key) => !Object.hasOwn(local, key))
    if (missing.length > 0) {
      throw new Error(
        `${localPath}: local override must contain the complete port bundle; missing ${missing.join(', ')}`
      )
    }
  }
  const merged = { ...base, ...local }
  for (const key of ['DEV_PROJECT_ID', ...portKeys]) {
    const override = String(env[key] || '').trim()
    if (override) merged[key] = override
  }
  const projectId = String(merged.DEV_PROJECT_ID || '').trim()
  if (!/^[a-z0-9][a-z0-9-]*$/u.test(projectId)) {
    throw new Error(`${manifestPath}: invalid DEV_PROJECT_ID`)
  }
  const parsed = Object.fromEntries(
    portKeys.map((key) => [key, parsePort(key, merged[key], manifestPath)])
  )
  const reservations = []
  for (const [key, port] of Object.entries(parsed)) {
    const end =
      key === 'DEV_AUX_PORT_START' ? port + auxPortRangeSize - 1 : port
    if (end > 65535) {
      throw new Error(
        `${manifestPath}: ${key} must reserve a complete ${auxPortRangeSize}-port range`
      )
    }
    const current = { key, start: port, end }
    const previous = reservations.find(
      (reservation) =>
        current.start <= reservation.end && reservation.start <= current.end
    )
    if (previous) {
      throw new Error(
        `${manifestPath}: ${key} range ${current.start}-${current.end} overlaps ${previous.key} range ${previous.start}-${previous.end}`
      )
    }
    reservations.push(current)
  }
  return Object.freeze({
    projectId,
    web: parsed.DEV_WEB_PORT,
    http: parsed.DEV_HTTP_PORT,
    grpc: parsed.DEV_GRPC_PORT,
    style: parsed.DEV_STYLE_PORT,
    auxStart: parsed.DEV_AUX_PORT_START,
  })
}

const isDirectRun =
  process.argv[1] &&
  pathToFileURL(path.resolve(process.argv[1])).href === import.meta.url

if (isDirectRun) {
  try {
    let projectRoot = path.resolve(import.meta.dirname, '..')
    const args = process.argv.slice(2)
    for (let index = 0; index < args.length; index += 1) {
      if (args[index] === '--check') continue
      if (args[index] === '--project-root') {
        projectRoot = path.resolve(args[index + 1] || '')
        index += 1
        continue
      }
      throw new Error(`unknown argument: ${args[index]}`)
    }
    const ports = loadDevPorts(projectRoot)
    process.stdout.write(
      `[dev-ports] ${ports.projectId}: web=${ports.web} http=${ports.http} grpc=${ports.grpc} style=${ports.style} aux-start=${ports.auxStart}\n`
    )
  } catch (error) {
    process.stderr.write(`[dev-ports] ${error.message}\n`)
    process.exit(1)
  }
}
