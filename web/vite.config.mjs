import { defineConfig, loadEnv } from 'vite'
import react from '@vitejs/plugin-react'
import { resolve } from 'path'
import { loadDevPorts } from '../scripts/dev-ports.mjs'

const DEV_HOST = '127.0.0.1'
const devPorts = loadDevPorts(resolve(import.meta.dirname, '..'))
const resolveRuntimeWebPort = (rawValue) => {
  const normalized = String(rawValue || '').trim()
  if (!normalized) return devPorts.web
  const port = Number(normalized)
  const auxEnd = devPorts.auxStart + 99
  if (!Number.isInteger(port) || port < 1024 || port > 65535) {
    throw new Error('DEV_WEB_RUNTIME_PORT 必须是 1024..65535 的整数端口')
  }
  if (
    port !== devPorts.web &&
    port !== devPorts.style &&
    (port < devPorts.auxStart || port > auxEnd)
  ) {
    throw new Error(
      `DEV_WEB_RUNTIME_PORT=${port} 必须使用固定 Web/Style 端口或辅助区间 ${devPorts.auxStart}-${auxEnd}`
    )
  }
  return port
}
const DEV_PORT = resolveRuntimeWebPort(process.env.DEV_WEB_RUNTIME_PORT)
const DEV_ORIGIN = `http://${DEV_HOST}:${DEV_PORT}`

const normalizeDevLocalUrl = (url) => {
  return String(url || '').replace(`http://localhost:${DEV_PORT}`, DEV_ORIGIN)
}

const devLocalhostOriginNormalizer = () => ({
  name: 'openai-oauth-dev-localhost-origin-normalizer',
  apply: 'serve',
  configureServer(server) {
    const printUrls = server.printUrls.bind(server)
    server.printUrls = () => {
      if (server.resolvedUrls?.local) {
        server.resolvedUrls.local =
          server.resolvedUrls.local.map(normalizeDevLocalUrl)
      }
      printUrls()
    }
  },
  transformIndexHtml() {
    return [
      {
        tag: 'script',
        injectTo: 'head-prepend',
        children: `
;(function () {
  var loc = window.location
  if (loc.protocol === 'http:' && loc.hostname === 'localhost' && loc.port === '${DEV_PORT}') {
    loc.replace('${DEV_ORIGIN}' + loc.pathname + loc.search + loc.hash)
  }
})()
`,
      },
    ]
  },
})

// https://vitejs.dev/config/
export default defineConfig(({ command, mode }) => {
  // 读取 .env.* 文件
  const env = loadEnv(mode, process.cwd(), '')

  const isProd = mode === 'production'
  const isDev = mode === 'development'
  const apiProxyTarget =
    env.VITE_API_PROXY_TARGET || `http://127.0.0.1:${devPorts.http}`

  // 只在开发环境打印调试信息，避免打包时报一堆东西
  if (!isProd) {
    console.log('env =', env)
    console.log('command =', command) // 'serve' | 'build'
    console.log('mode =', mode) // 'development' | 'production' | 自定义的mode
  }

  return {
    base: isDev ? '/' : env.VITE_BASE_URL || '/', // dev 在根，构建/预览在子路径

    plugins: [
      // 本机开发统一用 IPv4 origin，避免 localhost 解析或代理链路导致源模块加载抖动。
      isDev && devLocalhostOriginNormalizer(),
      react(), // 处理react
    ].filter(Boolean),

    /**
     * 用 esbuild 在【生产环境】移除所有 console / debugger
     *  - 只影响打包产物，不影响开发环境
     */
    esbuild: {
      drop: isProd ? ['console', 'debugger'] : [],
    },

    reportCompressedSize: false, // 报告压缩大小

    build: {
      outDir: 'build', // 输出目录
      assetsDir: 'assets', // 资源目录
      sourcemap: !isProd, // 生产关闭 sourcemap
      minify: 'esbuild',
      cssMinify: 'esbuild',
      target: 'es2018',
      rollupOptions: {
        output: {
          // 对齐 webpack 的命名风格
          entryFileNames: 'assets/[name].[hash].js',
          chunkFileNames: 'assets/[name].[hash].js',
          assetFileNames: 'assets/[name].[hash].[ext]',

          // 粗粒度拆分：react 等进 vendors，其余 node_modules 进 vendor
          manualChunks(id) {
            if (id.includes('node_modules')) {
              if (
                /[\\/]node_modules[\\/](react|react-dom|react-router|react-router-dom|antd|mobx)/.test(
                  id
                )
              ) {
                return 'vendors'
              }
              return 'vendor'
            }
          },
        },
      },
    },

    resolve: {
      alias: {
        '@': resolve(__dirname, './src'), // 别名
      },
      extensions: [
        '.js',
        '.jsx',
        '.ts',
        '.tsx',
        '.json',
        '.css',
        '.scss',
        '.sass',
      ],
    },

    // 开发缓存按仓库隔离，避免多个 Vite 项目共用 /tmp 缓存触发 Outdated Optimize Dep
    cacheDir: resolve(
      __dirname,
      mode === 'development' ? '.vite-cache' : 'build/.vite-cache'
    ),

    server: {
      host: '0.0.0.0', // 监听所有地址，方便局域网测试
      port: DEV_PORT,
      strictPort: true,
      open: DEV_ORIGIN,
      // 本机开发固定 IPv4，避免 localhost 优先解析到 ::1 时 HMR / 代理间歇失败。
      hmr: {
        host: DEV_HOST,
        clientPort: DEV_PORT,
      },
      proxy: {
        '/rpc': {
          target: apiProxyTarget,
          changeOrigin: true,
        },
        '/auth': {
          target: apiProxyTarget,
          changeOrigin: true,
        },
        '/public': {
          target: apiProxyTarget,
          changeOrigin: true,
        },
      },
      // 如果挂 NAS / WSL / Docker 卷，再打开下面这个
      // watch: {
      //   usePolling: true,
      //   interval: 100,
      // },
    },

    preview: {
      host: '127.0.0.1',
      port: devPorts.auxStart + 90,
      strictPort: true,
    },

    optimizeDeps: {
      include: ['react', 'react-dom', 'react-router-dom', 'react-helmet-async'],
    },
  }
})
