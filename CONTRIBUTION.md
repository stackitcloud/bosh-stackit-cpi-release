# Contributing to the STACKIT BOSH CPI

Thank you for your interest in contributing. Contributions to the code, tests,
documentation, examples, and issue tracker are welcome.

## Before you start

Please search the [existing issues](https://github.com/stackitcloud/bosh-stackit-cpi-release/issues)
before opening a new one. For a substantial change, open an issue first so the
approach can be discussed and duplicate work can be avoided.

Please do not use issues to report security vulnerabilities. Follow the process
in [SECURITY.md](SECURITY.md) instead.

## Development setup

The CPI source is in `src/stackit-cpi`. You need:

- Go 1.24 or newer
- BOSH CLI, if you are building a BOSH release
- STACKIT credentials only for integration tests

Clone the repository and switch to the `develop` branch, which is the
development trunk:

```bash
git clone https://github.com/stackitcloud/bosh-stackit-cpi-release.git
cd bosh-stackit-cpi-release
git switch develop
```

Build the CPI with:

```bash
cd src/stackit-cpi
GOFLAGS=-mod=mod go build -o ../../bin/stackit-cpi .
```

See the [development documentation](docs/README.md) and the main
[README](README.md) for configuration and deployment details.

## Tests and checks

Run the unit tests from `src/stackit-cpi`:

```bash
GOFLAGS=-mod=mod go test -p 12 -v ./cpi ./cpi/lib
```

Format Go changes before submitting them:

```bash
gofmt -w path/to/changed.go
```

Integration tests require STACKIT credentials and access to a test project.
Run them from `src/stackit-cpi/cpi/integration`:

```bash
GOFLAGS=-mod=mod go run github.com/onsi/ginkgo/v2/ginkgo -p -v \
  --flake-attempts 3 --randomize-all --fail-fast
```

Do not commit credentials, private keys, generated release artifacts, or
confidential customer data. Redact sensitive values from logs and issue
reports.

## Branches and pull requests

Create a focused branch from `develop`. Keep each pull request limited to one
logical change and update documentation or examples when behavior changes.

Use a clear commit and pull request title. Conventional Commit prefixes such
as `feat:`, `fix:`, `docs:`, `refactor:`, and `chore:` are recommended.

A pull request should explain:

- what changed and why;
- how the change was tested;
- any limitations, compatibility concerns, or operational impact; and
- the related issue, when one exists.

Please respond to review feedback and keep follow-up changes in the same pull
request when they belong to its original scope.

The `main` branch is used for release engineering. Pull requests for normal
development should target `develop` unless a maintainer asks otherwise.

## AI-assisted contributions

AI-assisted contributions are welcome and follow the same standards as all
other contributions. The contributor remains responsible for the submitted
code, documentation, tests, and licensing.

Before opening a pull request:

- review every generated or AI-suggested change;
- verify that the implementation is correct and does not add unsuitable
  dependencies or copied material with incompatible licensing;
- run the relevant tests and formatting checks; and
- disclose meaningful AI assistance in the pull request description when it
  materially contributed to the change.

Never provide credentials, private keys, customer information, or other
confidential material to an AI tool.

## Reporting bugs and requesting features

Open an issue with a concise description and enough context to reproduce or
evaluate the problem. For bugs, include:

- the CPI and BOSH versions;
- STACKIT region and relevant configuration, with secrets removed;
- steps to reproduce;
- expected and actual behavior; and
- relevant logs or error messages, redacted as needed.

For feature requests, describe the use case, expected behavior, and any
compatibility or operational requirements.

## License

By contributing, you agree that your contributions are provided under the
project's [MIT License](LICENSE).
