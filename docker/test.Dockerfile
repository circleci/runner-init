FROM busybox:stable-musl as build

ARG ARCH

COPY ./target/bin/${ARCH}/orchestrator /opt/circleci/bin/orchestrator
COPY ./target/bin/${ARCH}/fake-task-agent /opt/circleci/bin/circleci-agent
COPY ./docker/scripts/init.sh /init.sh

ENTRYPOINT ["/init.sh"]
