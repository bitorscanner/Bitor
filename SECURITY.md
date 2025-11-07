# Security Policy

## Reporting a Vulnerability

We take the security of Bitor seriously. If you discover a security vulnerability, please follow these steps:

### 🔒 Private Disclosure (Preferred)

**Use GitHub's Private Security Reporting:**

1. Go to the [Security tab](../../security) of this repository
2. Click "Report a vulnerability"
3. Fill out the security advisory form with:
   - Description of the vulnerability
   - Steps to reproduce
   - Potential impact
   - Suggested fix (if available)

**Alternative Method:**

If you cannot use GitHub's private reporting, you may open a security advisory directly or contact the maintainers through GitHub discussions (mark as security-related).

⚠️ **DO NOT** open a public GitHub issue for security vulnerabilities.

### Response Timeline

- **Initial Response**: Within 48 hours
- **Status Update**: Within 7 days
- **Fix Timeline**: Depends on severity (Critical: 7-14 days, High: 30 days, Medium: 60 days)

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| Latest  | :white_check_mark: |
| < Latest| :x:                |

We recommend always using the latest version of Bitor.

## Security Updates

Security advisories and updates are published in:
- [`docs/security/`](./docs/security/) - Detailed security advisories
- [`CHANGELOG.md`](./CHANGELOG.md) - Release notes with security fixes
- GitHub Security Advisories (for critical issues)

## Known Security Issues

### Fixed Vulnerabilities

- **[CVE-2025-PATH-TRAVERSAL](./docs/security/CVE-2025-PATH-TRAVERSAL.md)** (2025-11-03)
  - **Severity**: Critical (CVSS 9.1)
  - **Component**: Template Management System
  - **Status**: Fixed in latest version
  - **Impact**: Path traversal allowing arbitrary file read/write/delete

## Security Best Practices

When deploying Bitor:

1. **Authentication**: Always use strong passwords for admin accounts
2. **Network Security**: Deploy behind a firewall or VPN when possible
3. **Updates**: Keep Bitor updated to the latest version
4. **Permissions**: Use the principle of least privilege for user groups
5. **Monitoring**: Monitor logs for suspicious activity
6. **Backups**: Regularly backup your database and configuration

## Security Features

Bitor includes several security features:

- ✅ JWT-based authentication
- ✅ Role-based access control (RBAC)
- ✅ API key authentication for scans
- ✅ Path traversal protection
- ✅ Input validation and sanitization
- ✅ Activity logging
- ✅ Public template protection

## Acknowledgments

We appreciate security researchers who responsibly disclose vulnerabilities:

- **chimmeee** - Path traversal vulnerability in template handlers (2025-11-03)

## Contact

- **Security Issues**: Use [GitHub Private Security Reporting](../../security/advisories/new)
- **General Support**: Open a [GitHub Issue](https://github.com/yourusername/bitor/issues)
- **Discussions**: Join [GitHub Discussions](https://github.com/yourusername/bitor/discussions)

