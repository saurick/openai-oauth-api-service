const TABLE_ROW_INTERACTIVE_SELECTOR =
  'a,button,input,textarea,select,label,[role="button"]'

export const TABLE_ROW_INTERACTION_TITLE = '单击单选，双击编辑'

export function isInteractiveTableTarget(target) {
  if (!target || typeof target.closest !== 'function') {
    return false
  }
  return Boolean(target.closest(TABLE_ROW_INTERACTIVE_SELECTOR))
}

export function getExclusiveTableSelectionAfterClick(rowKey) {
  if (rowKey === undefined || rowKey === null) {
    return []
  }
  return [rowKey]
}

export function toggleExclusiveTableSelection(selectedRowKeys, rowKey, checked) {
  if (rowKey === undefined || rowKey === null) {
    return Array.isArray(selectedRowKeys) ? selectedRowKeys : []
  }
  if (checked) {
    return [rowKey]
  }
  return (Array.isArray(selectedRowKeys) ? selectedRowKeys : []).filter(
    (key) => String(key) !== String(rowKey)
  )
}
