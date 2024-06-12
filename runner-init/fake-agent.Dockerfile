FROM busybox:stable-musl as build

ARG ARCH=amd64

COPY ./target/bin/${ARCH}/fake-task-agent /opt/circleci/bin/circleci-agent
COPY ./runner-init/init.sh /init.sh

ENTRYPOINT ["/init.sh"]
