# Changelog

All notable changes to this project will be documented in this file.

## [Unreleased]

### Added
- **Professional CLI Framework**: Migrated from manual flag parsing to [Cobra](https://github.com/spf13/cobra), following best practices from [longbridge-terminal](https://github.com/longbridge/longbridge-terminal)
- **Improved Help System**: Rich help text with examples for all commands
- **Verbose Mode**: Global `--verbose` flag for detailed logging during operations
- **Shell Completion**: Auto-generated completion scripts for bash, zsh, fish, and PowerShell
- **Better Error Messages**: User-friendly error messages with emoji indicators (✓, ⚠, ❌, 🚀, 🛑, 🔧)
- **Enhanced Makefile**:
  - Default `make` now shows help (instead of building)
  - New `dev-verbose` target for mock mode with verbose output
  - Improved help with examples and version information

### Changed
- **Command Structure**:
  - `longbridge-fs --version` instead of `longbridge-fs version`
  - Consistent flag naming across all commands
  - Better organized command hierarchy
- **Controller Logging**: Enhanced with emoji indicators for better readability
- **Init Command**: Now provides verbose output showing each created file/directory

### Improved
- CLI follows modern command-line interface conventions
- Better integration with AI agents and scripting tools
- More informative progress indicators during operations
- Professional output formatting matching industry standards

## Inspiration

These improvements were inspired by the official [Longbridge Terminal CLI](https://github.com/longbridge/longbridge-terminal), which demonstrates best practices for building AI-native command-line tools for financial trading platforms.
