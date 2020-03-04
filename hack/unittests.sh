#!/usr/bin/env bash

set -e

OUTDIR="build/_output/coverage"
mkdir -p "$OUTDIR"

COVER_FILE="${OUTDIR}/cover.out"
FUNC_FILE="${OUTDIR}/coverage.txt"
HTML_FILE="${OUTDIR}/coverage.html"

echo "running unittests with coverage"

GOFLAGS=-mod=vendor go test -race -covermode=atomic -coverprofile="${COVER_FILE}" -v ./pkg/...

if [ -n "${CI}" ]; then

  # see https://github.com/openshift/release/pull/7863
  # and TODO add PR with new job config)
  #COVERALLS_TOKEN=$(cat /usr/local/coveralls-token)

  #go get github.com/mattn/goveralls
  if [ -n "${PULL_NUMBER}" ]; then
    echo "pushing presubmit coverage"
    #PULL_REQUEST_NUMBER="$PULL_NUMBER" GIT_BRANCH="PR-${PULL_NUMBER}" $(go env GOPATH)/bin/goveralls -coverprofile="$COVER_FILE" -service=Prow -repotoken="$COVERALLS_TOKEN"
  else
    echo "pushing postsubmit coverage"
    #GIT_BRANCH="$PULL_BASE_REF" $(go env GOPATH)/bin/goveralls -coverprofile="$COVER_FILE" -service=Prow -repotoken="$COVERALLS_TOKEN"
  fi

else

  echo "creating coverage reports"
  go tool cover -func="${COVER_FILE}" > "${FUNC_FILE}"
  go tool cover -html="${COVER_FILE}" -o "${HTML_FILE}"
  echo "find coverage reports at ${OUTDIR}"

fi
