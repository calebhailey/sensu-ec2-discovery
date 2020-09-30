# Changelog

All notable changes to this project will be documented in this file. This
changelog format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## Unreleased

### Changed
- Updated to new Sensu Go SDK
- Made --sensu-insecure-tls-skip-verify actually work
- Added check for region since it is actually required
- Changed hard coded http status codes to text equivalents
- Switch to go modules
- Remove arch renaming from goreleaser

## 0.1.0 – December 22, 2019

### Added

- Added this CHANGELOG.md

- Added .goreleaser.yml for automated builds & GitHub releases

- Added support for use as a `sensuctl` command plugin (goreleaser creates an
  executable named `bin/entrypoint`)

- Refactored to use the sensu-plugins-go-library

- Improved logging of error conditions

### Changed

- Updated plugin to integrate with the Sensu Go Entities API (i.e. the plugin
  now creates Sensu Go "entities" instead of Sensu Classic "clients")
