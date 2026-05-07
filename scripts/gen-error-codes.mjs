#!/usr/bin/env node
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const repoRoot = path.resolve(__dirname, '..')
const catalogPath = path.join(repoRoot, 'server/internal/errcode/catalog.go')
const outputPath = path.join(repoRoot, 'web/src/common/consts/errorCodes.generated.js')
const args = new Set(process.argv.slice(2))
const checkOnly = args.has('--check')
const printStdout = args.has('--stdout')

function toUpperSnake(name) {
  return name
    .replace(/([A-Z]+)([A-Z][a-z])/g, '$1_$2')
    .replace(/([a-z0-9])([A-Z])/g, '$1_$2')
    .toUpperCase()
}

function loadDefinitions(source) {
  const entryRegex = /^\s*([A-Za-z][A-Za-z0-9]*)\s*=\s*Definition\{\s*Name:\s*"[^"]+",\s*Code:\s*([0-9]+),\s*Message:\s*"(?:[^"\\]|\\.)*"\s*\}/gm
  const definitions = []
  let match
  while ((match = entryRegex.exec(source)) !== null) {
    definitions.push({
      ident: match[1],
      key: toUpperSnake(match[1]),
      code: Number(match[2]),
    })
  }
  return definitions
}

function render(definitions) {
  const lines = [
    '// 由 `node scripts/gen-error-codes.mjs` 自动生成；请勿手改。',
    '// 真源：`server/internal/errcode/catalog.go`。',
    'export const RpcErrorCode = Object.freeze({',
  ]

  for (const item of definitions) {
    lines.push(`  ${item.key}: ${item.code},`)
  }

  lines.push('})', '')
  return lines.join('\n')
}

if (!fs.existsSync(catalogPath)) {
  console.error(`[gen-error-codes] 未找到错误码目录：${catalogPath}`)
  process.exit(1)
}

const source = fs.readFileSync(catalogPath, 'utf8')
const definitions = loadDefinitions(source)
if (definitions.length === 0) {
  console.error('[gen-error-codes] 未从 catalog.go 解析到任何错误码定义')
  process.exit(1)
}

const rendered = render(definitions)
if (printStdout) {
  process.stdout.write(rendered)
  process.exit(0)
}

if (checkOnly) {
  if (!fs.existsSync(outputPath)) {
    console.error(`[gen-error-codes] 缺少生成文件：${path.relative(repoRoot, outputPath)}`)
    process.exit(1)
  }
  const current = fs.readFileSync(outputPath, 'utf8')
  if (current !== rendered) {
    console.error('[gen-error-codes] 前端错误码生成文件未同步，请执行：node scripts/gen-error-codes.mjs')
    process.exit(1)
  }
  console.log('[gen-error-codes] 通过')
  process.exit(0)
}

// 构建期只从服务端目录生成码表，消费侧分组/文案仍保留在手写文件里。
fs.writeFileSync(outputPath, rendered)
console.log(`[gen-error-codes] 已更新 ${path.relative(repoRoot, outputPath)}`)
