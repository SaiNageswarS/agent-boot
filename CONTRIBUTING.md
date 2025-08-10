# Contributing to Agent-Boot

Thank you for your interest in contributing to Agent-Boot! This document provides guidelines and information for contributors.

## 🚀 Getting Started

### Prerequisites

- Go 1.24+ installed
- Git
- Protocol Buffers compiler (for schema changes)

### Development Setup

1. **Fork and Clone**
   ```bash
   git clone https://github.com/your-username/agent-boot.git
   cd agent-boot
   ```

2. **Install Dependencies**
   ```bash
   go mod download
   ```

3. **Run Tests**
   ```bash
   go test ./...
   ```

4. **Generate Protocol Buffers** (if needed)
   ```bash
   cd proto
   ./build.sh
   ```

## 🛠️ Development Guidelines

### Code Style

- Follow standard Go conventions
- Use `gofmt` for formatting
- Add comments for exported functions and types
- Keep functions focused and small

### Testing

- Write tests for new functionality
- Maintain or improve test coverage
- Use table-driven tests where appropriate
- Include benchmarks for performance-critical code

Example test structure:
```go
func TestFeatureName(t *testing.T) {
    tests := []struct {
        name     string
        input    InputType
        expected OutputType
        wantErr  bool
    }{
        // test cases
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // test implementation
        })
    }
}
```

### Documentation

- Update README.md for user-facing changes
- Add package-level documentation
- Include examples in doc comments
- Update API documentation

## 📋 Types of Contributions

### 🐛 Bug Reports

When reporting bugs, please include:
- Go version and OS
- Minimal reproduction case
- Expected vs actual behavior
- Relevant logs or error messages

### ✨ Feature Requests

For feature requests:
- Describe the use case
- Explain why it would be valuable
- Consider backwards compatibility
- Provide implementation ideas if possible

### 🔧 Code Contributions

We welcome:
- Bug fixes
- New LLM provider support
- Tool implementations
- Performance improvements
- Documentation improvements

### Areas Looking for Contributions

- **LLM Providers**: OpenAI, Google Gemini, AWS Bedrock
- **Tools**: File operations, API calls, data processing
- **Examples**: Real-world use cases and tutorials
- **Performance**: Optimization and benchmarking
- **Documentation**: Tutorials, guides, API docs

## 🔄 Pull Request Process

1. **Create an Issue** (for large changes)
   - Discuss the change before implementing
   - Get feedback on approach

2. **Create a Branch**
   ```bash
   git checkout -b feature/descriptive-name
   ```

3. **Make Changes**
   - Follow coding guidelines
   - Add tests
   - Update documentation

4. **Test Your Changes**
   ```bash
   go test ./...
   go test ./... -race
   go test ./... -bench=.
   ```

5. **Commit**
   ```bash
   git commit -m "feat: add new LLM provider support"
   ```
   
   Use conventional commit format:
   - `feat:` new features
   - `fix:` bug fixes
   - `docs:` documentation changes
   - `refactor:` code refactoring
   - `test:` adding tests
   - `perf:` performance improvements

6. **Push and Create PR**
   ```bash
   git push origin feature/descriptive-name
   ```

7. **PR Review**
   - Address review comments
   - Update tests if needed
   - Ensure CI passes

## 🏗️ Architecture Guidelines

### Adding New LLM Providers

1. Implement the `LLMClient` interface:
   ```go
   type NewProviderClient struct {
       // provider-specific fields
   }
   
   func (c *NewProviderClient) GenerateInference(...) error {
       // implementation
   }
   
   func (c *NewProviderClient) GenerateInferenceWithTools(...) error {
       // implementation
   }
   
   func (c *NewProviderClient) Capabilities() llm.Capability {
       // return supported capabilities
   }
   
   func (c *NewProviderClient) GetModel() string {
       // return model name
   }
   ```

2. Add tests in `llm/new_provider_test.go`
3. Update documentation and examples

### Adding New Tools

1. Use the MCP tool builder:
   ```go
   tool := agent.NewMCPTool("tool-name", "description").
       StringParam("param", "description", required).
       WithHandler(handlerFunc).
       Build()
   ```

2. Implement handler with proper error handling
3. Add comprehensive tests
4. Document usage examples

### Schema Changes

1. Update `proto/agent.proto`
2. Run `./proto/build.sh` to regenerate
3. Update code using the schema
4. Test backwards compatibility

## 🧪 Testing Guidelines

### Test Structure

```
agent/
├── agent_test.go              # Unit tests for agent.go
├── agent_builder_test.go      # Unit tests for builder
├── execute_turn_based_test.go # Integration tests
└── integration_test.go        # End-to-end tests
```

### Test Categories

- **Unit Tests**: Test individual functions/methods
- **Integration Tests**: Test component interactions
- **Benchmark Tests**: Performance testing
- **Example Tests**: Verify documentation examples work

### Coverage Goals

- Maintain >70% overall coverage
- 100% coverage for critical paths
- Test error conditions
- Include edge cases

## 🔍 Code Review Guidelines

### For Contributors

- Keep PRs focused and small
- Provide clear descriptions
- Include tests and documentation
- Respond to feedback promptly

### For Reviewers

- Be constructive and helpful
- Focus on correctness and maintainability
- Consider performance implications
- Check test coverage

## 📦 Release Process

1. **Version Bump**: Update version in relevant files
2. **Changelog**: Update CHANGELOG.md with changes
3. **Tag**: Create a git tag following semantic versioning
4. **Release**: Create GitHub release with notes

## ❓ Getting Help

- **Questions**: Open a GitHub Discussion
- **Issues**: Create a GitHub Issue
- **Chat**: Join our community Discord (link TBD)

## 📜 Code of Conduct

We are committed to providing a welcoming and inclusive environment. Please:

- Be respectful and professional
- Welcome newcomers and help them learn
- Give constructive feedback
- Focus on what's best for the community

## 🙏 Recognition

Contributors will be:
- Listed in CONTRIBUTORS.md
- Mentioned in release notes
- Recognized in project documentation

Thank you for contributing to Agent-Boot! 🎉
