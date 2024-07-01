# CircleCI Runner Init Changelog

This document serves to keep track of all changes made to the Runner init images and agents. Please follow the guidelines below while adding an entry:

- Each code change that is not related to tests or CI/CD should have its own entry under the `Edge` section.
- Every entry should include the pull request number associated with the change.
- Internal changes with no visible or functional impact on users should be marked as 'Internal'.

By following these guidelines, we can easily determine which changes should be included in the public changelog located at [https://circleci.com/changelog/runner/](https://circleci.com/changelog/runner/). Thank you for your contribution!

## Edge

- [#26](https://github.com/circleci/runner-init/pull/26) [INTERNAL] Handle shutdown of a task with a termination grace period.
- [#25](https://github.com/circleci/runner-init/pull/25) [INTERNAL] Implement the ability to execute a task and a custom entrypoint.
- [#24](https://github.com/circleci/runner-init/pull/24) [INTERNAL] Add checksum validation of the task token to help detect transmission errors.
- [#18](https://github.com/circleci/runner-init/pull/18) [INTERNAL] Set up the initial base for orchestrator's `run-task` command. This includes adding a health check server and the configuration required to execute task agent.
- [#17](https://github.com/circleci/runner-init/pull/17) [INTERNAL] Build server images.
- [#16](https://github.com/circleci/runner-init/pull/16) [INTERNAL] Add an acceptance test framework and cases for the init command. In addition, some changes were made to the CLI configuration to account for limitations on positional arguments of the ex test runner.
- [#14](https://github.com/circleci/runner-init/pull/14) [INTERNAL] Implement the init script as a mode of the orchestrator. This allows for the use of scratch for a minimal base image.
- [#13](https://github.com/circleci/runner-init/pull/13) [INTERNAL] Use [GoReleaser](https://goreleaser.com/) for building and pushing the images and manifests.
- [#12](https://github.com/circleci/runner-init/pull/12) [INTERNAL] Use [GoReleaser](https://goreleaser.com/) for building the binaries.
- [#11](https://github.com/circleci/runner-init/pull/11) [INTERNAL] Download task agent binaries directly via the Dockerfile.
- [#10](https://github.com/circleci/runner-init/pull/10) [INTERNAL] Set up linting tools, initiated a changelog, and performed other configurations in preparation for the orchestration agent.
