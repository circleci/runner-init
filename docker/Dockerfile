ARG PICARD_VERSION=agent

FROM circleci/picard:${PICARD_VERSION} AS task-agent-image

FROM scratch AS builder

ARG ARCH

COPY --from=task-agent-image /opt/circleci/linux/${ARCH}/circleci-agent /

FROM --platform=linux/amd64 alpine AS compressor
COPY --from=builder /circleci-agent /
RUN apk add --no-cache lz4 \
    && lz4 /circleci-agent /circleci-agent.lz4

FROM scratch AS files
ARG ARCH
COPY --from=compressor /circleci-agent.lz4 /
COPY ./target/bin/${ARCH}/orchestrator /

FROM scratch

COPY --from=files / /

ENTRYPOINT ["/orchestrator", "init"]
