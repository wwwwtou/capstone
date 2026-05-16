## Run Locally

**Prerequisites:**  Node.js


1. Install dependencies:
   `npm install`
2. Set the `GEMINI_API_KEY` in [.env.local](.env.local) to your Gemini API key
3. Run the app:
   `npm run dev`

## Minimal CD with GitHub Actions

This repo now includes a minimal CD flow that:

1. Runs real CI checks in GitHub Actions.
2. Triggers a Render deploy hook after `main` passes.
3. Uses the root `Dockerfile` to run the production server that serves the built frontend and API.

### What you need

- A free Render account
- A GitHub repository for this project
- A Render deploy hook URL stored as a GitHub secret

### Render IaC

The repository now includes a `render.yaml` Blueprint Specification at the project root.
This is the IaC source of truth for Render: it defines the service type, region, build/runtime
selection, health check, and environment variables in code.

The file currently maps to a single Docker web service:

- service name: `e-commerce-video-recsys-mvp`
- service type: `web`
- runtime: `docker`
- region: `singapore`
- plan: `free`
- health check: `/api/v1/health`
- production env: `NODE_ENV=production`
- secret env placeholder: `GEMINI_API_KEY` (sync disabled)

### Why this setup

This is the smallest real CD setup that still fits the repo:
GitHub Actions runs CI and then asks Render to redeploy the app.
No AWS account is required.

### Create the Render service

1. Sign in to Render.
2. Create a new **Blueprint Instance** from this GitHub repo.
3. Render will read `render.yaml` and provision the service automatically.
4. Keep the service type as **Docker** so it uses the root `Dockerfile`.
5. Create a deploy hook in Render and copy its URL.
6. Add the hook URL to GitHub repository secrets as `DEPLOY_HOOK_URL`.
7. If Render still warns about ports, confirm the service is not created as a Node runtime service by mistake.

### GitHub Actions flow

On every push to `main`:

- Backend lint + gosec + tests + build run first.
- Frontend lint + build run next.
- If all jobs pass, GitHub Actions triggers the Render deploy hook.

### Local commands

```bash
npm install
npm run dev
```

### Notes

- If `DEPLOY_HOOK_URL` is missing, the deploy job fails on purpose so you know CD is not wired yet.
- This is a real CD trigger, but the deployment target is Render rather than AWS.
- The app must be deployed as a Docker service; otherwise Render may run a Node preset that listens on localhost and fails the port scan.
- `render.yaml` is the IaC layer for Render; GitHub Actions is the CI/CD orchestrator.
- If you later want AWS again, you can swap the deploy hook job for ECR + ECS or EC2 SSH deploy.

------
