FROM scratch as builder

ARG ARCH

COPY ./target/bin/${ARCH}/orchestrator /
COPY ./target/bin/${ARCH}/fake-task-agent /circleci-agent

FROM scratch

COPY --from=builder / /

ENTRYPOINT ["/orchestrator", "init"]
