#!/usr/bin/env bash
# Creates one ECR repository per image: dhaba-TEAM/<service>, eight in total.
# Safe to run again: existing repositories are left as they are.
#
# Usage:  TEAM=myteam AWS_REGION=eu-west-1 scripts/create-ecr-repos.sh
set -euo pipefail

: "${TEAM:?set TEAM (your team name)}"
: "${AWS_REGION:?set AWS_REGION}"

SERVICES=(web menu-api orders-api kitchen-worker stats-api postgres redis ops-monitor)

# Keep only the newest 20 images in each repository
LIFECYCLE_POLICY='{
  "rules": [{
    "rulePriority": 1,
    "description": "Keep only the newest 20 images",
    "selection": { "tagStatus": "any", "countType": "imageCountMoreThan", "countNumber": 20 },
    "action": { "type": "expire" }
  }]
}'

for svc in "${SERVICES[@]}"; do
    repo="dhaba-${TEAM}/${svc}"

    if out=$(aws ecr describe-repositories --region "$AWS_REGION" --repository-names "$repo" 2>&1); then
        echo "already exists: ${repo}"
    elif echo "$out" | grep -q "RepositoryNotFoundException"; then
        aws ecr create-repository \
            --region "$AWS_REGION" \
            --repository-name "$repo" \
            --image-scanning-configuration scanOnPush=true >/dev/null
        echo "created:        ${repo}"
    else
        echo "ERROR checking ${repo}: ${out}" >&2
        exit 1
    fi

    # Both calls can be repeated with the same values without any change
    aws ecr put-image-scanning-configuration \
        --region "$AWS_REGION" --repository-name "$repo" \
        --image-scanning-configuration scanOnPush=true >/dev/null
    aws ecr put-lifecycle-policy \
        --region "$AWS_REGION" --repository-name "$repo" \
        --lifecycle-policy-text "$LIFECYCLE_POLICY" >/dev/null
done
