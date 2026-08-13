# mrembiasz-blog

Astro static blog served from a private S3 bucket through CloudFront.

## Local development

```sh
npm ci
npm run dev
```

## CI checks

```sh
npm run lint:frontend
npm run scan:frontend:packages
npm run build
```

Jenkins runs the frontend checks and Terraform plan in parallel before
main-only Terraform apply and static site deployment.

## Documentation

- [CI/CD pipeline overview](docs/cicd-pipeline.md)
- [Architecture decisions](docs/adr)
