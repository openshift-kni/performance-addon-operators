#!/bin/bash

# expect oc to be in PATH by default
OC_TOOL="${OC_TOOL:-oc}"

$OC_TOOL delete performanceprofile --all

$OC_TOOL delete ns openshift-performance-addon

$OC_TOOL -n openshift-marketplace delete catalogsource performance-addon-operator-catalogsource

$OC_TOOL delete mcp worker-rt

