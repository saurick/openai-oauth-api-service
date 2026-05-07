/**
 * 十进制转十六进制
 * @param {number} numberT 十进制
 * @returns {string} 十六进制
 */
export const int2hex = (numberT) => {
  let numNew = numberT
  if (typeof numberT === 'string') {
    numNew = Number(numberT)
  }
  let reuslt = numNew.toString(16)
  if (reuslt.length % 2 !== 0) {
    reuslt = `0${reuslt}`
  }
  return numNew < 0 ? `-${reuslt}` : reuslt
}

/**
 * 十六进制转十进制
 * @param {string} hex 十六进制
 * @returns {number} 十进制
 */
export const hexToDecimal = (hex) => parseInt(hex, 16)

/**
 * 睡眠
 * @param {number} milliseconds 毫秒
 * @returns {Promise} 睡眠
 */
export const sleep = async (milliseconds) => {
  await new Promise((resolve) => {
    return setTimeout(resolve, milliseconds)
  })
}

/**
 * 查找父节点
 * @param {Object} node 节点
 * @param {String} className 类名
 * @returns {Object} 父节点
 */
export const findParentNodeByClassName = (node, className) => {
  if (node.className !== className) {
    return findParentNodeByClassName(node?.parentNode, className)
  }
  if (node.className === className) {
    return node
  }
}

/**
 * 判断是否为undefined
 * @param {any} str 值
 * @param {boolean} emptyStringCheck 是否检查空字符串
 * @returns {boolean} 是否为undefined
 */
export const isUndefined = (str, emptyStringCheck) => {
  if (
    typeof str === 'undefined' ||
    str === null ||
    str === 'undefined' ||
    str === 'null'
  ) {
    return true
  }
  if (
    emptyStringCheck &&
    typeof str === 'string' &&
    str.toString().trim().length === 0
  ) {
    return true
  }
  return false
}

/**
 * 判断是否为数组
 * @param {any} val 值
 * @returns {boolean} 是否为数组
 */
export const isTypeArray = (val) => {
  return Object.prototype.toString.call(val) === '[object Array]'
}

/**
 * 判断是否为字符串
 * @param {any} val 值
 * @returns {boolean} 是否为字符串
 */
export const isTypeString = (val) => {
  return Object.prototype.toString.call(val) === '[object String]'
}

/**
 * 获取对象的值
 * @param {Object} obj 对象
 * @param {String} key 键
 * @returns {any} 值
 */
export const getValForKey = (obj, key) => {
  if (!isUndefined(key)) {
    if (isTypeString(key)) {
      const keyArray = key.split('.')

      if (keyArray.length === 1) {
        return obj[key]
      }
      let finalValue = obj
      let i
      let l
      for (i = 0, l = keyArray.length; i < l; i += 1) {
        const currKey = keyArray[i]
        const currValue = finalValue[currKey]

        if (!isUndefined(currValue)) {
          finalValue = currValue
        } else {
          finalValue = undefined
          break
        }
      }

      return finalValue
    }
    return obj[key]
  }
}
