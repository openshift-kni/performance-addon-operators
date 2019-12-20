#!/bin/sh

echo "waiting for updating..."
# waiting only once since many manifests may be applied at the same time
oc wait mcp/worker --for condition=updating --timeout 2m
echo "waiting for updated..."
until oc wait mcp/worker --for condition=updated --timeout 600s ; do sleep 1 ; done