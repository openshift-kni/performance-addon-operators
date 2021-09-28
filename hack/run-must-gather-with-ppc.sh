#!/bin/bash

# expect oc to be in PATH by default
OC_TOOL="${OC_TOOL:-oc}"
CONTAINER_RUNTIME="${CONTAINER_RUNTIME:-podman}"
# Current Mustgather version
VERSION="4.9-snapshot"
# Must gather image url
MUST_GATHER_IMG="quay.io/openshift-kni/performance-addon-operator-must-gather:${VERSION}"
TIMESTAMP="$(date +%a-%b-%d-%H%M%S)"
MUST_GATHER_VOL="/tmp/mustgather-$VERSION-$TIMESTAMP"
PROFILE="/tmp/performance_profile.yaml"
PODMAN_CMD="${CONTAINER_RUNTIME} run --entrypoint performance-profile-creator"
PAO_IMG="quay.io/openshift-kni/performance-addon-operator:${VERSION}"
OC_CMD="oc adm must-gather --image=${MUST_GATHER_IMG} --dest-dir=${MUST_GATHER_VOL}"
PPC_OPTIONS="--rt-kernel --mcp-name worker-cnf --reserved-cpu-count 4 --split-reserved-cpus-across-numa true"
#TOOLS_BIN_DIR="build/_output/bin"
#PC_BINARY="../../build/_output/bin/performance-profile-creator"
#PPC_CMD="${PPC_BINARY} --must-gather-dir-path ${MUST_GATHER_VOL} $PPC_OPTIONS"
PPC_CMD="${PODMAN_CMD} -v ${MUST_GATHER_VOL}:${MUST_GATHER_VOL}:z ${PAO_IMG} $PPC_OPTIONS --must-gather-dir-path ${MUST_GATHER_VOL}"
YUM_CMD="yum -y install podman"

collect_mustgather()
{
#collect must gather from
echo -e "Executing ${OC_CMD} command"
${OC_CMD}

if [ $? != 0 ]
then
    echo "${OC_CMD} failed"
fi
return 0
}

create_performance_profile()
{

if  [ -d "${MUST_GATHER_VOL}" ]
then
    echo "Execute ${PPC_CMD}"
    ${PPC_CMD} > $PROFILE
    echo -e "$PROFILE created"
    if [ $? != 0 ]
    then
        echo "${PPC_CMD} failed"
    fi
fi
return 0

}

apply_performance_profile()
{
    CMD="${OC_TOOL} create -f $PROFILE"
    if [ -f "$PROFILE" ];
    then
        echo "Executing command ${CMD}"
        ${CMD}
        if [ $? != 0 ]
        then
            echo "${CMD} failed"
        fi
    else
        echo "$PROFILE doesn't exist"
    fi
}

Help()
{
    # Script Usage
    echo "Generate Must gather using PAO Image and Generate profile using Performance Profile creator"
    echo
    echo "Syntax: $(basename $0) -mustgather|creatprofile|h"
    echo "options:"
    echo "mustgather          Generate mustgather"
    echo "createprofile       Generate mustgather and create Performance Profile"
    echo "applyprofile        Apply the performance profile created"
    echo "h                   Print this help"
}


main()
{
	while [ "$1" != "" ]; do
		case $1 in
		    -h | --help )
                        Help
                        exit 0
                        ;;
                    -mustgather )
                        collect_mustgather
                        ;;
                    -createprofile )
			collect_mustgather
                        create_performance_profile
                        ;;
                    -applyprofile )
                        apply_performance_profile
                        ;;
                    * )
                        Help
                        exit 0
                esac
                shift
        done
}
main "$@"
