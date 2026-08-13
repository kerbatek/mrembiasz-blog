# CI/CD pipeline overview

This repository uses one Jenkins multibranch pipeline at `deploy/Jenkinsfile`.
The pipeline runs on ephemeral Kubernetes agent Pods defined in
`deploy/jenkins/agent-pod.yaml`.

## Pipeline flow

```text
Git push or pull request
  -> Jenkins multibranch job
  -> ephemeral Kubernetes agent Pod
  -> Astro build
  -> Terraform plan
  -> Terraform apply on main
  -> S3 sync on main
  -> CloudFront invalidation on main
```

## Stages

`Build Astro` runs in the `node` container and executes
`deploy/scripts/build-astro.sh`.

`Terraform Plan` runs in the `terraform` container and assumes:

```text
arn:aws:iam::047588357922:role/mrembiasz-blog-jenkins-plan
```

The plan script uses `terraform plan -lock=false` so the plan role can stay
read-only for Terraform state and does not need state lock write permissions.
The generated binary plan and readable text plan are archived as Jenkins
artifacts.

`Terraform Apply` runs only on `main` and assumes:

```text
arn:aws:iam::047588357922:role/mrembiasz-blog-jenkins-deploy
```

`Deploy Static Site` also runs only on `main` with the deploy role. It syncs
`dist/` to the private S3 site bucket and creates a CloudFront invalidation.

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

PR and branch builds can build Astro and run Terraform plan, but they cannot
assume the deploy role. Only the `main` Jenkins subject can mutate AWS
infrastructure or publish the website.

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
- app IAM bootstrap permissions for the `mrembiasz-blog-deploy` policy and its
  attachment to the deploy role

This role is intentionally broader because `terraform apply` creates and
updates AWS resources, and the static deployment publishes files to S3 and
invalidates CloudFront. Its OIDC trust is limited to the `main` Jenkins subject
so PR and branch builds cannot assume it.
