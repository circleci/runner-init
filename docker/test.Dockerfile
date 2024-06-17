FROM scratch as builder

ARG ARCH

COPY ./target/bin/${ARCH}/orchestrator /
COPY ./target/bin/${ARCH}/fake-task-agent /

FROM scratch

COPY --from=builder / /

ENTRYPOINT ["/orchestrator", "init"]
