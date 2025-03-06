# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.5.3] - 2024-03-28

### Added
- Added Attack Surface Management module (currently disabled for beta)
- Added improved cost tracking for cloud resources
- Added real-time cost calculation for running scans
- Added support for multiple cloud providers in scan infrastructure
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
- Added automatic migration for existing findings to support new hash-based deduplication:
  - Generates hashes for all existing findings
  - Creates history entries for proper tracking
  - Preserves all existing data and relationships
  - Handles duplicate findings by updating history entries
  - Migration runs automatically on application startup

### Changed
- Optimized scan cost calculations to reduce unnecessary API calls
- Improved precision in cost display and calculations
- Enhanced UI responsiveness for scan management
- Updated scan status display with more detailed information
- Improved findings page performance with optimized database queries
- Enhanced findings filtering and sorting capabilities
- Updated authentication middleware to support both user sessions and API keys
- Improved error handling in scan result processing
- Enhanced scan status tracking and updates
- Improved file upload handling with progress tracking
- Changed findings storage to use hash-based deduplication for better tracking and history

### Fixed
- Fixed cost calculation precision issues for small amounts
- Fixed duplicate UI elements in scan management
- Fixed issues with scan status updates
- Fixed template directory handling in Nuclei scans
- Fixed findings bulk update endpoint path
- Fixed permission checks for admin users
- Fixed scan scheduling validation
- Fixed notification delivery reliability
- Fixed user invitation token handling
- Fixed notification system to handle cases where no notification rules are configured
  - Added graceful handling when no rules exist for scan events
  - Improved logging to show when notifications are skipped due to missing rules
  - Fixed error handling in finding rollups for scans without findings
  - Added default values for finding summaries when no findings exist

### Security
- Implemented secure API key generation and validation
- Enhanced permission checks for findings access
- Added validation for file uploads
- Improved authentication token handling
- Improved notification manager to use configured rules instead of hardcoded channels
  - Updated `NotifyScanFinished` to respect notification rules
  - Added logging for notification rule matching and channel selection
  - Removed hardcoded channel lists from notification methods

### Migration Notes
- When upgrading to 0.5.3, the application will automatically:
  1. Add hashes to all existing findings in nuclei_results
  2. Create corresponding entries in nuclei_findings_history
  3. Update scan rollups as needed
  4. No manual intervention required
  5. No data loss will occur during migration

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

### Changed
- Improved notification manager to use configured rules instead of hardcoded channels
  - Updated `NotifyScanFinished` to respect notification rules
  - Added logging for notification rule matching and channel selection
  - Removed hardcoded channel lists from notification methods

### Added
- Added better logging for notification events
  - Log when no enabled rules are found for an event
  - Log the number of rules found and matching channels
  - Log when notifications are skipped due to missing rules

 