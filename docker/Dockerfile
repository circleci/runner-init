FROM circleci/picard:agent as task-agent-image

FROM busybox:stable-musl

ARG ARCH

COPY --from=task-agent-image /opt/circleci/linux/${ARCH}/circleci-agent /opt/circleci/bin/circleci-agent

COPY ./target/bin/${ARCH}/orchestrator /opt/circleci/bin/orchestrator
COPY ./docker/scripts/init.sh /init.sh

ENTRYPOINT ["/init.sh"]
