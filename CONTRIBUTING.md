# Contributing to Quantix

Thank you for your interest in contributing to Quantix! This document provides guidelines for contributing to this project.

## Code of Conduct

This project and everyone participating in it is governed by our Code of Conduct. By participating, you are expected to uphold this code.

## How Can I Contribute?

### Reporting Bugs

- Use the GitHub issue tracker
- Use the bug report template
- Include detailed steps to reproduce
- Include your environment details

### Suggesting Enhancements

- Use the feature request template
- Describe the problem and proposed solution
- Consider the impact on existing functionality

### Pull Requests

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## Development Setup

1. Clone the repository
```bash
git clone https://github.com/sy-vendor/Quantix.git
cd Quantix
```

2. Install dependencies
```bash
go mod download
```

3. Run tests
```bash
go test ./...
```

4. Build the project
```bash
go build
```

## Code Style

- Follow Go conventions and best practices
- Use meaningful variable and function names
- Add comments for complex logic
- Keep functions small and focused
- Write tests for new functionality

## Testing

- Write unit tests for new features
- Ensure all tests pass before submitting PR
- Add integration tests for complex features

## Documentation

- Update README.md if adding new features
- Add inline comments for complex code
- Update API documentation if needed

## Commit Messages

Use conventional commit format:
- `feat:` for new features
- `fix:` for bug fixes
- `docs:` for documentation changes
- `style:` for formatting changes
- `refactor:` for code refactoring
- `test:` for adding tests
- `chore:` for maintenance tasks

## Questions?

If you have questions about contributing, please open an issue or contact the maintainers.

Thank you for contributing to Quantix! 🚀 