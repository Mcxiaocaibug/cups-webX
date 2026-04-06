# Frontend

This is the Vue 3 + Vite frontend for CUPS Web.

Build:

```bash
cd frontend
bun install --frozen-lockfile
bun run build
```

If Bun is unavailable, use:

```bash
cd frontend
npm ci
npm run build
```

The production bundle is written to `../internal/webui/dist` so the Go server can embed it with the `frontend_dist` build tag.
