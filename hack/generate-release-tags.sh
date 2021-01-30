#!/bin/bash

# This script generates a release.tags file containing all tags which should be build and pushed
# based on the git tag

TAGS_FILE=".tags"

# Current tag
TAG=${TAG:-$(git describe --abbrev=0 --tags)}
echo "using tag: $TAG"

# We expect tags with pattern major.minor.patch-build[.prelease]
regex='([[:digit:]]+).([[:digit:]]+).([[:digit:]]+)-([[:digit:]]+).?(.*)?'
if ! [[ $TAG =~ $regex ]]
then
  echo "Invalid version, it should be in format major.minor.patch-build[.prelease]"
  exit 1
fi

MAJOR=${BASH_REMATCH[1]}
MINOR=${BASH_REMATCH[2]}
PATCH=${BASH_REMATCH[3]}
BUILD=${BASH_REMATCH[4]}
PRE=${BASH_REMATCH[5]}


echo "major: ${MAJOR}"
echo "minor: ${MINOR}"
echo "patch: ${PATCH}"
echo "build: ${BUILD}"
echo "pre: ${PRE}"

# We always build the complete tag
echo -n "$TAG" > "$TAGS_FILE"

# If pre is empty, we also build major.minor.patch
if [ -z "$PRE" ]; then
  echo ",${MAJOR}.${MINOR}.${PATCH}" >> "$TAGS_FILE"
fi
