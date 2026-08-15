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

sonar-scanner \
  -Dsonar.projectKey="mrembiasz-blog-${component}" \
  -Dsonar.projectName="$project_name" \
  -Dsonar.projectVersion="${GIT_COMMIT:-local}" \
  -Dsonar.sources="$sources" \
  > "$log_file" 2>&1
scanner_status=$?

cat "$log_file"

if [ "$scanner_status" -ne 0 ]; then
  exit "$scanner_status"
fi

sed -n 's/.*INFO  \(ANALYSIS SUCCESSFUL, you can find the results at: .*\)/\1/p' "$log_file" \
  | tail -n 1 > "$check_text_file"
