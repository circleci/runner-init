# CircleCI Runner Init Changelog

This document serves to keep track of all changes made to the Runner init images and agents. Please follow the guidelines below while adding an entry:

- Each code change that is not related to tests or CI/CD should have its own entry under the `Edge` section.
- Every entry should include the pull request number associated with the change.
- Internal changes with no visible or functional impact on users should be marked as 'Internal'.

By following these guidelines, we can easily determine which changes should be included in the public changelog located at [https://circleci.com/changelog/runner/](https://circleci.com/changelog/runner/). Thank you for your contribution!

## Edge

- [#39](https://github.com/circleci/runner-init/pull/39) [INTERNAL] Fix `init` command matching on startup 
- [#37](https://github.com/circleci/runner-init/pull/37) [INTERNAL] Refactored command execution structure within the orchestrator and ensured cleanup of the child process group spawned by the task agent.
- [#36](https://github.com/circleci/runner-init/pull/36) [INTERNAL] Integrate the runner API client in the task orchestrator. This enables retrying or reporting infrastructure failures as appropriate.
- [#33](https://github.com/circleci/runner-init/pull/33) [INTERNAL] Start signing the init images and manifests using [Cosign](https://docs.sigstore.dev/about/overview/).
- [#32](https://github.com/circleci/runner-init/pull/32) [INTERNAL] Implement an HTTP client for the runner API to allow the orchestrator to retry or fail tasks directly in the future.
- [#31](https://github.com/circleci/runner-init/pull/31) [INTERNAL] Wait on the task context finishing to stop the process reaper since the parent context can get cancelled first.
- [#30](https://github.com/circleci/runner-init/pull/30) [INTERNAL] Add init-like behaviour (reaping child subprocesses to prevent them from becoming zombie processes) to the task orchestrator.
- [#28](https://github.com/circleci/runner-init/pull/28) [INTERNAL] Pass orchestrator configuration through an environment variable from a Kubernetes Secret. This variable is stripped from the environment that is passed to the task agent command to prevent token leakage in the task environment.
- [#27](https://github.com/circleci/runner-init/pull/27) [INTERNAL] Handle shutdown of a task with a termination grace period.
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
