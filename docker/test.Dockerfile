FROM scratch as builder

ARG TARGETPLATFORM

COPY ./target/bin/${TARGETPLATFORM}/orchestrator /
COPY ./target/bin/${TARGETPLATFORM}/fake-task-agent /circleci-agent

FROM scratch

COPY --from=builder / /

ENTRYPOINT ["/orchestrator", "init"]
