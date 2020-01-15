#!/bin/bash

if [[ -n "$(git status --porcelain pkg/apis deploy/crds)" ]]; then
        echo "uncommitted generated files. run 'make generate' and commit results."
        exit 1
fi
