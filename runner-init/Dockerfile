FROM busybox:stable-musl as build

ARG ARCH=amd64

COPY ./bin/circleci-agent-${ARCH} /opt/circleci/circleci-agent
RUN ln -s /opt/circleci/circleci-agent /opt/circleci/circleci

FROM scratch as temp

COPY --from=build /opt/circleci/circleci-agent /opt/circleci/circleci-agent
COPY  --from=build /opt/circleci/circleci /opt/circleci/circleci
COPY --from=build /bin/cp /bin/cp

FROM scratch

COPY --from=temp / /

ENTRYPOINT ["/bin/cp"]