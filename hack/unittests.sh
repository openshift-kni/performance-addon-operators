#!/usr/bin/env bash

set -e

go_tests() {
  OUTDIR="build/_output/coverage"
  mkdir -p "$OUTDIR"

  COVER_FILE="${OUTDIR}/cover.out"
  FUNC_FILE="${OUTDIR}/coverage.txt"
  HTML_FILE="${OUTDIR}/coverage.html"

  echo "running unittests with coverage"
  GOFLAGS=-mod=vendor go test -race -covermode=atomic -coverprofile="${COVER_FILE}" -v ./pkg/... ./controllers/... ./api/...

  if [[ -n "${DRONE}" ]]; then

    # Uploading coverage report to coveralls.io
    go get github.com/mattn/goveralls

    # we should update the vendor/modules.txt once we got a new package
    go mod vendor
    $(go env GOPATH)/bin/goveralls -coverprofile="$COVER_FILE" -service=drone.io

  else

    echo "creating coverage reports"
    go tool cover -func="${COVER_FILE}" > "${FUNC_FILE}"
    go tool cover -html="${COVER_FILE}" -o "${HTML_FILE}"
    echo "find coverage reports at ${OUTDIR}"

  fi
}

bash_tests() {
  echo "running asset script unit tests"
  ASSETDIR="build/assets/scripts"
  pushd $ASSETDIR >/dev/null
  shopt -qs nullglob
  TESTFILES="*_test.sh"
  shopt -qu nullglob
  FAILURES=0
  for f in $TESTFILES; do
    TMPOUT=$(mktemp)
    echo -n "Testing $f... "
    set +e # Hide any failure or output for now, for better error reporting
      ./$f &>$TMPOUT
      RESULT=$?
    set -e
    if [[ $RESULT -eq 0 ]]; then
      echo "Success"
    else
      ((FAILURES += 1))
      echo "FAILED!"
      echo "---------------------------------------------------------------"
      cat $TMPOUT
      echo "---------------------------------------------------------------"
    fi
    rm $TMPOUT
  done
  popd >/dev/null
  if [[ $FAILURES -gt 0 ]]; then
    exit 1
  fi
}

go_tests
bash_tests
