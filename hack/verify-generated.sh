#!/bin/bash

if [[ -n "$(git status --porcelain pkg/apis deploy)" ]]; then
        echo "uncommitted generated files. run 'make generate' and commit results."
        echo "$(git status --porcelain pkg/apis deploy/crds deploy/olm-catalog)"
        exit 1
fi
