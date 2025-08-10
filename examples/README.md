# Agent-Boot Examples

This directory contains example applications demonstrating various features of the Agent-Boot framework.

## 📚 Available Examples

### 🧮 [Simple Calculator](./simple-calculator/)
A basic agent with a calculator tool that demonstrates:
- Tool creation and parameter handling
- Real-time progress reporting
- Basic agent-tool interaction
- Console output formatting

**Features**: Basic math operations, step-by-step calculation, console progress updates

### 🔍 RAG Agent *(Coming Soon)*
An advanced Retrieval-Augmented Generation agent that demonstrates:
- Document indexing and search
- Context summarization
- Multiple tool orchestration
- Vector database integration

### 🤖 Multi-Tool Research Assistant *(Coming Soon)*
A comprehensive research agent featuring:
- Web search capabilities
- Database queries
- File operations
- Report generation

### 🌐 gRPC Streaming Service *(Coming Soon)*
A network-enabled agent service that shows:
- gRPC server implementation
- Client-server communication
- Streaming responses over network
- Service deployment patterns

### 🔌 Custom LLM Provider *(Coming Soon)*
Implementation guide for adding new LLM providers:
- Provider interface implementation
- Authentication handling
- Error management
- Testing strategies

## 🚀 Quick Start

1. **Prerequisites**
   ```bash
   # Install Go 1.24+
   go version
   
   # Install Ollama (for local examples)
   curl -fsSL https://ollama.ai/install.sh | sh
   ollama pull llama3.2:latest
   
   # Set environment variables
   export OLLAMA_HOST="http://localhost:11434"
   ```

2. **Clone and Setup**
   ```bash
   git clone https://github.com/SaiNageswarS/agent-boot.git
   cd agent-boot
   go mod download
   ```

3. **Run an Example**
   ```bash
   # Run the calculator example
   go run examples/simple-calculator/main.go
   ```

## 🛠️ Development

### Creating New Examples

1. Create a new directory:
   ```bash
   mkdir examples/my-example
   ```

2. Add a `main.go` file with your implementation
3. Include a `README.md` with setup instructions
4. Add to this examples index

### Example Structure

```
examples/
├── README.md                 # This file
├── simple-calculator/        # Basic calculator agent
│   ├── main.go              # Implementation
│   └── README.md            # Setup guide
├── rag-agent/               # RAG example (coming soon)
├── multi-tool-assistant/    # Multi-tool example (coming soon)
├── grpc-service/           # Network service example (coming soon)
└── custom-llm-provider/    # Provider implementation guide (coming soon)
```

### Best Practices

- **Self-contained**: Each example should be runnable independently
- **Well-documented**: Include setup instructions and explanations
- **Error handling**: Show proper error handling patterns
- **Testing**: Include example tests where applicable
- **Comments**: Add helpful comments explaining key concepts

## 📋 Example Categories

### 🎯 **Beginner Examples**
- Simple Calculator: Basic tool usage
- Echo Agent: Minimal agent setup
- Static Responses: No-tool agent

### 🚀 **Intermediate Examples**
- RAG Agent: Document search and summarization
- Multi-Tool Agent: Complex tool orchestration
- Custom Tools: Building specialized tools

### 🏗️ **Advanced Examples**
- gRPC Service: Network deployment
- Custom Providers: LLM integration
- Performance Optimization: High-throughput scenarios

## 🧪 Testing Examples

Run tests for all examples:
```bash
go test ./examples/...
```

Run a specific example test:
```bash
go test ./examples/simple-calculator/...
```

## 🤝 Contributing Examples

We welcome example contributions! Please:

1. Follow the existing structure
2. Include comprehensive documentation
3. Add error handling
4. Test your example thoroughly
5. Update this README

See our [Contributing Guide](../CONTRIBUTING.md) for more details.

## 📞 Support

- 📖 Documentation: Check individual example READMEs
- 🐛 Issues: Report problems via GitHub Issues
- 💬 Questions: Use GitHub Discussions
- 📧 Contact: [sai@example.com](mailto:sai@example.com)

---

Happy coding! 🎉
