import assert from 'node:assert/strict'
import test from 'node:test'
import fs from 'node:fs'
import path from 'node:path'
import vm from 'node:vm'

function loadTableInteractionModule() {
  const filePath = path.resolve(import.meta.dirname, './tableInteraction.js')
  const source = fs.readFileSync(filePath, 'utf8')
  const transformed = source
    .replace(/export function /g, 'function ')
    .replace(/export const /g, 'const ')
    .concat(
      '\nmodule.exports = { TABLE_ROW_INTERACTION_TITLE, isInteractiveTableTarget, getExclusiveTableSelectionAfterClick, toggleExclusiveTableSelection };\n'
    )

  const sandbox = {
    module: { exports: {} },
    exports: {},
  }
  vm.runInNewContext(transformed, sandbox, { filename: filePath })
  return sandbox.module.exports
}

const {
  TABLE_ROW_INTERACTION_TITLE,
  getExclusiveTableSelectionAfterClick,
  isInteractiveTableTarget,
  toggleExclusiveTableSelection,
} = loadTableInteractionModule()

function localArray(value) {
  return Array.from(value)
}

test('tableInteraction: 行点击只保留当前选择', () => {
  assert.deepEqual(localArray(getExclusiveTableSelectionAfterClick(2)), [2])
  assert.deepEqual(localArray(getExclusiveTableSelectionAfterClick(null)), [])
})

test('tableInteraction: 勾选框按互斥选择处理', () => {
  assert.deepEqual(
    localArray(toggleExclusiveTableSelection([1], 2, true)),
    [2]
  )
  assert.deepEqual(
    localArray(toggleExclusiveTableSelection([2], 2, false)),
    []
  )
  assert.deepEqual(
    localArray(toggleExclusiveTableSelection([1, 2], 1, false)),
    [2]
  )
})

test('tableInteraction: 行内操作控件不触发行选择', () => {
  const button = {
    closest(selector) {
      return selector.includes('button') ? this : null
    },
  }
  const cellText = {
    closest() {
      return null
    },
  }

  assert.equal(isInteractiveTableTarget(button), true)
  assert.equal(isInteractiveTableTarget(cellText), false)
})

test('tableInteraction: 保留表格行交互提示文案', () => {
  assert.equal(TABLE_ROW_INTERACTION_TITLE, '单击单选，双击编辑')
})
