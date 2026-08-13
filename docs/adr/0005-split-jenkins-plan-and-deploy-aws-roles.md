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

Protect `main` in GitHub because it is part of the AWS deployment boundary. The
repository uses an active GitHub ruleset named `protect main` for the default
branch. It requires pull request flow and the Jenkins check before merge:

```text
continuous-integration/jenkins/pr-head
```

The repository is operated as a solo-maintained public repository. GitHub
settings were checked with GitHub CLI and show that `kerbatek` is the only
collaborator with write/admin access, pull request creation is restricted to
collaborators, the ruleset has no bypass actors, and required approving reviews
are disabled.

Required reviews are intentionally disabled because this is a solo-maintained
repository and GitHub does not let the pull request author approve their own PR.
The required Jenkins check is the enforced merge gate.

PR Jenkinsfile changes still run with the plan role, so the plan role must stay
non-destructive. A malicious PR can alter CI behavior, but it cannot assume the
deploy role unless that change reaches protected `main`. The destructive AWS
boundary is enforced by the deploy role trust policy plus GitHub ruleset and
required Jenkins status check.

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

GitHub branch protection must stay enabled for `main`. If it is removed, a
direct push to `main` could trigger the Jenkins subject that is allowed to
assume the deploy role.

The plan role needs enough read permissions for Terraform provider refresh and
plan generation, but it must not receive S3 object writes to the site bucket,
CloudFront invalidation, or IAM write permissions.

The deploy role is broader by design. It owns production mutation for this app
and must remain restricted to the `main` Jenkins subject.
