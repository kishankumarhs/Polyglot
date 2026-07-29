# Contributing to Polyglot

Thank you for your interest in contributing to Polyglot! We welcome contributions from the community and appreciate your help in making this project better.

## Code of Conduct

Please review our [Code of Conduct](CODE_OF_CONDUCT.md) before participating in this project. All contributors are expected to adhere to it.

## Getting Started

1. **Fork the repository** on GitHub
2. **Clone your fork** locally:
   ```bash
   git clone https://github.com/YOUR-USERNAME/Polyglot.git
   cd Polyglot
   ```
3. **Add upstream remote**:
   ```bash
   git remote add upstream https://github.com/kishankumarhs/Polyglot.git
   ```
4. **Create a feature branch**:
   ```bash
   git checkout -b feature/your-feature-name
   ```

## Development Setup

### Prerequisites
- Go 1.21+
- Python 3.8+
- Node.js 18+
- .NET SDK 6.0+
- Make
- C compiler (gcc/clang or MinGW on Windows)

### Building

```bash
# Build the native library
make build-native

# Run tests
make test

# Build language bindings
make build-python
make build-node
make build-dotnet
```

Refer to [Build Guide](docs/build.md) for detailed instructions.

## Workflow & Submodules

This project uses Git submodules for language bindings. Please read [Submodule Workflow](docs/SUBMODULE-WORKFLOW.md) and [Quick Reference](SUBMODULE-QUICK-REFERENCE.md) before making changes to bindings.

## Making Changes

### Commit Guidelines

- Use clear, descriptive commit messages
- Reference related issues when applicable (e.g., "Fixes #123")
- Keep commits focused on a single logical change
- Write commit messages in imperative mood ("Add feature" not "Added feature")

Example:
```
Add HTTP sink error retry logic

- Implement exponential backoff for failed requests
- Add configurable max retries parameter
- Log retry attempts at debug level

Fixes #456
```

### Code Style

- **Go**: Follow [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- **Python**: Follow [PEP 8](https://www.python.org/dev/peps/pep-0008/)
- **JavaScript/TypeScript**: Follow [Google JavaScript Style Guide](https://google.github.io/styleguide/tsguide.html)
- **.NET**: Follow [C# Coding Conventions](https://learn.microsoft.com/en-us/dotnet/csharp/fundamentals/coding-style/coding-conventions)

Run linters before committing:
```bash
make lint
```

### Testing

- Write tests for new functionality
- Ensure all tests pass locally before pushing:
  ```bash
  make test
  ```
- Add integration tests in the appropriate `tests/` directory
- Maintain or improve code coverage

## Areas for Contribution

### Core Logger (Go)
- Performance optimizations
- Bug fixes in async queue or file rotation
- New sink implementations
- Configuration enhancements

### Language Bindings
- Bug fixes or API improvements
- Better error handling and validation
- Performance tuning
- Test coverage improvements

### Documentation
- Clarifying existing guides
- Adding tutorials or examples
- Improving API documentation
- Fixing typos or formatting

### CI/CD & Build
- Improving build scripts
- GitHub Actions workflow optimizations
- Cross-platform testing

## Pull Request Process

1. **Create a descriptive PR title** that clearly explains the change
2. **Fill out the PR template** with:
   - Summary of changes
   - Motivation and context
   - How to test
   - Screenshots (if applicable)
   - Checklist of requirements

3. **Ensure CI passes**:
   - All GitHub Actions workflows pass
   - No breaking changes to the C ABI

4. **Address review feedback** promptly
5. **Keep PR focused** - avoid mixing unrelated changes

### PR Checklist

- [ ] Code follows style guidelines
- [ ] Tests added/updated for changes
- [ ] Documentation updated if needed
- [ ] No new warnings or errors
- [ ] Commit history is clean
- [ ] No breaking changes to public APIs or ABI

## Language Binding Contributions

When contributing to a specific language binding:

1. Check if it's a git submodule (likely in `bindings/`)
2. **Update the submodule** in its own repository first
3. **Update** the main Polyglot repo's submodule reference
4. Ensure the ABI (`api/abi.json`) hasn't changed
5. Test against the latest core library

See [Repositories](docs/REPOSITORIES.md) for the independent package locations:
- 📦 npm: [@polyglot/logger](https://github.com/kishankumarhs/polyglot-node)
- 📦 PyPI: [polyglot-logger](https://github.com/kishankumarhs/polyglot-py)
- 📦 NuGet: [Polyglot.Logger](https://github.com/kishankumarhs/polyglot-csharp)

## Reporting Issues

When reporting bugs or suggesting features:

1. **Check existing issues** to avoid duplicates
2. **Provide a clear title and description**
3. **Include steps to reproduce** (for bugs)
4. **Share your environment** (OS, language version, etc.)
5. **Attach logs or screenshots** if helpful

## Questions?

- 📖 Check [docs/](docs/) for documentation
- 💬 Open a discussion in GitHub Discussions
- 🐛 Open an issue for bug reports or feature requests

## Recognition

Contributors are recognized in our community. Significant contributions may be highlighted in release notes and project documentation.

---

Thank you for contributing to Polyglot! 🎉
