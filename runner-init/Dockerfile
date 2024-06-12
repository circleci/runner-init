FROM circleci/picard:agent as task-agent-image

FROM busybox:stable-musl

ARG ARCH=amd64

COPY --from=task-agent-image /opt/circleci/linux/${ARCH}/circleci-agent /opt/circleci/bin/circleci-agent
COPY ./runner-init/init.sh /init.sh

ENTRYPOINT ["/init.sh"]
