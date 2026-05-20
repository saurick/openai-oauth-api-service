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

export function toggleTablePageSelection(selectedRowKeys, rowKeys, checked) {
  const current = Array.isArray(selectedRowKeys) ? selectedRowKeys : []
  const pageKeys = Array.isArray(rowKeys)
    ? rowKeys.filter((key) => key !== undefined && key !== null)
    : []
  if (pageKeys.length === 0) {
    return current
  }
  const pageKeySet = new Set(pageKeys.map((key) => String(key)))
  if (!checked) {
    return current.filter((key) => !pageKeySet.has(String(key)))
  }
  const currentKeySet = new Set(current.map((key) => String(key)))
  const next = [...current]
  for (const key of pageKeys) {
    if (!currentKeySet.has(String(key))) {
      next.push(key)
      currentKeySet.add(String(key))
    }
  }
  return next
}
