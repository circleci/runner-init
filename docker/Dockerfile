ARG PICARD_VERSION=agent

FROM circleci/picard:${PICARD_VERSION} AS task-agent-image

FROM scratch AS builder

ARG TARGETPLATFORM

COPY --from=task-agent-image /opt/circleci/${TARGETPLATFORM}/circleci-agent /
COPY ./target/bin/${TARGETPLATFORM}/orchestrator /

FROM scratch

COPY --from=builder / /

ENTRYPOINT ["/orchestrator", "init"]
