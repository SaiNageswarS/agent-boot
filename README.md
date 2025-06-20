# 🚀 Agent Boot

**A high-performance, multi-language RAG system that supercharges Claude with domain-specific knowledge through intelligent document processing and semantic search.**

Agent Boot is a production-ready platform that combines the best of Go's performance with Python's ML ecosystem, delivering a seamless AI-powered search experience through Claude's MCP (Model Context Protocol).

## ✨ Features

- **🔥 Blazing Fast**: Go-powered backend with gRPC services for maximum performance
- **🧠 Smart Processing**: Python-based ML pipeline for document understanding and entity extraction
- **🔍 Hybrid Search**: Vector + text search with medical entity enhancement
- **⚡ Real-time**: Temporal workflows for scalable document processing
- **🌐 Multi-tenant**: Secure, isolated environments per tenant
- **🤖 Claude Integration**: Native MCP agent for seamless AI interactions
- **☁️ Cloud Native**: Azure Blob Storage + MongoDB with auto-scaling

## 🏗️ Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Claude + MCP  │───▶│   search-core    │───▶│   pySideCar     │
│     Agent       │    │   (Go Backend)   │    │ (Python ML/NLP) │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                                │                       │
                                ▼                       ▼
                       ┌────────────────────────────────────────┐
                       │            **go-api-boot**             │
                       │(Bundled api-gRpc, ODM-Mongo (Search),  │
                       │         Workers-Temporal, az blob)     │
                       └────────────────────────────────────────┘

```

### 🎯 The Perfect Fusion

**Go Backend (`search-core`)**
- High-performance gRPC services
- Temporal workers for orchestration
- Vector & text search endpoints
- Authentication & multi-tenancy
- Powered by [go-api-boot](https://github.com/SaiNageswarS/go-api-boot) 

**Python ML Pipeline (`pySideCar`)**
- PDF → Markdown conversion (pymupdf4llm)
- Medical entity extraction (SciSpacy + UMLS)
- Intelligent text chunking with sentence boundaries
- Advanced windowing strategies

**Claude MCP Agent (`mcp-agent`)**
- Real-time health insights from journal articles
- Seamless integration with Claude Desktop
- Context-aware query processing

## 🚀 Quick Start

### Prerequisites

- Go 1.23+
- Python 3.11+
- MongoDB with Vector Search
- Azure Blob Storage
- Temporal.io cluster

### 1. Clone & Setup

```bash
git clone https://github.com/your-org/agent-boot
cd agent-boot

# Setup Go backend
cd search-core
go mod download
```

### 2. Environment Configuration

```bash
# .env file
MONGODB_URI=mongodb://localhost:27017
AZURE_STORAGE_ACCOUNT=your_account
AZURE_STORAGE_KEY=your_key
TEMPORAL_HOST_PORT=localhost:7233
ANTHROPIC_API_KEY=your_key
JINA_API_KEY=your_key
SEARCH_CORE_AUTH_TOKEN=your_token
```

### 3. Generate Protocol Buffers

```bash
cd proto
./build.sh
```

### 4. Start the Backend

```bash
cd search-core
go run main.go
```

### 5. Launch Python ML Worker

```bash
cd pySideCar
pip install -r requirements.txt
python main.py
```

### 6. Setup MCP Agent

```bash
cd mcp-agent
go run main.go
```

Add to your Claude Desktop config:
```json
{
  "mcpServers": {
    "agent-boot": {
      "command": "./mcp-agent",
      "args": []
    }
  }
}
```

## 📖 Usage

### Document Processing

Upload a PDF to trigger the complete processing pipeline:

```bash
# Upload document
curl -X POST http://localhost:8080/upload \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -F "file=@research_paper.pdf" \
  -F "tenant=healthcare"
```

The system automatically:
1. **Converts** PDF → Markdown
2. **Chunks** into logical sections
3. **Extracts** medical entities (UMLS)
4. **Embeds** using Jina AI
5. **Indexes** for hybrid search

### Querying with Claude

Simply ask Claude health-related questions:

> "What are the latest treatments for Type 2 diabetes?"

The MCP agent will:
- Process your query
- Search the knowledge base
- Return relevant journal insights with citations

### Direct API Access

```bash
# Search endpoint
curl -X POST http://localhost:50051/search \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -d '{
    "queries": ["diabetes treatment", "insulin resistance"]
  }'
```

## 🔧 Configuration

### Search Core (`config.ini`)

```ini
[dev]
sign_up_allowed = true
vector_search_enabled = true
text_search_enabled = true
temporal_host_port = localhost:7233
temporal_py_task_queue = searchCorePySideCar
```

### Python Sidecar

```python
# Enhanced medical entity processing
MEDICAL_ENTITIES = "medical_entities"

# Configure chunking parameters
WINDOW_SIZE = 700      # Max tokens per chunk
STRIDE = 600          # Overlap between chunks
MIN_SECTION_BYTES = 4000  # Minimum section size
```

## 🏥 Medical AI Enhancement

Agent Boot includes specialized medical AI capabilities:

- **Entity Linking**: UMLS integration for medical concept recognition
- **Section Intelligence**: Hierarchical document understanding  
- **Confidence Filtering**: Only high-quality entity extractions (85%+ confidence)
- **Abbreviation Handling**: Medical acronym resolution
- **Citation Tracking**: Source attribution for all insights

## 🔒 Security & Multi-tenancy

- **JWT Authentication**: Secure API access
- **Tenant Isolation**: Complete data separation
- **Azure Integration**: Enterprise-grade security
- **Input Validation**: Comprehensive request sanitization

## 📊 Performance

- **Sub-second search**: Optimized vector operations
- **Concurrent processing**: Temporal workflow orchestration
- **Memory efficient**: Streaming document processing
- **Auto-scaling**: Cloud-native architecture

## 🛠️ Development

### Project Structure

```
agent-boot/
├── search-core/          # Go backend services
│   ├── services/         # gRPC implementations  
│   ├── workers/          # Temporal activities
│   └── db/              # MongoDB models
├── pySideCar/           # Python ML pipeline
│   └── workers/         # Document processing
├── mcp-agent/           # Claude MCP integration
└── proto/               # Protocol buffer definitions
```

### Adding New Domains

1. **Define Enhancement**: Add to `indexer_types.py`
2. **Create Processor**: Implement entity extraction logic
3. **Update Workflow**: Modify `window_section_chunks` activity
4. **Configure Search**: Adjust indexing parameters

### Testing

```bash
# Go tests
cd search-core && go test ./...

# Python tests  
cd pySideCar && python -m pytest

# Integration tests
make test-integration
```

## 🤝 Contributing

We welcome contributions! Please follow these steps:

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Submit a pull request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- **[go-api-boot](https://github.com/SaiNageswarS/go-api-boot)**: The fantastic Go framework powering our backend
- **SciSpacy**: Medical NLP capabilities
- **Temporal.io**: Workflow orchestration
- **Anthropic**: Claude AI integration
- **Jina AI**: Vector embeddings

---

**Built with ❤️ for the AI-powered future**