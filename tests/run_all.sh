#!/usr/bin/env bash

fix_permissions () {
   chmod -R 777 tmp/
   if [ -f "tmp/id_rsa.pub" ] ; then
      chmod 0600 tmp/id_rsa.pub
   fi
}

cleanup() {
  local auth_method=$1

  echo "Tearing down docker containers"
  docker stop go-scp-test
  docker rm go-scp-test

  echo "Cleaning up"
  if [[ "$auth_method" == "ssh_agent" ]]; then
    ssh-add -d ./tmp/id_rsa
  fi
  rm -r tmp/
  mkdir tmp/
  touch tmp/.gitkeep
}

run_test() {
  local auth_method=$1

  fix_permissions
  echo "Testing with auth method: $auth_method"

  echo "Running tests"
  METHOD="$auth_method" go test -v
  if [ $? -ne 0 ]; then 
     cleanup
     exit 1
  fi
}

run_docker_container() {
  local enable_password_auth=$1

  docker run -d \
    --name go-scp-test \
    -p 2244:22 \
    -e SSH_USERS=bram:1000:1000 \
    -e SSH_ENABLE_PASSWORD_AUTH=$enable_password_auth \
    -v $(pwd)/tmp:/data/  \
    -v $(pwd)/data:/input  \
    -v $(pwd)/entrypoint.d/:/etc/entrypoint.d/ \
    ${extra_mount:-} \
    panubo/sshd
}

if [ -z "${GITHUB_ACTIONS}" ]; then
   AUTH_METHODS="password private_key private_key_with_passphrase ssh_agent"
else
   AUTH_METHODS="password"
fi

for auth_method in $AUTH_METHODS ; do
  case "$auth_method" in
    "password")
      echo "Testing with password auth"
      run_docker_container true
      sleep 5
      run_test "$auth_method"
      cleanup
      ;;
    "private_key" | "private_key_with_passphrase" | "ssh_agent")
      echo "Testing with $auth_method auth"
      ssh-keygen -t rsa -f ./tmp/id_rsa -N ""
      if [[ "$auth_method" == "private_key_with_passphrase" ]]; then
        ssh-keygen -p -f ./tmp/id_rsa -P "" -N "passphrase"
      fi
      if [[ "$auth_method" == "ssh_agent" ]]; then
        ssh-add ./tmp/id_rsa
      fi
      extra_mount="-v $(pwd)/tmp/id_rsa.pub:/etc/authorized_keys/bram:ro"
      run_docker_container false
      sleep 5
      run_test "$auth_method"
      cleanup "$auth_method"
      ;;
    *)
      echo "Unsupported auth method $auth_method"
      exit 1
      ;;
  esac
done
