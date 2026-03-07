import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [vue()],
  server: {
    port: 5173, // 保持你的前端端口
    // 代理配置
    proxy: {
      '/api': {
        target: 'http://localhost:8080', 
        changeOrigin: true
      }
    }
  }
})