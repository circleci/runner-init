FROM busybox:stable-musl

ARG ARCH=amd64

COPY ./bin/circleci-agent-${ARCH} /opt/circleci/circleci-agent
COPY ./runner-init/init.sh /init.sh

ENTRYPOINT ["/init.sh"]