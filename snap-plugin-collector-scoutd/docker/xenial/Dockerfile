FROM ubuntu:16.04

ARG BUILD_DATE
ARG SNAP_VERSION=latest

ENV SNAP_VERSION=${SNAP_VERSION}
ENV SNAP_TRUST_LEVEL=0
ENV SNAP_LOG_LEVEL=1
ENV CI_URL=https://s3-us-west-2.amazonaws.com/snap.ci.snap-telemetry.io

LABEL vendor="solarwinds" \
      name="Snap Ubuntu 16.04" \
      build-date=$BUILD_DATE \
      io.snap-telemetry.snap.version=$SNAP_VERSION \
      io.snap-telemetry.snap.version.is-beta=

EXPOSE 8181

ADD ${CI_URL}/snap/${SNAP_VERSION}/linux/x86_64/snapteld  /opt/snap/sbin/snapteld
ADD ${CI_URL}/snap/${SNAP_VERSION}/linux/x86_64/snaptel  /opt/snap/bin/snaptel
COPY init_snap /usr/local/bin/init_snap
COPY snapteld.conf /etc/snap/snapteld.conf
RUN chmod +x /usr/local/bin/init_snap

# golang
ADD https://storage.googleapis.com/golang/go1.8.3.linux-amd64.tar.gz /usr/local
RUN echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile

# additional tools and binaries
RUN apt-get update && apt-get -y install ruby tmux sudo

CMD /usr/local/bin/init_snap && /opt/snap/sbin/snapteld -t ${SNAP_TRUST_LEVEL} -l ${SNAP_LOG_LEVEL} -o ''
