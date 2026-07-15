import assert from 'node:assert/strict'
import path from 'node:path'
import test from 'node:test'

import { loadDevPorts } from './dev-ports.mjs'

const repoRoot = path.resolve(import.meta.dirname, '..')

test('tracked manifest defines the OAuth local development bundle', () => {
  const ports = loadDevPorts(repoRoot, {})
  assert.deepEqual(ports, {
    projectId: 'openai-oauth-api-service',
    web: 5176,
    http: 8400,
    grpc: 9400,
    style: 6176,
    auxStart: 15300,
  })
})

test('manifest rejects overlapping ports inside one project bundle', () => {
  assert.throws(
    () => loadDevPorts(repoRoot, { DEV_HTTP_PORT: '5176' }),
    /DEV_HTTP_PORT.*overlaps DEV_WEB_PORT/u
  )
  assert.throws(
    () => loadDevPorts(repoRoot, { DEV_WEB_PORT: '15350' }),
    /DEV_AUX_PORT_START.*overlaps DEV_WEB_PORT/u
  )
})
