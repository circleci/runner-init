#!/usr/bin/env bash

set -eu -o pipefail

reportDir="test-reports"

# This variable is used, but shellcheck can't tell.
# shellcheck disable=SC2034
help_build="Build the Go binaries and archives"
build() {
    set -x

    BUILD_VERSION="${BUILD_VERSION:-dev}" go tool goreleaser \
        --clean \
        --config "${BUILD_CONFIG:-./.goreleaser/binaries.yaml}" \
        --skip=validate "$@"

    echo "${BUILD_VERSION:-dev}" | tee ./target/version.txt
}

help_license_attributions="Regenerate the third-party license attributions file."
license-attributions() {
  go tool go-licenses report --ignore=gotest  ./... \
    --template ./templates/licenses-csv.tpl >${LICENSE_ATTRIBUTIONS_FILE:-./go-project-licenses.csv}
}

# This variable is used, but shellcheck can't tell.
# shellcheck disable=SC2034
help_images="Build and push the Docker images and manifests."
images() {
    set -x

    skip="${SKIP_PUSH:-true}"
    SKIP_PUSH="${skip}" \
        SKIP_PUSH_TEST_AGENT="${SKIP_PUSH_TEST_AGENT:-${skip}}" \
        IMAGE_TAG_SUFFIX="${IMAGE_TAG_SUFFIX:-""}" \
        PICARD_VERSION="${PICARD_VERSION:-agent}" \
        go tool goreleaser \
        --clean \
        --config "${BUILD_CONFIG:-./.goreleaser/dockers.yaml}" \
        --skip=validate "$@"
}

# This variable is used, but shellcheck can't tell.
# shellcheck disable=SC2034
help_images_for_server="Build and push the Docker images and manifests for supported server versions."
images-for-server() {
    MAJOR_SERVER_VERSION=4
    MINOR_VERSION_START=7
    MINOR_VERSION_END=9

    for minor in $(seq ${MINOR_VERSION_START} ${MINOR_VERSION_END}); do
        if [ "${minor}" -eq "${MINOR_VERSION_END}" ]; then
            branch="main"
        else
            branch="server-${MAJOR_SERVER_VERSION}.${minor}"
        fi

        git -C "${SERVER_REPO_PATH:?'server repo path required'}" checkout "${branch}"

        picard_version="$(yq .circleci/picard -r "${SERVER_REPO_PATH}/images.yaml")"
        echo "Building for build-agent version ${picard_version}"

        PICARD_VERSION=${picard_version} \
            IMAGE_TAG_SUFFIX="-server-${MAJOR_SERVER_VERSION}.${minor}" \
            SKIP_PUSH_TEST_AGENT='true' \
            ./do images
    done
}

# This variable is used, but shellcheck can't tell.
# shellcheck disable=SC2034
help_lint="Run golanci-lint to lint go files."
lint() {
    eval "go tool golangci-lint run ${1:-}"
}

# This variable is used, but shellcheck can't tell.
# shellcheck disable=SC2034
help_lint_report="Run golanci-lint to lint Go files and generate an XML report."
lint-report() {
    output="${reportDir}/lint.xml"
    echo "Storing results as JUnit XML in ${output}" >&2
    mkdir -p "${reportDir}"

    lint "--timeout 5m --output.junit-xml.path stdout | tee ${output}"
}

# This variable is used, but shellcheck can't tell.
# shellcheck disable=SC2034
help_test="Run the tests"
test() {
    mkdir -p "${reportDir}"
    # -count=1 is used to forcibly disable test result caching
    go tool gotestsum --junitfile="${reportDir}/junit.xml" -- -race -count=1 "${@:-./...}"
}

# This variable is used, but shellcheck can't tell.
# shellcheck disable=SC2034
help_smoke_tests="Run the smoke tests"
smoke-tests() {
    GOTESTSUM_FORMAT="${GOTESTSUM_FORMAT:-standard-verbose}" \
        test -test.timeout=20m -tags smoke ./internal/testing/ci/smoke
}

# This variable is used, but shellcheck can't tell.
# shellcheck disable=SC2034
help_go_mod_tidy="Run 'go mod tidy' to clean up module files."
go-mod-tidy() {
    go mod tidy -v
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
