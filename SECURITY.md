# Security Policy

## Supported Versions

Runme is distributed as a rolling release. Security fixes are provided in the
latest stable release of the Runme CLI and the Runme VS Code extension.
Please upgrade to the latest release before reporting an issue, and include the
exact version you tested.

| Version            | Supported          |
| ------------------ | ------------------ |
| Latest stable CLI  | :white_check_mark: |
| Latest VS Code extension | :white_check_mark: |
| Older releases     | :x:                |

## Reporting a Vulnerability

Please **do not open a public GitHub issue** for security vulnerabilities.

Use one of the private channels below:

1. If GitHub private vulnerability reporting is enabled for this repository,
   open the **Security** tab and choose **Report a vulnerability**. This keeps
   the report private and lets maintainers coordinate a fix and advisory.
2. Otherwise, contact the maintainers listed in
   [MAINTAINERS.md](MAINTAINERS.md) directly.

Please include:

- affected component (CLI, VS Code extension, server) and exact version;
- operating system and architecture;
- a description of the impact and the attack scenario;
- minimal reproduction steps or a proof of concept;
- any relevant logs, configuration, or source references.

Do not include secrets, credentials, or personal data in the report.

## What to Expect

- We aim to acknowledge new reports within 7 days.
- We aim to provide an initial assessment within 14 days.
- We will work with you on a fix and a coordinated disclosure timeline.
  Our default target is 90 days, but we can adjust it for complex issues.
- Reporters are credited in the advisory and CVE record unless they prefer to
  remain anonymous.

## Scope

This policy covers the Runme CLI, the Runme VS Code extension, and the Runme
server and local protocol, including their release artifacts.

## Disclosure

Please allow us to fix and release an update before publishing details. We will
coordinate the publication of a security advisory and, where appropriate, a CVE
record with the reporter.

Thank you for helping keep Runme and its users safe.
