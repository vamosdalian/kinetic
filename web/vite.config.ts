import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'
import path from "path"
import fs from "node:fs/promises"
import tailwindcss from "@tailwindcss/vite"

const docsSourceDir = path.resolve(__dirname, "docs")
const docsOutputDirName = "docs-content"
const docsifySourceDir = path.resolve(__dirname, "node_modules", "docsify", "lib")
const docsifyOutputDirName = path.join("docsify", "vendor")

function getContentType(filePath: string) {
  if (filePath.endsWith(".md")) {
    return "text/markdown; charset=utf-8"
  }
  if (filePath.endsWith(".html")) {
    return "text/html; charset=utf-8"
  }
  if (filePath.endsWith(".js")) {
    return "application/javascript; charset=utf-8"
  }
  if (filePath.endsWith(".css")) {
    return "text/css; charset=utf-8"
  }
  return "text/plain; charset=utf-8"
}

function isPathInside(baseDir: string, targetPath: string) {
  return targetPath === baseDir || targetPath.startsWith(`${baseDir}${path.sep}`)
}

function createStaticDirMiddleware(baseDir: string, defaultFile: string) {
  return async (req: { url?: string }, res: { statusCode: number; setHeader: (name: string, value: string) => void; end: (body?: string | Buffer) => void }, next: () => void) => {
    const requestPath = decodeURIComponent((req.url ?? "/").split("?")[0])
    const relativePath = requestPath.replace(/^\/+/, "") || defaultFile
    const absolutePath = path.resolve(baseDir, relativePath)

    if (!isPathInside(baseDir, absolutePath)) {
      res.statusCode = 403
      res.end("Forbidden")
      return
    }

    try {
      const stat = await fs.stat(absolutePath)
      const filePath = stat.isDirectory() ? path.join(absolutePath, defaultFile) : absolutePath
      const content = await fs.readFile(filePath)
      res.setHeader("Content-Type", getContentType(filePath))
      res.end(content)
    } catch {
      next()
    }
  }
}

function docsContentPlugin() {
  let outDir = path.resolve(__dirname, "dist")

  return {
    name: "kinetic-docs-content",
    configResolved(config: { build: { outDir: string } }) {
      outDir = path.resolve(__dirname, config.build.outDir)
    },
    configureServer(server: {
      middlewares: {
        use: (path: string, handler: (req: { url?: string }, res: { statusCode: number; setHeader: (name: string, value: string) => void; end: (body?: string | Buffer) => void }, next: () => void) => void | Promise<void>) => void
      }
    }) {
      server.middlewares.use(`/${docsOutputDirName}`, createStaticDirMiddleware(docsSourceDir, "README.md"))
      server.middlewares.use(`/${docsifyOutputDirName}`, createStaticDirMiddleware(docsifySourceDir, "docsify.min.js"))
    },
    async closeBundle() {
      const docsTargetDir = path.join(outDir, docsOutputDirName)
      const docsifyTargetDir = path.join(outDir, docsifyOutputDirName)
      await fs.rm(docsTargetDir, { recursive: true, force: true })
      await fs.rm(docsifyTargetDir, { recursive: true, force: true })
      await fs.cp(docsSourceDir, docsTargetDir, { recursive: true })
      await fs.cp(docsifySourceDir, docsifyTargetDir, { recursive: true })
    },
  }
}

// https://vite.dev/config/
export default defineConfig({
  plugins: [
    react(),
    tailwindcss(),
    docsContentPlugin(),
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
