import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'
import path from "path"
import tailwindcss from "@tailwindcss/vite"

// https://vite.dev/config/
export default defineConfig({
  plugins: [
    react(),
    tailwindcss(),
  ],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
  server: {
    proxy: {
      '/api': {
        target: 'http://localhost:9898',
        changeOrigin: true,
      }
    }
  },
  build: {
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (!id.includes("node_modules")) {
            return undefined
          }

          if (id.includes("@xyflow") || id.includes("@dnd-kit") || id.includes("react-resizable-panels")) {
            return "workflow-vendor"
          }

          if (id.includes("@codemirror") || id.includes("codemirror")) {
            return "editor-vendor"
          }

          if (id.includes("recharts") || id.includes("@tanstack/react-table")) {
            return "analytics-vendor"
          }

          return undefined
        },
      },
    },
  },
  test: {
    environment: "jsdom",
    setupFiles: "./src/test/setup.ts",
  }
})
