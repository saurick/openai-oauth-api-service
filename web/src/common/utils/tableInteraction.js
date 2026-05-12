const TABLE_ROW_INTERACTIVE_SELECTOR =
  'a,button,input,textarea,select,label,[role="button"]'

export const TABLE_ROW_INTERACTION_TITLE = '单击单选，复选框多选，双击编辑'

export function isInteractiveTableTarget(target) {
  if (!target || typeof target.closest !== 'function') {
    return false
  }
  return Boolean(target.closest(TABLE_ROW_INTERACTIVE_SELECTOR))
}

export function getTableSelectionAfterClick(selectedRowKeys, rowKey) {
  if (rowKey === undefined || rowKey === null) {
    return []
  }
  return [rowKey]
}

export function toggleTableSelection(selectedRowKeys, rowKey, checked) {
  if (rowKey === undefined || rowKey === null) {
    return Array.isArray(selectedRowKeys) ? selectedRowKeys : []
  }
  const current = Array.isArray(selectedRowKeys) ? selectedRowKeys : []
  if (checked) {
    if (current.some((key) => String(key) === String(rowKey))) {
      return current
    }
    return [...current, rowKey]
  }
  return current.filter((key) => String(key) !== String(rowKey))
}
