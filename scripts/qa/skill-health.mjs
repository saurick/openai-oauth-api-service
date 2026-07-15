#!/usr/bin/env node

import fs from 'node:fs'
import path from 'node:path'
import process from 'node:process'
import { fileURLToPath } from 'node:url'

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../..')
const skillsRoot = path.join(root, '.agents', 'skills')
const readmePath = path.join(skillsRoot, 'README.md')
const errors = []
const namePattern = /^[a-z0-9]+(?:-[a-z0-9]+)*$/u

function parseScalar(raw, source) {
  const value = raw.trim()
  if (!value) throw new Error(`${source}: empty YAML value`)
  if (value.startsWith('"')) return JSON.parse(value)
  if (value.startsWith("'") && value.endsWith("'")) return value.slice(1, -1)
  return value
}

function parseFrontmatter(content, source) {
  const match = content.match(/^---\r?\n([\s\S]*?)\r?\n---(?:\r?\n|$)/u)
  if (!match) throw new Error(`${source}: missing frontmatter`)
  const values = {}
  for (const line of match[1].split(/\r?\n/u)) {
    if (!line.trim()) continue
    const pair = line.match(/^([a-z_]+):\s*(.+)$/u)
    if (!pair) throw new Error(`${source}: unsupported frontmatter line`)
    if (Object.hasOwn(values, pair[1])) throw new Error(`${source}: duplicate frontmatter key ${pair[1]}`)
    values[pair[1]] = parseScalar(pair[2], source)
  }
  if (Object.keys(values).sort().join(',') !== 'description,name') {
    throw new Error(`${source}: frontmatter must contain only name and description`)
  }
  return values
}

function parseMetadata(content, source) {
  const lines = content.split(/\r?\n/u).filter((line) => line.trim())
  if (lines.shift()?.trim() !== 'interface:') throw new Error(`${source}: expected interface top-level key`)
  const values = {}
  const allowed = new Set(['display_name', 'short_description', 'default_prompt', 'icon_small', 'icon_large', 'brand_color'])
  for (const line of lines) {
    const pair = line.match(/^  ([a-z_]+):\s*(.+)$/u)
    if (!pair || !allowed.has(pair[1])) throw new Error(`${source}: unsupported interface line`)
    if (Object.hasOwn(values, pair[1])) throw new Error(`${source}: duplicate interface key ${pair[1]}`)
    values[pair[1]] = parseScalar(pair[2], source)
  }
  for (const key of ['display_name', 'short_description', 'default_prompt']) {
    if (!values[key]) throw new Error(`${source}: missing ${key}`)
  }
  return values
}

function validateLinks(content, skillDir, source) {
  for (const match of content.matchAll(/\[[^\]]*\]\(([^)]+)\)/gu)) {
    const target = match[1].trim().split('#', 1)[0]
    if (!target || target.startsWith('/') || /^[a-z][a-z0-9+.-]*:/iu.test(target) || /[<>]/u.test(target)) continue
    if (!fs.existsSync(path.resolve(skillDir, decodeURIComponent(target)))) errors.push(`${source}: missing relative link ${target}`)
  }
}

const skillNames = fs.readdirSync(skillsRoot, { withFileTypes: true }).filter((entry) => entry.isDirectory()).map((entry) => entry.name).sort()
for (const skillName of skillNames) {
  const skillDir = path.join(skillsRoot, skillName)
  const skillPath = path.join(skillDir, 'SKILL.md')
  const metadataPath = path.join(skillDir, 'agents', 'openai.yaml')
  try {
    if (!namePattern.test(skillName) || skillName.length > 64) throw new Error(`${skillDir}: invalid directory name`)
    const skillContent = fs.readFileSync(skillPath, 'utf8')
    const frontmatter = parseFrontmatter(skillContent, skillPath)
    if (frontmatter.name !== skillName) errors.push(`${skillPath}: name does not match directory`)
    if (frontmatter.description.length > 1024 || /[<>]/u.test(frontmatter.description)) errors.push(`${skillPath}: invalid description`)
    const metadata = parseMetadata(fs.readFileSync(metadataPath, 'utf8'), metadataPath)
    const shortLength = Array.from(metadata.short_description).length
    if (shortLength < 25 || shortLength > 64) errors.push(`${metadataPath}: short_description length ${shortLength}, expected 25-64`)
    if (!/^[\x20-\x7e]+$/u.test(metadata.display_name)) errors.push(`${metadataPath}: display_name must remain English/ASCII`)
    if (!metadata.default_prompt.includes(`$${skillName}`)) errors.push(`${metadataPath}: default_prompt must mention $${skillName}`)
    validateLinks(skillContent, skillDir, skillPath)
  } catch (error) {
    errors.push(error.message)
  }
}

let commonPrefix = skillNames[0] || ''
for (const skillName of skillNames.slice(1)) {
  let index = 0
  while (index < commonPrefix.length && commonPrefix[index] === skillName[index]) index += 1
  commonPrefix = commonPrefix.slice(0, index)
}
const projectPrefix = commonPrefix.slice(0, commonPrefix.lastIndexOf('-') + 1)
const listed = [...fs.readFileSync(readmePath, 'utf8').matchAll(/\$([a-z0-9]+(?:-[a-z0-9]+)+)/gu)]
  .map((match) => match[1])
  .filter((name) => name.startsWith(projectPrefix))
for (const skillName of skillNames) {
  const count = listed.filter((name) => name === skillName).length
  if (count !== 1) errors.push(`${readmePath}: expected one $${skillName}, got ${count}`)
}
for (const listedName of new Set(listed)) {
  if (!skillNames.includes(listedName)) errors.push(`${readmePath}: stale $${listedName}`)
}

if (errors.length > 0) {
  for (const error of errors) console.error(`[skill-health] ${error}`)
  process.exitCode = 1
} else {
  console.log(`[skill-health] status=ok skills=${skillNames.length}`)
}
