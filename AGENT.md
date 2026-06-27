# Workspace Agent Rules

- In this workspace, run all Git operations with elevated permissions from the start. Non-elevated Git frequently leaves or fails on `.git/index.lock` here.
- In the `forcepser` fork, keep the upstream version string unchanged unless the user explicitly asks to change it.
- For fork-only `forcepser` releases, bump the `_rebN` tag suffix instead of changing the base version number.