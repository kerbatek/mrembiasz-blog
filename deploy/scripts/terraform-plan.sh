#!/usr/bin/env sh
set -eu

terraform -chdir=terraform init -input=false
terraform -chdir=terraform fmt -check
terraform -chdir=terraform validate

set +e
terraform -chdir=terraform plan -input=false -lock=false -detailed-exitcode -out=tfplan
plan_status="$?"
set -e

case "$plan_status" in
  0)
    echo "No Terraform changes." > terraform/tfplan-summary.txt
    ;;
  2)
    echo "Terraform changes detected. Review tfplan.txt in the Jenkins job artifacts." > terraform/tfplan-summary.txt
    ;;
  *)
    exit "$plan_status"
    ;;
esac

terraform -chdir=terraform show -no-color tfplan > terraform/tfplan.txt
