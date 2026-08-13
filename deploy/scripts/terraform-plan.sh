#!/usr/bin/env sh
set -eu

terraform -chdir=terraform init -input=false
terraform -chdir=terraform fmt -check
terraform -chdir=terraform validate
terraform -chdir=terraform plan -input=false -lock=false -out=tfplan
terraform -chdir=terraform show -no-color tfplan > terraform/tfplan.txt
