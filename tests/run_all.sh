#!/usr/bin/env bash
#
# Runs the integration tests against an SSH server, once per auth method.
#
# Everything runs in rootless podman, inside a single pod so the client and
# server share a network namespace (the tests connect to 127.0.0.1:2244):
#   - server: panubo/sshd, listening on port 2244
#   - client: the golang image, running `go test` (and, for the ssh_agent
#     method, its own ssh-agent)
#
# Can be run from any directory, e.g. `./tests/run_all.sh` from the repo root.
#
# Environment overrides:
#   AUTH_METHODS  space-separated list of methods to test
#   SERVER_IMAGE  SSH server image
#   CLIENT_IMAGE  Go image used to run the tests
#   GO_TEST_ARGS  extra arguments for `go test`, e.g. "-run TestCopy"

set -u

TESTS_DIR=$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")
REPO_ROOT=$(dirname "$TESTS_DIR")
cd "$TESTS_DIR" || exit 1

POD=go-scp-test
SERVER=go-scp-test-server
GO_VERSION=$(awk '/^go /{ split($2, v, "."); print v[1] "." v[2] }' "$REPO_ROOT/go.mod")
SERVER_IMAGE=${SERVER_IMAGE:-docker.io/panubo/sshd:latest}
CLIENT_IMAGE=${CLIENT_IMAGE:-docker.io/library/golang:$GO_VERSION}

if [ -z "${AUTH_METHODS:-}" ]; then
   if [ -z "${GITHUB_ACTIONS:-}" ]; then
      AUTH_METHODS="password private_key private_key_with_passphrase ssh_agent"
   else
      AUTH_METHODS="password"
   fi
fi

reset_tmp() {
  # Files written by the SSH user are owned by a subordinate uid on the host,
  # so remove them from inside podman's user namespace.
  podman unshare rm -rf tmp/
  mkdir tmp/
  touch tmp/.gitkeep
}

cleanup() {
  echo "Tearing down pod"
  podman pod rm -f --ignore "$POD" >/dev/null
  echo "Cleaning up"
  reset_tmp
}

fix_permissions() {
   # The SSH user inside the server container is not mapped to the host user.
   chmod -R 777 tmp/
   if [ -f "tmp/id_rsa.pub" ] ; then
      # Mounted as /etc/authorized_keys/bram, where it appears root-owned; sshd
      # reads it as bram, so it must be world-readable (but not writable).
      chmod 0644 tmp/id_rsa.pub
   fi
}

# Runs a bash script in the client container, inside the pod.
# The container's root user is the host user, so anything it writes to the
# repository is owned by the host user.
run_client() {
  local auth_method=$1 script=$2
  podman run --rm \
    --pod "$POD" \
    -v "$REPO_ROOT":/src \
    -w /src/tests \
    -v go-scp-test-gomod:/go/pkg/mod \
    -v go-scp-test-gocache:/root/.cache/go-build \
    -e METHOD="$auth_method" \
    "$CLIENT_IMAGE" \
    bash -c "$script"
}

start_server() {
  local enable_password_auth=$1; shift

  podman run -d \
    --pod "$POD" \
    --name "$SERVER" \
    --log-driver k8s-file \
    -e SSH_USERS=bram:1000:1000 \
    -e SSH_ENABLE_PASSWORD_AUTH="$enable_password_auth" \
    -v "$TESTS_DIR/tmp":/data/ \
    -v "$TESTS_DIR/data":/input \
    -v "$TESTS_DIR/entrypoint.d":/etc/entrypoint.d/ \
    "$@" \
    "$SERVER_IMAGE" \
    /usr/sbin/sshd -D -e -f /etc/ssh/sshd_config -p 2244 >/dev/null || return 1

  # Wait until sshd accepts connections (and entrypoint.d has run).
  for _ in $(seq 1 60); do
    if podman logs "$SERVER" 2>&1 | grep -q "Server listening"; then
      return 0
    fi
    if [ "$(podman inspect -f '{{.State.Running}}' "$SERVER")" != "true" ]; then
      break
    fi
    sleep 0.5
  done
  echo "SSH server did not start:"
  podman logs "$SERVER"
  return 1
}

run_test() {
  local auth_method=$1 setup=$2

  fix_permissions
  echo "Running tests with auth method: $auth_method"
  run_client "$auth_method" "set -e; $setup go test -v ${GO_TEST_ARGS:-}"
}

trap cleanup EXIT

podman pod rm -f --ignore "$POD" >/dev/null
reset_tmp

for auth_method in $AUTH_METHODS ; do
  podman pod create --name "$POD" >/dev/null || exit 1

  case "$auth_method" in
    "password")
      echo "Testing with password auth"
      start_server true || exit 1
      run_test "$auth_method" "" || exit 1
      ;;
    "private_key" | "private_key_with_passphrase" | "ssh_agent")
      echo "Testing with $auth_method auth"
      passphrase=""
      if [[ "$auth_method" == "private_key_with_passphrase" ]]; then
        passphrase="passphrase"
      fi
      run_client "$auth_method" \
        "ssh-keygen -q -t rsa -f ./tmp/id_rsa -N '$passphrase'" || exit 1

      setup=""
      if [[ "$auth_method" == "ssh_agent" ]]; then
        # Agent lives and dies with the client container.
        # ssh-add refuses keys that fix_permissions made world-readable.
        setup='chmod 600 ./tmp/id_rsa; eval "$(ssh-agent -s)" >/dev/null; ssh-add ./tmp/id_rsa;'
      fi

      start_server false \
        -v "$TESTS_DIR/tmp/id_rsa.pub":/etc/authorized_keys/bram:ro || exit 1
      run_test "$auth_method" "$setup" || exit 1
      ;;
    *)
      echo "Unsupported auth method $auth_method"
      exit 1
      ;;
  esac

  podman pod rm -f "$POD" >/dev/null
  reset_tmp
done
