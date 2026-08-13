# ADR 0003: Use AWS IAM OIDC for Jenkins credentials

Date: 2026-08-11

## Status

Accepted

## Context

Jenkins validates and deploys the Astro blog through AWS. CI needs credentials
to run `terraform plan`; deployment needs stronger credentials for
`terraform apply`, syncing `dist/` to S3, and invalidating CloudFront caches.

Storing long-lived AWS access keys in Jenkins would create a credential rotation
and leakage risk. The deployment permission should also stay narrower than a
general-purpose AWS user.

## Decision

Use AWS IAM OIDC federation for Jenkins credentials.

Jenkins jobs will request short-lived AWS credentials by assuming app-specific
IAM roles through OIDC. Jenkins itself, the Jenkins Kubernetes platform, and
the shared OIDC provider are managed by separate infrastructure repositories.

This repository will own only app-specific IAM policies and attachments. The
roles and OIDC trust policies are owned by the Jenkins platform repository.
The pipeline uses separate roles for read-only planning and production
deployment:

- `mrembiasz-blog-jenkins-plan` is used by CI builds and can run
  `terraform plan -lock=false`.
- `mrembiasz-blog-jenkins-deploy` is trusted only for the `main` Jenkins
  subject and can run `terraform apply` plus publish the built site.

The intended credential flow is:

```text
Jenkins CI pipeline
  -> OIDC token
  -> AWS STS AssumeRoleWithWebIdentity
  -> mrembiasz-blog-jenkins-plan
  -> terraform plan

Jenkins main deployment stages
  -> OIDC token
  -> AWS STS AssumeRoleWithWebIdentity
  -> mrembiasz-blog-jenkins-deploy
  -> terraform apply
  -> aws s3 sync dist/ s3://bucket --delete
```

## Rationale

OIDC avoids storing long-lived AWS access keys in Jenkins. Compromise of a
single build credential is limited by token lifetime and the IAM role policy.

Separate plan and deploy roles keep the trust boundary explicit. Pull request
and regular CI jobs can inspect infrastructure changes without receiving
permissions to change AWS resources. Only the `main` Jenkins subject is trusted
to assume the deploy role.

This matches the production architecture better than an IAM user because the
pipeline uses temporary credentials for each run instead of maintaining static
secrets.

## Consequences

AWS access for Jenkins depends on the external Jenkins and OIDC platform
configuration being correct.

This repository owns only the app-specific policies and attachments required by
this pipeline.

The apply role policy should be limited to this app's deployment bucket and
CloudFront distribution. Additional permissions should be added only when the
pipeline actually needs them.
