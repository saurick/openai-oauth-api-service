import assert from 'node:assert/strict'
import test from 'node:test'
import fs from 'node:fs'
import path from 'node:path'
import vm from 'node:vm'

function loadErrorCodesModule(extraExports = []) {
  const generatedPath = path.resolve(
    import.meta.dirname,
    './errorCodes.generated.js'
  )
  const generatedSource = fs.readFileSync(generatedPath, 'utf8')
  const generatedTransformed = generatedSource
    .replace(/export const /g, 'const ')
    .concat('\nmodule.exports = { RpcErrorCode };\n')

  const generatedSandbox = { module: { exports: {} }, exports: {} }
  vm.runInNewContext(generatedTransformed, generatedSandbox, {
    filename: generatedPath,
  })

  const filePath = path.resolve(import.meta.dirname, './errorCodes.js')
  const source = fs.readFileSync(filePath, 'utf8')
  const extraNamedExports =
    extraExports.length > 0 ? `, ${extraExports.join(', ')}` : ''
  const transformed = source
    .replace(
      /import\s+\{\s*RpcErrorCode\s*\}\s+from\s+["']\.\/errorCodes\.generated\.js["']\s*/u,
      'const { RpcErrorCode } = __generated__\n'
    )
    .replace(/export\s+\{\s*RpcErrorCode\s*\}\s*/u, '')
    .replace(/export const /g, 'const ')
    .replace(/export function /g, 'function ')
    .concat(
      `\nmodule.exports = { RpcErrorCode, AUTH_FAILURE_ERROR_CODES, isAuthFailureCode${extraNamedExports}, DEFAULT_RPC_ERROR_MESSAGES };\n`
    )

  const sandbox = {
    module: { exports: {} },
    exports: {},
    __generated__: generatedSandbox.module.exports,
  }
  vm.runInNewContext(transformed, sandbox, { filename: filePath })
  return sandbox.module.exports
}

const {
  RpcErrorCode,
  AUTH_FAILURE_ERROR_CODES,
  isAuthFailureCode,
  DEFAULT_RPC_ERROR_MESSAGES,
} = loadErrorCodesModule()

test('errorCodes: 所有错误码保持唯一', () => {
  const values = Object.values(RpcErrorCode)
  assert.equal(new Set(values).size, values.length)
})

test('errorCodes: 仅登录态失效错误触发重新登录', () => {
  assert.deepEqual(
    [...AUTH_FAILURE_ERROR_CODES],
    [
      RpcErrorCode.AUTH_EXPIRED,
      RpcErrorCode.AUTH_INVALID,
      RpcErrorCode.AUTH_REQUIRED,
    ]
  )
  assert.equal(isAuthFailureCode(RpcErrorCode.AUTH_REQUIRED), true)
  assert.equal(isAuthFailureCode(RpcErrorCode.AUTH_EXPIRED), true)
  assert.equal(isAuthFailureCode(RpcErrorCode.AUTH_INVALID), true)
  assert.equal(isAuthFailureCode(RpcErrorCode.PERMISSION_DENIED), false)
  assert.equal(isAuthFailureCode(RpcErrorCode.ADMIN_DISABLED), false)
})

test('errorCodes: 默认文案覆盖核心鉴权错误', () => {
  assert.equal(
    DEFAULT_RPC_ERROR_MESSAGES[RpcErrorCode.AUTH_REQUIRED],
    '请先登录'
  )
  assert.equal(
    DEFAULT_RPC_ERROR_MESSAGES[RpcErrorCode.PERMISSION_DENIED],
    '权限不足'
  )
})
