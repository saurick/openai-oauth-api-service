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
      '\nmodule.exports = { TABLE_ROW_INTERACTION_TITLE, isInteractiveTableTarget, getTableSelectionAfterClick, toggleTableSelection };\n'
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
  getTableSelectionAfterClick,
  isInteractiveTableTarget,
  toggleTableSelection,
} = loadTableInteractionModule()

function localArray(value) {
  return Array.from(value)
}

test('tableInteraction: 行点击保持互斥单选', () => {
  assert.deepEqual(localArray(getTableSelectionAfterClick([], 2)), [2])
  assert.deepEqual(localArray(getTableSelectionAfterClick([1], 2)), [2])
  assert.deepEqual(localArray(getTableSelectionAfterClick([1], null)), [])
  assert.deepEqual(localArray(getTableSelectionAfterClick([1, 2], 2)), [2])
})

test('tableInteraction: 勾选框按多选处理', () => {
  assert.deepEqual(localArray(toggleTableSelection([1], 2, true)), [1, 2])
  assert.deepEqual(localArray(toggleTableSelection([2], 2, false)), [])
  assert.deepEqual(localArray(toggleTableSelection([1, 2], 1, false)), [2])
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
  assert.equal(TABLE_ROW_INTERACTION_TITLE, '单击单选，复选框多选，双击编辑')
})
