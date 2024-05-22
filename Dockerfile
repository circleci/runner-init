FROM busybox:stable-musl as build

FROM scratch as temp

ARG ARCH=amd64

COPY ./circleci-agent-${ARCH} /opt/circleci/circleci-agent
COPY --from=build /bin/cp /bin/cp

FROM scratch

COPY --from=temp / /

CMD ["/bin/cp"]