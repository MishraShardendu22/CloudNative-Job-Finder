#!/usr/bin/env bash
set -euo pipefail

ACTION="${1:-release}"
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

NAMESPACE="${DOCKERHUB_NAMESPACE:-shardendumishra22}"
VERSION_TAG="${IMAGE_TAG:-v2}"
SHA_TAG="${IMAGE_TAG_SHA:-$(git rev-parse --short HEAD 2>/dev/null || date +%Y%m%d%H%M%S)}"

SERVICES=(
  api-gateway
  user-service
  resume-service
  job-scraper
  job-processor
  job-matcher
  recommendation-service
  scheduler
  outbox-relay
  feature-aggregator
  resume-parser
  frontend
)

usage() {
  cat <<EOF
Usage: $(basename "$0") [login|build|push|release]

Actions:
  login    Perform Docker Hub login (interactive or token-based)
  build    Build and tag all service images
  push     Push all tagged images
  release  Login, then build and push all images

Environment variables:
  DOCKERHUB_NAMESPACE (default: shardendumishra22)
  IMAGE_TAG           (default: v2)
  IMAGE_TAG_SHA       (default: short git SHA)
  DOCKERHUB_USERNAME  (optional, for non-interactive login)
  DOCKERHUB_TOKEN     (optional, for non-interactive login)
EOF
}

dockerfile_for() {
  case "$1" in
    api-gateway) echo "services/api-gateway/Dockerfile" ;;
    user-service) echo "services/user-service/Dockerfile" ;;
    resume-service) echo "services/resume-service/Dockerfile" ;;
    job-scraper) echo "services/job-scraper/Dockerfile" ;;
    job-processor) echo "services/job-processor/Dockerfile" ;;
    job-matcher) echo "services/job-matcher/Dockerfile" ;;
    recommendation-service) echo "services/recommendation-service/Dockerfile" ;;
    scheduler) echo "services/scheduler/Dockerfile" ;;
    outbox-relay) echo "services/outbox-relay/Dockerfile" ;;
    feature-aggregator) echo "services/feature-aggregator/Dockerfile" ;;
    resume-parser) echo "services/resume-parser/Dockerfile" ;;
    frontend) echo "frontend/Dockerfile" ;;
    *)
      echo "unknown service: $1" >&2
      exit 1
      ;;
  esac
}

context_for() {
  case "$1" in
    frontend) echo "frontend" ;;
    *) echo "." ;;
  esac
}

ensure_login() {
  if [[ -n "${DOCKERHUB_USERNAME:-}" && -n "${DOCKERHUB_TOKEN:-}" ]]; then
    echo "$DOCKERHUB_TOKEN" | docker login --username "$DOCKERHUB_USERNAME" --password-stdin
    return
  fi

  if [[ -t 0 ]]; then
    docker login
    return
  fi

  echo "Docker Hub login required. Set DOCKERHUB_USERNAME and DOCKERHUB_TOKEN for non-interactive login." >&2
  exit 1
}

unique_tags() {
  local seen_latest=0
  local seen_version=0
  local seen_sha=0

  if [[ "latest" != "" ]]; then
    echo "latest"
    seen_latest=1
  fi

  if [[ "$VERSION_TAG" != "" && "$VERSION_TAG" != "latest" ]]; then
    echo "$VERSION_TAG"
    seen_version=1
  fi

  if [[ "$SHA_TAG" != "" && "$SHA_TAG" != "latest" && "$SHA_TAG" != "$VERSION_TAG" ]]; then
    echo "$SHA_TAG"
    seen_sha=1
  fi

  # keep shellcheck quiet for intentionally tracked local flags
  : "$seen_latest" "$seen_version" "$seen_sha"
}

build_service() {
  local service="$1"
  local image="${NAMESPACE}/${service}"
  local dockerfile
  local context
  dockerfile="$(dockerfile_for "$service")"
  context="$(context_for "$service")"

  local tags=()
  while IFS= read -r tag; do
    tags+=("$tag")
  done < <(unique_tags)

  if [[ ${#tags[@]} -eq 0 ]]; then
    echo "No tags resolved for $service" >&2
    exit 1
  fi

  echo "Building ${image}:${tags[0]} from ${dockerfile}"
  docker build --pull -f "$dockerfile" -t "${image}:${tags[0]}" "$context"

  for tag in "${tags[@]:1}"; do
    echo "Tagging ${image}:${tag}"
    docker tag "${image}:${tags[0]}" "${image}:${tag}"
  done
}

push_service() {
  local service="$1"
  local image="${NAMESPACE}/${service}"

  while IFS= read -r tag; do
    echo "Pushing ${image}:${tag}"
    docker push "${image}:${tag}"
  done < <(unique_tags)
}

build_all() {
  for service in "${SERVICES[@]}"; do
    build_service "$service"
  done
}

push_all() {
  for service in "${SERVICES[@]}"; do
    push_service "$service"
  done
}

case "$ACTION" in
  login)
    ensure_login
    ;;
  build)
    build_all
    ;;
  push)
    ensure_login
    push_all
    ;;
  release)
    ensure_login
    build_all
    push_all
    ;;
  *)
    usage
    exit 1
    ;;
esac
