FROM busybox:stable-musl as build

ARG ARCH=amd64

COPY ./bin/circleci-fake-agent-${ARCH} /opt/circleci/bin/circleci-agent
COPY ./runner-init/init.sh /init.sh

ENTRYPOINT ["/init.sh"]
