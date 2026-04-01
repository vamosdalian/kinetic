# Kinetic Web

This package contains the React and Vite frontend for Kinetic. The production build is embedded into the Go binary and served by the backend.

## Commands

```bash
npm ci
npm run dev
npm test
npm run lint
npm run build
```

## Notes

- The dev server proxies `/api` requests to `http://localhost:9898`.
- `npm run build` produces the static assets embedded by [`web/embed.go`](/Users/lmc10232/project/Kinetic/web/embed.go).
- Docs content under [`web/docs`](/Users/lmc10232/project/Kinetic/web/docs) is copied into the final build by the Vite config.
