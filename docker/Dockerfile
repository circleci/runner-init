FROM circleci/picard:agent as task-agent-image

FROM scratch as builder

ARG ARCH

COPY --from=task-agent-image /opt/circleci/linux/${ARCH}/circleci-agent /
COPY ./target/bin/${ARCH}/orchestrator /

FROM scratch

COPY --from=builder / /

ENTRYPOINT ["/orchestrator", "init"]
