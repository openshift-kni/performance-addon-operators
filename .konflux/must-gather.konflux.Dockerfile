FROM registry.redhat.io/openshift4/ose-must-gather:v4.12@sha256:b8286a441e352c026e4b8803188dc4f3b51857e8ef60f332836e0bfd24c67a3e AS mgbuilder

WORKDIR /
COPY . .

# Save original gather script
RUN mv /usr/bin/gather /usr/bin/gather_original

RUN mkdir -p /usr/libexec/must-gather/performance-addon-operator && \
    cp /must-gather/collection-scripts/* /usr/libexec/must-gather/performance-addon-operator/
RUN mkdir -p /etc/node-gather && \
    cp /must-gather/node-gather/* /etc/node-gather/

FROM registry.redhat.io/openshift/golang-builder:golang-builder-v1.19-rhel8@sha256:9de5282b1ad80c0bc857b889c130a9018757f54200e35960ae343c052d535f13 AS paobuilder

WORKDIR /app
COPY . .
RUN make dist-gather-sysinfo

FROM registry.redhat.io/ubi8/ubi-minimal:latest@sha256:62d3baa1ca1cdc4613ad79521b21bdef5baa81ea9167441c731acef18b8253cf

RUN microdnf install -y pciutils util-linux hostname tar rsync ethtool ; microdnf clean all

# Copy must-gather required binaries
COPY --from=mgbuilder /usr/bin/openshift-must-gather /usr/bin/openshift-must-gather
COPY --from=mgbuilder /usr/bin/oc /usr/bin/oc

COPY --from=mgbuilder /usr/libexec/must-gather/performance-addon-operator/* /usr/bin/
COPY --from=mgbuilder /etc/node-gather/* /etc/node-gather/

# the binary follows the must-gather convention of helper tools called gather_$SOMETHING
COPY --from=paobuilder /app/build/_output/bin/gather-sysinfo /usr/bin/gather_sysinfo

ENTRYPOINT ["/usr/bin/gather"]

LABEL com.redhat.component="performance-addon-operator-must-gather-rhel8-container" \
      name="openshift4/performance-addon-operator-must-gather-rhel8" \
      version="v4.12.15" \
      summary="performance-addon-operator data gathering image" \
      io.openshift.expose-services="" \
      io.openshift.tags="data,images" \
      io.k8s.description="performance-addon-operator data gathering image." \
      io.k8s.display-name="performance-addon-operator-must-gather" \
      description="performance-addon-operator data gathering image." \
      maintainer="openshift-operators@redhat.com" \
      io.openshift.maintainer.component="Performance Addon Operator" \
      io.openshift.maintainer.product="OpenShift Container Platform" \
      distribution-scope="public" \
      release="v4.12.15" \
      url="https://github.com/openshift-kni/performance-addon-operator" \
      vendor="Red Hat, Inc." \
      cpe="cpe:/a:redhat:openshift:4.12::el8"
