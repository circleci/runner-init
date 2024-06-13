# CircleCI Runner Init Changelog

This document serves to keep track of all changes made to the Runner init images and agents. Please follow the guidelines below while adding an entry:

- Each code change that is not related to tests or CI/CD should have its own entry under the `Edge` section.
- Every entry should include the pull request number associated with the change.
- Internal changes with no visible or functional impact on users should be marked as 'Internal'.

By following these guidelines, we can easily determine which changes should be included in the public changelog located at [https://circleci.com/changelog/runner/](https://circleci.com/changelog/runner/). Thank you for your contribution!

## Edge

- [#13](https://github.com/circleci/runner-init/pull/13) [INTERNAL] Use [GoReleaser](https://goreleaser.com/) for building and pushing the images and manifests.
- [#12](https://github.com/circleci/runner-init/pull/12) [INTERNAL] Use [GoReleaser](https://goreleaser.com/) for building the binaries.
- [#11](https://github.com/circleci/runner-init/pull/11) [INTERNAL] Download task agent binaries directly via the Dockerfile.
- [#10](https://github.com/circleci/runner-init/pull/10) [INTERNAL] Set up linting tools, initiated a changelog, and performed other configurations in preparation for the orchestration agent.
