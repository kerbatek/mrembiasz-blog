# ADR 0002: Use Jenkins as the CI/CD pipeline provider

Date: 2026-08-11

## Status

Accepted

## Context

The site is an Astro static frontend deployed to private S3 and served publicly
through CloudFront.

The project needs a CI/CD pipeline that builds the site, runs checks, and
publishes `dist/` to AWS.

The target delivery flow is:

```text
Git push
  -> Jenkins Controller in homelab Kubernetes
  -> Ephemeral Jenkins agent Pod
  -> npm ci -> tests -> astro build
  -> dist/
  -> S3
  -> CloudFront
  -> Website
```

## Decision

Use Jenkins as the CI/CD pipeline provider.

Jenkins will run as a controller inside the homelab Kubernetes cluster. Builds
will run on ephemeral Kubernetes agent Pods instead of permanent runners. The
pipeline definition will live in the repository as a `Jenkinsfile`.

Running Jenkins in the homelab is acceptable because Jenkins is only on the
publishing path. If the homelab or Jenkins is unavailable, new content cannot be
built and deployed during that outage, but the already deployed website keeps
serving from AWS through S3 and CloudFront.

The initial pipeline will:

1. Install dependencies with `npm ci`.
2. Run the project's tests.
3. Build the static site with Astro.
4. Deploy `dist/` to S3 with:

```sh
aws s3 sync dist/ s3://bucket --delete
```

Jenkins will receive narrowly scoped AWS permissions for deployment, preferably
through an IAM deployment role instead of a broadly privileged AWS user.

## Rationale

The production architecture is intended to avoid always-on cloud servers and
their fixed monthly cost. The website is served by S3 and CloudFront, and any
backend work should use managed or serverless AWS services instead of deployed
cloud VMs.

Running Jenkins in the existing homelab Kubernetes cluster reuses capacity that
is already online. It does not introduce another always-on cloud cost. Running
Jenkins on an EC2 instance would add a continuously billed server just to
publish content.

Jenkins still breaks the pure serverless idea for CI/CD, but not for the
production runtime. Builds use ephemeral Kubernetes agent Pods, which keeps
build workers disposable and avoids maintaining permanent runners.

Hosted CI would reduce homelab operations, but it would move the pipeline away
from the existing cluster and require trusting another hosted runner path with
deployment credentials.

The accepted tradeoff is that homelab outages can delay publishing. They do not
take down the website because the deployed assets are served from AWS by S3 and
CloudFront.

## Consequences

This keeps the CI/CD path small, explicit, and maintainable for a production
static site.

Jenkins adds operational responsibility because the controller runs in the
homelab Kubernetes cluster. The failure mode is limited to delayed publishing,
not website downtime. Ephemeral agent Pods keep build workers disposable and
avoid permanent runner maintenance.
