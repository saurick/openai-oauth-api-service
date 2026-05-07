export class JsonRpcMock {
  constructor({ url }) {
    this.url = url
  }

  async call(method, params = {}, options = {}) {
    console.log('[MOCK] RPC', this.url, method, params)
    await new Promise((r) => setTimeout(r, 300))

    // ---- system ----
    if (this.url === 'system') {
      if (method === 'ping') {
        return {
          code: 0,
          message: 'OK',
          ping: { pong: 'mock-pong' },
        }
      }

      if (method === 'version') {
        return {
          code: 0,
          message: 'OK',
          version: { version: 'mock-1.0.0' },
        }
      }
    }

    // ---- auth ----
    if (this.url === 'auth') {
      if (method === 'login') {
        if (params.username === 'error') {
          // 模拟业务错误
          return {
            code: 401,
            message: 'Invalid username',
          }
        }

        return {
          code: 0,
          message: 'OK',
          login: {
            userId: 'mock-user-001',
            nickname: params.username || 'mock-nick',
          },
        }
      }

      if (method === 'logout') {
        return {
          code: 0,
          message: 'OK',
          logout: { success: true },
        }
      }
    }

    // 模拟 JSON-RPC 业务错误
    return {
      code: 400,
      message: `Unknown method: ${method}`,
    }
  }
}
