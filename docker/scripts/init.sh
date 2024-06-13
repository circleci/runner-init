#!/usr/bin/env sh

dest=${1:?'task agent dest must be specified'}

cp /opt/circleci/bin/circleci-agent "$dest"/circleci-agent
ln -s "$dest"/circleci-agent "$dest"/circleci
