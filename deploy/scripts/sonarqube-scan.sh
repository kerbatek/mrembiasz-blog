#!/usr/bin/env sh
set -u

component=${1:-}

case "$component" in
  frontend)
    project_name='mrembiasz-blog frontend'
    sources='src/frontend'
    ;;
  backend)
    project_name='mrembiasz-blog backend'
    sources='src/backend,tests/backend'
    ;;
  terraform)
    project_name='mrembiasz-blog terraform'
    sources='terraform,deploy'
    ;;
  *)
    echo "Usage: $0 frontend|backend|terraform" >&2
    exit 2
    ;;
esac

log_file="sonar-${component}-scanner.log"
check_text_file="sonar-${component}-dashboard-url.txt"
rm -f "$log_file" "$check_text_file"

set -- \
  -Dsonar.projectKey="mrembiasz-blog-${component}" \
  -Dsonar.projectName="$project_name" \
  -Dsonar.sources="$sources"

if [ "${BRANCH_NAME:-main}" != 'main' ]; then
  set -- "$@" -Dsonar.newCode.referenceBranch=main
fi

if [ "$component" = 'backend' ]; then
  set -- "$@" \
    -Dsonar.tests="$sources" \
    -Dsonar.test.inclusions='**/*_test.go' \
    -Dsonar.exclusions='**/*_test.go,dist/**,.astro/**,node_modules/**,terraform/.terraform/**,deploy/backend-lambdas/**'
fi

sonar-scanner "$@" > "$log_file" 2>&1
scanner_status=$?

cat "$log_file"

if [ "$scanner_status" -ne 0 ]; then
  exit "$scanner_status"
fi

sed -n 's/.*INFO  \(ANALYSIS SUCCESSFUL, you can find the results at: .*\)/\1/p' "$log_file" \
  | tail -n 1 > "$check_text_file"
