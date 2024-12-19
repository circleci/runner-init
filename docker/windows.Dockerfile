ARG PICARD_VERSION=agent

FROM --platform=${BUILDPLATFORM} circleci/picard:${PICARD_VERSION} AS task-agent-image

FROM --platform=${BUILDPLATFORM} scratch AS builder

ARG TARGETPLATFORM

COPY --from=task-agent-image /opt/circleci/${TARGETPLATFORM}/circleci-agent /circleci-agent.exe
COPY ./target/bin/${TARGETPLATFORM}/orchestrator.exe /

FROM mcr.microsoft.com/windows/nanoserver:ltsc2025

COPY --from=builder / /

ENTRYPOINT ["c:\\orchestrator.exe", "init"]
