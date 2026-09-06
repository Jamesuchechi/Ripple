# Security Policy & Vulnerability Disclosure

## Supported Versions

Only the latest major release (`v1.x`) of Ripple receives security updates and security patches.

| Version | Supported          |
| ------- | ------------------ |
| 1.x     | :white_check_mark: |
| < 1.0   | :x:                |

## Reporting a Vulnerability

The Ripple team takes security seriously. If you discover a security vulnerability in Ripple (such as multi-tenant data leakage, API key verification bypass, HMAC webhook tampering, or unauthorized queue access), please do NOT create a public GitHub issue.

Instead, please report security vulnerabilities via email to:
**security@ripple.dev**

Include the following details in your report:
- Type of vulnerability (e.g. multi-tenant isolation breach, authentication bypass, denial of service).
- Step-by-step instructions or proof-of-concept payload to reproduce the flaw.
- Affected component or package (`auth`, `dispatcher`, `store`, `queue`, `notifier`).

### Disclosure Timeline
- **Acknowledgement**: Within 24 hours of receiving your report.
- **Triage & Patch**: Security patch published within 7 business days.
- **Public Advisory**: Published via GitHub Security Advisories following patch deployment.

## Security Architecture Summary

Ripple enforces multi-layer security controls:
1. **SHA-256 API Key Hashing**: Plaintext API keys are never stored in the database.
2. **Strict Multi-Tenant Key Namespacing**: All Redis ZSETs, atomic counters, locks, and NATS JetStream channels are strictly scoped by `project_id`.
3. **Signed Webhooks**: All outbound HTTP webhook payloads are signed with an `X-Ripple-Signature` HMAC-SHA256 digest to prevent payload tampering and forgery.
