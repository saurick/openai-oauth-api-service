import assert from 'node:assert/strict'
import test from 'node:test'
import fs from 'node:fs'
import path from 'node:path'
import vm from 'node:vm'

function loadErrorCodesModule() {
  const generatedPath = path.resolve(
    import.meta.dirname,
    '../consts/errorCodes.generated.js'
  )
  const generatedSource = fs.readFileSync(generatedPath, 'utf8')
  const generatedTransformed = generatedSource
    .replace(/export const /g, 'const ')
    .concat('\nmodule.exports = { RpcErrorCode };\n')

  const generatedSandbox = { module: { exports: {} }, exports: {} }
  vm.runInNewContext(generatedTransformed, generatedSandbox, {
    filename: generatedPath,
  })

  const filePath = path.resolve(import.meta.dirname, '../consts/errorCodes.js')
  const source = fs.readFileSync(filePath, 'utf8')
  const transformed = source
    .replace(
      /import\s+\{\s*RpcErrorCode\s*\}\s+from\s+["']\.\/errorCodes\.generated\.js["']\s*/u,
      'const { RpcErrorCode } = __generated__\n'
    )
    .replace(/export\s+\{\s*RpcErrorCode\s*\}\s*/u, '')
    .replace(/export const /g, 'const ')
    .replace(/export function /g, 'function ')
    .concat(
      '\nmodule.exports = { RpcErrorCode, AUTH_FAILURE_ERROR_CODES, isAuthFailureCode, DEFAULT_RPC_ERROR_MESSAGES };\n'
    )

  const sandbox = {
    module: { exports: {} },
    exports: {},
    __generated__: generatedSandbox.module.exports,
  }
  vm.runInNewContext(transformed, sandbox, { filename: filePath })
  return sandbox.module.exports
}

function loadErrorMessageModule() {
  const filePath = path.resolve(import.meta.dirname, './errorMessage.js')
  const source = fs.readFileSync(filePath, 'utf8')
  const transformed = source
    .replace(
      /import\s+\{\s*logout\s*\}\s+from\s+["']@\/common\/auth\/auth["']\s*/u,
      'const { logout } = __auth__\n'
    )
    .replace(
      /import\s+\{[\s\S]*?\}\s+from\s+["']@\/common\/consts\/errorCodes["']\s*/u,
      'const { DEFAULT_RPC_ERROR_MESSAGES, RpcErrorCode, isAuthFailureCode } = __errorCodes__\n'
    )
    .replace(/export function /g, 'function ')
    .replace(/export const /g, 'const ')
    .concat(
      '\nmodule.exports = { getUserFacingErrorMessage, getActionErrorMessage, handleRpcError, ERROR_MESSAGES };\n'
    )

  const sandbox = {
    module: { exports: {} },
    exports: {},
    __auth__: { logout() {} },
    __errorCodes__: errorCodesModule,
  }
  vm.runInNewContext(transformed, sandbox, { filename: filePath })
  return sandbox.module.exports
}

const errorCodesModule = loadErrorCodesModule()
const { RpcErrorCode, DEFAULT_RPC_ERROR_MESSAGES } = errorCodesModule
const { getUserFacingErrorMessage, getActionErrorMessage } =
  loadErrorMessageModule()

test('errorMessage: 网络错误统一翻译为中文', () => {
  assert.equal(
    getUserFacingErrorMessage(
      { message: 'Network error', isNetworkError: true },
      '登录失败，请稍后重试'
    ),
    '网络错误，请稍后重试'
  )
})

test('errorMessage: 已知错误码优先走现有中文码表', () => {
  assert.equal(
    getUserFacingErrorMessage(
      { message: 'Business error', code: RpcErrorCode.AUTH_REQUIRED },
      '登录失败，请稍后重试'
    ),
    DEFAULT_RPC_ERROR_MESSAGES[RpcErrorCode.AUTH_REQUIRED]
  )
})

test('errorMessage: 已是中文的后端文案保持原样', () => {
  assert.equal(
    getUserFacingErrorMessage('用户名已存在', '注册失败，请稍后重试'),
    '用户名已存在'
  )
})

test('errorMessage: 未知英文原文收口到页面 fallback', () => {
  assert.equal(
    getUserFacingErrorMessage(
      { message: 'temporary upstream failure' },
      '加载失败，请稍后重试'
    ),
    '加载失败，请稍后重试'
  )
})

test('errorMessage: 动作型 helper 自动补齐标准中文兜底', () => {
  assert.equal(
    getActionErrorMessage({ message: 'temporary upstream failure' }, '登录'),
    '登录失败，请稍后重试'
  )
})

test('errorMessage: 动作型 helper 支持自定义后缀', () => {
  assert.equal(
    getActionErrorMessage({ message: 'temporary upstream failure' }, '登录', {
      suffix: '请检查账号密码',
    }),
    '登录失败，请检查账号密码'
  )
})
