import { defineConfig } from 'vite';
import legacy from '@vitejs/plugin-legacy';
import path from 'path';

export default defineConfig({
  root: 'web/src',
  publicDir: '../static/assets',

  build: {
    outDir: '../static/dist',
    emptyOutDir: true,

    rollupOptions: {
      input: {
        main: path.resolve(__dirname, 'web/src/js/main.js')
      },
      output: {
        // JS 和 CSS 都使用固定文件名，方便 Go 模板引用
        entryFileNames: 'js/[name].js',
        chunkFileNames: 'js/[name].js',
        assetFileNames: (assetInfo) => {
          if (/\.css$/.test(assetInfo.name)) {
            return 'css/[name][extname]';
          }
          return 'assets/[name].[hash][extname]';
        }
      }
    },

    // 生产构建优化
    minify: 'terser',
    terserOptions: {
      compress: {
        drop_console: true,
        drop_debugger: true
      }
    },
    sourcemap: false,

    // 代码分割配置
    chunkSizeWarningLimit: 500,
    cssCodeSplit: true
  },

  // 开发服务器配置
  server: {
    port: 3000,
    strictPort: false,
    open: false,
    cors: true,

    // 代理到 Go 后端
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true
      },
      '/metrics': {
        target: 'http://localhost:8080',
        changeOrigin: true
      }
    }
  },

  // 插件配置
  plugins: [
    // 支持旧版浏览器
    legacy({
      targets: ['defaults', 'not IE 11']
    })
  ],

  // 路径别名
  resolve: {
    alias: {
      '@': path.resolve(__dirname, 'web/src'),
      '@js': path.resolve(__dirname, 'web/src/js'),
      '@css': path.resolve(__dirname, 'web/src/css'),
      '@components': path.resolve(__dirname, 'web/src/js/components'),
      '@utils': path.resolve(__dirname, 'web/src/js/utils'),
      '@api': path.resolve(__dirname, 'web/src/js/api')
    }
  },

  // CSS 配置
  css: {
    postcss: './postcss.config.js'
  }
});
