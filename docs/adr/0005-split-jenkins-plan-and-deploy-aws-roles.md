# ADR 0005: Split Jenkins AWS roles for plan and deploy

Date: 2026-08-13

## Status

Accepted

## Context

The Jenkins pipeline runs for pull requests, branches, and `main`. All builds
need to build Astro and run Terraform plan, but only `main` should be able to
change AWS infrastructure or publish site assets.

Using one AWS role for every Jenkins run would make PR and branch builds
eligible for deploy permissions. That would weaken the CI/CD boundary even if
the Jenkinsfile only runs apply and deploy stages on `main`.

## Decision

Use two app-specific Jenkins AWS roles managed in
`<private-infra-repo>/terraform/mrembiasz-blog-state`:

- `mrembiasz-blog-jenkins-plan`
- `mrembiasz-blog-jenkins-deploy`

The plan role trusts Jenkins subjects for this repository's PR and branch jobs:

```text
https://jenkins.mrembiasz.pl/job/mrembiasz-blog/job/*/
```

The deploy role trusts only the `main` Jenkins subject:

```text
https://jenkins.mrembiasz.pl/job/mrembiasz-blog/job/main/
```

The Jenkinsfile assumes the plan role for `Terraform Plan` and the deploy role
only for `Terraform Apply` and `Deploy Static Site`, both guarded by
`when { branch 'main' }`.

## Rationale

This keeps review-time credentials different from production mutation
credentials. PR and branch builds can inspect proposed infrastructure changes,
but cannot assume the role that can apply Terraform, write to the site bucket,
or invalidate CloudFront.

`terraform plan` runs with `-lock=false` so the plan role only needs read
access to remote state. The deploy role keeps state read/write and lock access
because `terraform apply` must serialize changes.

## Consequences

The Jenkins platform repository must keep the role trust policies aligned with
the Jenkins job URL subjects.

The plan role needs enough read permissions for Terraform provider refresh and
plan generation, but it must not receive S3 object writes to the site bucket,
CloudFront invalidation, or IAM write permissions.

The deploy role is broader by design. It owns production mutation for this app
and must remain restricted to the `main` Jenkins subject.
