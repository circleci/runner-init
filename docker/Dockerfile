ARG PICARD_VERSION=agent

FROM circleci/picard:${PICARD_VERSION} AS task-agent-image

FROM scratch AS builder

ARG ARCH

COPY --from=task-agent-image /opt/circleci/linux/${ARCH}/circleci-agent /
COPY ./target/bin/${ARCH}/orchestrator /

FROM scratch

COPY --from=builder / /

ENTRYPOINT ["/orchestrator", "init"]
