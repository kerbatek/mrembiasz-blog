# ADR 0003: Use AWS IAM OIDC for Jenkins credentials

Date: 2026-08-11

## Status

Accepted

## Context

Jenkins deploys the Astro build output to AWS. The pipeline needs AWS
credentials for actions such as syncing `dist/` to S3 and, later, invalidating
CloudFront caches.

Storing long-lived AWS access keys in Jenkins would create a credential rotation
and leakage risk. The deployment permission should also stay narrower than a
general-purpose AWS user.

## Decision

Use AWS IAM OIDC federation for Jenkins deployment credentials.

Jenkins jobs will request short-lived AWS credentials by assuming an
app-specific IAM deployment role through OIDC. Jenkins itself, the Jenkins
Kubernetes platform, and the shared OIDC provider are managed by separate
infrastructure repositories.

This repository will own only the app-specific deployment role and permissions.
The role will grant only the permissions required by this blog's deployment
pipeline, starting with access to sync the static site assets to the target S3
bucket.

The intended credential flow is:

```text
Jenkins pipeline
  -> OIDC token
  -> AWS STS AssumeRoleWithWebIdentity
  -> short-lived deployment credentials
  -> aws s3 sync dist/ s3://bucket --delete
```

## Rationale

OIDC avoids storing long-lived AWS access keys in Jenkins. Compromise of a
single build credential is limited by token lifetime and the IAM role policy.

A dedicated deployment role keeps the trust boundary explicit: Jenkins can
publish the static site, but it does not receive broad account permissions.

This matches the production architecture better than an IAM user because the
pipeline uses temporary credentials for each run instead of maintaining static
secrets.

## Consequences

AWS access for Jenkins depends on the external Jenkins and OIDC platform
configuration being correct.

This repository owns only the app-specific roles and policies required by this
pipeline.

The initial role policy should be limited to the deployment bucket. Additional
permissions, such as CloudFront invalidation, should be added only when the
pipeline actually needs them.
