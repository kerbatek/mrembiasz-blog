# CI/CD pipeline overview

This repository uses one Jenkins multibranch pipeline at `deploy/Jenkinsfile`.
The pipeline runs on ephemeral Kubernetes agent Pods defined in
`deploy/jenkins/agent-pod.yaml`.

## Pipeline flow

```text
Git push or pull request
  -> Jenkins multibranch job
  -> ephemeral Kubernetes agent Pod
  -> frontend checks/build and Go backend validation in parallel
  -> SonarQube frontend, backend, and Terraform scans with quality gates
  -> Terraform plan
  -> Terraform apply on main
  -> Lambda code deploy on main
  -> S3 sync on main
  -> CloudFront invalidation on main
  -> smoke test on main
```

## Stages

Validation runs in parallel. Frontend stages run sequentially in the `node`
container:

```text
Install Frontend Dependencies -> npm ci
Scan Frontend Packages        -> npm run scan:frontend:packages
Lint Frontend                 -> npm run lint:frontend
Build Astro                   -> npm run build
```

Backend stages run sequentially in the `go` container:

```text
Download Backend Dependencies -> deploy/scripts/download-backend-dependencies.sh
Run Backend Tests             -> deploy/scripts/test-backend-lambdas.sh
Build Go Lambdas              -> deploy/scripts/build-backend-lambdas.sh
```

`Run Backend Tests` runs each backend Lambda module in parallel internally.
`Build Go Lambdas` creates the Lambda `bootstrap` binaries and zip files that
Terraform reads during plan and that the release stage deploys on `main`.
The Jenkins agent Pod mounts the repo cache PVC at `/cache`. The node container
uses it for npm's cache, the Go container uses it for `GOMODCACHE` and
`GOCACHE`, and the Terraform container uses it for provider plugins.

Each Jenkins stage publishes a GitHub check run with the stage name through the
Jenkins Checks API and GitHub Checks plugins. The check links back to the
Jenkins build URL and reports the failing stage directly in GitHub.

`lint:frontend` runs:

- Prettier formatting checks for frontend source and config files
- ESLint checks for JavaScript, TypeScript, and Astro code
- `astro check` for Astro and TypeScript diagnostics

ESLint is the code lint step. It catches likely bugs and unsafe or hard-to-read
frontend code before the site is built.

`scan:frontend:packages` runs `npm audit --audit-level=high` against the locked
frontend dependency graph. CI fails on high or critical npm advisories.

SonarQube runs after validation succeeds. It analyzes frontend, backend, and
Terraform code as separate SonarQube projects so each area has its own scan and
quality gate GitHub check:

```text
mrembiasz-blog-frontend
mrembiasz-blog-backend
mrembiasz-blog-terraform
```

Those projects must already exist in SonarQube, or the Jenkins token must have
permission to create projects during analysis.
Jenkins must have a SonarQube server named `SonarQube` configured in the
SonarQube Scanner plugin, and the SonarQube projects must send webhooks to
Jenkins so `waitForQualityGate` can resume the build.
If one SonarQube area fails, Jenkins still runs the remaining SonarQube scans
so GitHub shows which areas passed and failed. Terraform plan and release only
run when all quality gates pass.

The intended SonarQube quality gate is enforced on new code only:

```text
Issues                       is greater than       0
Security Hotspots Reviewed   is less than          100%
Duplicated Lines (%)         is greater than       3.0%
```

Coverage is intentionally not part of the gate yet because CI does not publish
frontend or backend coverage reports to SonarQube.

`Terraform Plan` runs after validation and the SonarQube quality gates succeed,
and assumes:

```text
arn:aws:iam::047588357922:role/mrembiasz-blog-jenkins-plan
```

The plan script uses `terraform plan -lock=false` so the plan role can stay
read-only for Terraform state and does not need state lock write permissions.
The readable text plan (`tfplan.txt`) and plan summary (`tfplan-summary.txt`)
are archived as Jenkins artifacts. The `Terraform Plan` GitHub check title is
set from the plan result, for example `No Terraform changes.`. `Terraform
Apply` reuses that same title on `main`.

`Terraform Apply` runs only on `main` and assumes:

```text
arn:aws:iam::047588357922:role/mrembiasz-blog-jenkins-deploy
```

`Deploy Static Site` also runs only on `main` with the deploy role. It syncs
`dist/` to the private S3 site bucket, creates a CloudFront invalidation, and
waits for that invalidation to complete.

`Deploy Backend Lambdas` runs only on `main` with the deploy role. It updates
the Lambda function code from `deploy/backend-lambdas/*.zip` with
`aws lambda update-function-code` and waits for each function update to finish.

`Smoke Test Website` runs after the invalidation completes. The Astro build
embeds the Jenkins `GIT_COMMIT` into a `deploy-id` meta tag, and the smoke test
checks that `https://blog.mrembiasz.pl/` serves that exact deploy id.

Jenkins disables concurrent builds for this pipeline. That keeps Terraform
apply, S3 sync, and CloudFront invalidation serialized for this repository.

## DNS and certificate dependency

DNS for `blog.mrembiasz.pl` is managed outside AWS. Terraform requests and
tracks the CloudFront ACM certificate in `us-east-1`, but the ACM validation
CNAME and the final `blog.mrembiasz.pl` hostname record must be maintained in
the external DNS provider.

During first setup or certificate replacement, `terraform apply` waits for ACM
validation. If the external DNS record is missing or stale, the apply will wait
until Terraform times out.

## AWS role boundary

The Jenkins platform repository owns the OIDC provider and role trust policies.
This app repository owns only app-specific AWS resources and policy attachments.

The intended trust boundary is:

```text
mrembiasz-blog-jenkins-plan
  trusts: https://jenkins.mrembiasz.pl/job/mrembiasz-blog/job/*/

mrembiasz-blog-jenkins-deploy
  trusts: https://jenkins.mrembiasz.pl/job/mrembiasz-blog/job/main/
```

PR and branch builds can build Astro, test/build Go Lambdas, and run Terraform
plan, but only after validation succeeds. They cannot assume the deploy role.
Only the `main` Jenkins subject can mutate AWS infrastructure, update Lambda
code, or publish the website.

## GitHub merge boundary

Because the deploy role trusts only the Jenkins `main` subject, GitHub branch
protection is part of the AWS security boundary. A change should not reach
`main` unless the protected branch rules allow it.

This repository uses an active GitHub ruleset named `protect main` for the
default branch. It blocks deletion and non-fast-forward updates, requires pull
request flow, allows squash merge, and requires only the Jenkins Checks API
summary check:

```text
Jenkins
```

That aggregate check represents the full Jenkins pipeline result, so it is the
only required merge check. Area-level GitHub Checks are also published for
readability and debugging, but they are not individually required:

```text
Frontend
Backend
SonarQube Terraform
Terraform Plan
Release
```

The legacy GitHub Status API context is disabled in Jenkins Branch Source using
the `Skip build status notifications` behavior:

```text
continuous-integration/jenkins/pr-head
```

The repository is operated as a solo-maintained public repository. GitHub
settings were checked with GitHub CLI and show:

```text
collaborators with write/admin access: kerbatek
pull_request_creation_policy: collaborators_only
ruleset bypass actors: none
required approving reviews: 0
```

Required reviews are intentionally disabled. This is a solo-maintained
repository, and GitHub does not let the pull request author approve their own
PR. The enforced merge gate is the required Jenkins status check before a PR
can reach `main`.

PR Jenkinsfile changes still run with the plan role, so the plan role must stay
non-destructive. A malicious PR can alter CI behavior, but it cannot assume the
deploy role unless that change reaches protected `main`. The destructive AWS
boundary is enforced by the deploy role trust policy plus GitHub ruleset and
required Jenkins status check.

## Platform permissions

The blog-specific Jenkins roles are defined in
`<private-infra-repo>/terraform/mrembiasz-blog-state`.

`mrembiasz-blog-jenkins-plan` has:

- AWS `ReadOnlyAccess`
- read-only access to the Terraform state bucket and state object

This is enough for `terraform plan -lock=false`: Terraform can read remote
state and inspect AWS resources, but it cannot update the state lock, modify
the website bucket, create CloudFront invalidations, or write IAM changes.

`mrembiasz-blog-jenkins-deploy` has:

- read/write access to the Terraform state object and `.tflock` object
- AWS managed S3, CloudFront, and ACM permissions used to create and update
  this site's infrastructure
- Lambda code update permissions for this app's backend functions
- app IAM bootstrap permissions for the `mrembiasz-blog-deploy` policy and its
  attachment to the deploy role

This role is intentionally broader because `terraform apply` creates and
updates AWS resources, backend deployment updates Lambda code, and the static
deployment publishes files to S3 and invalidates CloudFront. Its OIDC trust is
limited to the `main` Jenkins subject so PR and branch builds cannot assume it.
