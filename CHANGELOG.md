# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Added support for bulk updates of findings status (acknowledged, false positive, remediated)
- Added API key authentication for scan results submission
- Added chunked file upload support for large scan results
- Added retry mechanism for failed chunk uploads
- Added scan scheduling system with support for daily, weekly, and monthly schedules
- Added support for custom cron expressions in scan scheduling
- Added notification service integration for scan events
- Added Nuclei Findings Save as Default Filter
- Added support for all official Nuclei profiles including:
  - Cloud configurations (AWS, Azure, Alibaba)
  - Compliance checks
  - CVE scanning
  - Default login detection
  - Kubernetes cluster security
  - Known Exploited Vulnerabilities (KEV)
  - Misconfigurations
  - OSINT gathering
  - Penetration testing
  - Privilege escalation
  - Subdomain takeovers
  - Windows security auditing
  - WordPress security

### Changed
- Improved findings page performance with optimized database queries
- Enhanced findings filtering and sorting capabilities
- Updated authentication middleware to support both user sessions and API keys
- Improved error handling in scan result processing
- Enhanced scan status tracking and updates
- Improved file upload handling with progress tracking

### Fixed
- Fixed findings bulk update endpoint path
- Fixed permission checks for admin users
- Fixed scan scheduling validation
- Fixed notification delivery reliability
- Fixed user invitation token handling

### Security
- Implemented secure API key generation and validation
- Enhanced permission checks for findings access
- Added validation for file uploads
- Improved authentication token handling

## [0.5.2] - 2024-03-27

### Added
- Added severity override functionality in findings
- Added status badges with tooltips for acknowledged, false positive, and remediated states
- Added bulk update functionality for findings status
- Added findings table improvements with better filtering and sorting
- Added severity color coding in findings table and modals
- Added findings grouping by template ID with collapsible sections
- Added real-time updates for findings status changes
- Added display of extracted results in findings details
- Added improved visualization of matched content in findings

### Changed
- Improved findings modal UI with better organization of information
- Enhanced severity display to show both original and override severities
- Updated status indicators to use consistent icons across the application
- Improved findings table performance with optimized queries
- Enhanced findings filtering with multiple status filters
- Enhanced display of matched content with better formatting and context
- Updated terminal font in findings modal to use Consolas, Monaco, and Courier New for better readability
- Optimized terminal display height in findings modal for better space utilization

### Fixed
- Fixed issue where severity updates weren't immediately reflected in the UI
- Fixed findings table pagination issues
- Fixed status badge styling consistency
- Fixed real-time updates for findings modifications
- Fixed type errors in findings components
- Fixed display of extracted results formatting

### Security
- Updated findings collection rules to properly handle admin permissions
- Enhanced access control for findings management

## [0.5.1] - 2024-03-XX

### Fixed
- Fixed version display in UI to correctly show the GitHub release tag version
- Fixed version handling in development mode to show "development" instead of checking for updates
- Fixed version injection in build process to properly set version from GitHub tags
- User invitation system now properly sets passwords during account and user creation is complete.

## [0.5.0] - 2024-02-04
First release of the Orbit Scanner.

### Added
- Basic scan functionality
- Web interface
- Backend API
- Database integration 
