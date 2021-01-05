#!/usr/bin/env bash
if [ "{{.CPUList}}" == "UNRESTRICTED" ]; then
	exec /bin/runc "$@"
fi
if [ -n "$3" ] && [ "$3" == "start" ]; then
        if [ -n "$4" ]; then
                bundle=$(/bin/runc state "$4" 2>/dev/null | jq .bundle -r)
                if [ -n "$bundle" -a -f "$bundle/config.json" ]; then
                        command=$(grep "/usr/bin/pod" "$bundle/config.json")
                        if [ -n "$command" ]; then
                                cpuset=$(find /sys/fs/cgroup/cpuset/ -name cpuset.cpus | grep $4)
                                echo "{{.CPUList}}" > $cpuset
                        fi
                fi
        fi
fi
taskset -c "{{.CPUList}}" /bin/runc "$@"
