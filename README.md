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

### Why this setup

This is the smallest real CD setup that still fits the repo:
GitHub Actions runs CI and then asks Render to redeploy the app.
No AWS account is required.

### Create the Render service

1. Sign in to Render.
2. Create a new **Web Service** from this GitHub repo.
3. Set the service type to **Docker** so it uses the root `Dockerfile`.
4. Make sure the start command is the Docker default, or leave it empty so Render uses `Dockerfile`.
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
- If you later want AWS again, you can swap the deploy hook job for ECR + ECS or EC2 SSH deploy.
