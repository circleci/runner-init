#!/usr/bin/env bash

set -eu -o pipefail

# This variable is used, but shellcheck can't tell.
# shellcheck disable=SC2034
help_download_taskagent="Download task agents via the picard image"
download-taskagents() {
    id=$(docker create circleci/picard:agent)

    docker cp "$id":/opt/circleci/linux/amd64/circleci-agent ./bin/circleci-agent-amd64
    docker cp "$id":/opt/circleci/linux/arm64/circleci-agent ./bin/circleci-agent-arm64

    docker rm -v "$id"
}

# This variable is used, but shellcheck can't tell.
# shellcheck disable=SC2034
help_build_fake_agents="Build the fake agent go binaries"
build-fake-agents() {
    GOOS=linux GOARCH=amd64 go build -C ./fake-agent -o ../bin/circleci-fake-agent-amd64 ./
    GOOS=linux GOARCH=arm64 go build -C ./fake-agent -o ../bin/circleci-fake-agent-arm64 ./
}

# This variable is used, but shellcheck can't tell.
# shellcheck disable=SC2034
help_build_docker_images="Build the runner init images"
build-docker-images() {
    repo=${1:?'image repo name must be specified'}
    arch=${2:?'image arch must be specified'}

    docker build -t circleci/"$repo":agent-"$arch" --build-arg ARCH="$arch" -f ./runner-init/Dockerfile .
    docker build -t circleci/"$repo":test-agent-"$arch" --build-arg ARCH="$arch" -f ./runner-init/fake-agent.Dockerfile .
}

# This variable is used, but shellcheck can't tell.
# shellcheck disable=SC2034
help_publish_docker_images="Publish the runner init images"
publish-docker-images() {
    repo=${1:?'image repo name must be specified'}
    arch=${2:?'image arch must be specified'}

    docker push circleci/"$repo":agent-"$arch"
    docker push circleci/"$repo":test-agent-"$arch"
}

# This variable is used, but shellcheck can't tell.
# shellcheck disable=SC2034
help_publish_docker_manifest="Publish the mutliarch manifest"
publish-docker-manifest() {
    repo=${1:?'image repo name must be specified'}

    docker manifest create circleci/runner-init:agent \
        --amend circleci/runner-init:agent-amd64 \
        --amend circleci/runner-init:agent-arm64

    docker manifest push circleci/runner-init:agent

    docker manifest create circleci/runner-init:test-agent \
        --amend circleci/runner-init:test-agent-amd64 \
        --amend circleci/runner-init:test-agent-arm64

    docker manifest push circleci/runner-init:test-agent
}

help-text-intro() {
    echo "
DO

A set of simple repetitive tasks that adds minimally
to standard tools used to build and test the service.
(e.g. go and docker)
"
}

### START FRAMEWORK ###
# Do Version 0.0.4

# This variable is used, but shellcheck can't tell.
# shellcheck disable=SC2034
help_completion="Print shell completion function for this script.

Usage: $0 completion SHELL"
completion() {
    local shell
    shell="${1-}"

    if [ -z "$shell" ]; then
        echo "Usage: $0 completion SHELL" 1>&2
        exit 1
    fi

    case "$shell" in
    bash)
        (
            echo
            echo '_dotslashdo_completions() { '
            # shellcheck disable=SC2016
            echo '  COMPREPLY=($(compgen -W "$('"$0"' list)" "${COMP_WORDS[1]}"))'
            echo '}'
            echo 'complete -F _dotslashdo_completions '"$0"
        )
        ;;
    zsh)
        cat <<EOF
_dotslashdo_completions() {
  local -a subcmds
  subcmds=()
  DO_HELP_SKIP_INTRO=1 $0 help | while read line; do
EOF
        cat <<'EOF'
    cmd=$(cut -f1  <<< $line)
    cmd=$(awk '{$1=$1};1' <<< $cmd)

    desc=$(cut -f2- <<< $line)
    desc=$(awk '{$1=$1};1' <<< $desc)

    subcmds+=("$cmd:$desc")
  done
  _describe 'do' subcmds
}

compdef _dotslashdo_completions do
EOF
        ;;
    fish)
        cat <<EOF
complete -e -c do
complete -f -c do
for line in (string split \n (DO_HELP_SKIP_INTRO=1 $0 help))
EOF
        cat <<'EOF'
  set cmd (string split \t $line)
  complete -c do  -a $cmd[1] -d $cmd[2]
end
EOF
        ;;
    esac
}

list() {
    declare -F | awk '{print $3}'
}

# This variable is used, but shellcheck can't tell.
# shellcheck disable=SC2034
help_help="Print help text, or detailed help for a task."
help() {
    local item
    item="${1-}"
    if [ -n "${item}" ]; then
        local help_name
        help_name="help_${item//-/_}"
        echo "${!help_name-}"
        return
    fi

    if [ -z "${DO_HELP_SKIP_INTRO-}" ]; then
        type -t help-text-intro >/dev/null && help-text-intro
    fi
    for item in $(list); do
        local help_name text
        help_name="help_${item//-/_}"
        text="${!help_name-}"
        [ -n "$text" ] && printf "%-30s\t%s\n" "$item" "$(echo "$text" | head -1)"
    done
}

case "${1-}" in
list) list ;;
"" | "help") help "${2-}" ;;
*)
    if ! declare -F "${1}" >/dev/null; then
        printf "Unknown target: %s\n\n" "${1}"
        help
        exit 1
    else
        "$@"
    fi
    ;;
esac
### END FRAMEWORK ###
