# mrembiasz-blog

Astro static blog served from a private S3 bucket through CloudFront.

## Local development

```sh
npm ci
npm run dev
```

Blog posts live in `src/content/blog/*.mdx`.

Mermaid diagrams can be written as fenced `mermaid` code blocks in posts. They
are rendered to static SVG images during `astro build`, so local and CI builds
need Playwright Chromium available.

## CI checks

```sh
npm run lint:frontend
npm run scan:frontend:packages
npm run build
deploy/scripts/download-backend-dependencies.sh
deploy/scripts/test-backend-lambdas.sh
deploy/scripts/build-backend-lambdas.sh
```

Jenkins runs frontend checks and Go backend test/build validation in parallel,
then Terraform plan before main-only Terraform apply, Lambda code deployment,
and static site deployment.

## Documentation

- [CI/CD pipeline overview](docs/cicd-pipeline.md)
- [Architecture decisions](docs/adr)
