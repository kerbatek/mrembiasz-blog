# ADR 0004: Use Terraform as infrastructure as code

Date: 2026-08-11

## Status

Accepted

## Context

The blog runs on app-specific AWS infrastructure: S3, CloudFront, IAM deployment
roles, and later serverless backend resources. These resources need to be
reproducible, reviewable, and recoverable without relying on manual console
changes.

## Decision

Use Terraform to manage this application's AWS infrastructure.

Terraform will define the production AWS resources required by this application,
starting with:

1. S3 bucket for static assets.
2. CloudFront distribution.
3. IAM deployment role and policies.
4. App-specific IAM trust policy for the deployment role.

Jenkins itself, the Jenkins Kubernetes platform, and the shared OIDC provider
configuration are managed by separate infrastructure repositories. This
repository owns only the AWS resources and IAM roles/policies needed by this
application.

Application source code, blog content, and Jenkins pipeline logic will stay in
the repository outside Terraform modules.

## Rationale

Terraform gives the AWS infrastructure a declarative source of truth that can be
reviewed before it changes production.

Manual console changes are fast once, but they are hard to audit and repeat.
Ad hoc scripts can create resources, but they do not manage drift or dependency
ordering as clearly as Terraform.

Terraform also fits the static and serverless architecture because most
resources are managed AWS services with stable configuration.

## Consequences

Infrastructure changes should go through `terraform plan` before apply.

Terraform state becomes production metadata and must be stored safely. Local
state is acceptable only before shared infrastructure exists; remote state
should be added when the AWS account layout is ready.
