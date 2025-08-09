# Terminal AI

A blazing-fast terminal AI assistant built with Go, featuring streaming responses, intelligent caching, and a beautiful CLI interface.

## Features

- ⚡ **Lightning Fast**: Cold start in <30ms, sub-millisecond cached responses
- 🔄 **Streaming Responses**: Real-time token-by-token output with progress indicators
- 💾 **Smart Caching**: LRU cache with TTL for instant repeated queries
- 🎨 **Beautiful UI**: Themed terminal output with markdown rendering and syntax highlighting
- 💬 **Interactive Chat**: Maintain context across conversations with history management
- 🔧 **Highly Configurable**: YAML config, environment variables, and CLI flags
- 🔒 **Secure**: API keys never logged, secure storage with validation
- 📊 **Metrics**: Token usage tracking, cache statistics, performance monitoring
- 🌊 **Robust**: Exponential backoff, rate limiting, connection pooling
- 🧠 **Reasoning Models**: Full support for GPT-5 and O-series models with configurable reasoning effort

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/user/terminal-ai.git
cd terminal-ai

# Build and install
make install
```

### Using Go

```bash
go install github.com/user/terminal-ai@latest
```

## Quick Start

### 1. Set up your OpenAI API key:

```bash
# Option 1: Environment variable (recommended)
export OPENAI_API_KEY="sk-..."

# Option 2: Configuration wizard
terminal-ai config --init

# Option 3: Direct configuration
terminal-ai config set openai.api_key "sk-..."

# Option 4: Config file (~/.opt/terminal-ai-config.yaml)
mkdir -p ~/.opt
echo "openai:
  api_key: sk-..." > ~/.opt/terminal-ai-config.yaml
```

### 2. Test your configuration:

```bash
# Validate configuration
terminal-ai config --validate

# Test API connection
terminal-ai config --test
```

### 3. Start using the assistant:

```bash
# Shell command mode (-s or default)
terminal-ai -s "list all docker containers"
terminal-ai "find large files"  # Same as above, defaults to -s

# Query mode for concise answers
terminal-ai -q "What is Docker?"

# Interactive chat mode (-c)
terminal-ai -c

# With options
terminal-ai -q -m gpt-5 "Explain quantum computing"
terminal-ai -s --service-tier priority "find large files"
terminal-ai --no-stream "list processes"
```

## Configuration

Configuration can be set via (in order of precedence):
1. Command-line flags
2. Environment variables (prefix: `TERMINAL_AI_`)
3. Configuration file (`~/.opt/terminal-ai-config.yaml` or `./.terminal-ai.yaml`)
4. Default values

### Configuration File

Create `~/.opt/terminal-ai-config.yaml`:

```yaml
openai:
  api_key: ${OPENAI_API_KEY}  # Can reference environment variables
  model: gpt-5-mini            # Default reasoning model
  max_tokens: 2000
  temperature: 1.0             # Must be 1.0 for reasoning models
  reasoning_effort: low        # low, medium, high
  service_tier: default        # auto, default, priority, flex, scale
  timeout: 30s
  organization: ""             # Optional: OpenAI organization ID

# Model Types:
# Reasoning (temp=1.0): gpt-5, gpt-5-mini, gpt-5-nano, o1, o1-mini, o3, o3-mini, o4-mini
# Non-reasoning: gpt-4.1, gpt-4.1-mini, gpt-4.1-nano, gpt-4o, gpt-4o-mini

# Service Tiers:
# - auto: Uses project settings
# - default: Standard processing (default for all models)
# - priority: Faster performance (enterprise only)
# - flex: Non-time-sensitive tasks
# - scale: Dedicated capacity

cache:
  enabled: true
  ttl: 5m  # Cache time-to-live
  max_size: 100  # Maximum cache size in MB
  strategy: lru  # Eviction strategy

ui:
  theme: dark  # dark or light
  streaming_enabled: true
  markdown_rendering: true
  color_output: true

logging:
  level: info  # trace, debug, info, warn, error
  format: json  # json or text
  file: ""  # Optional: log file path
```

### Environment Variables

All configuration options can be set via environment variables:

```bash
# OpenAI settings
export TERMINAL_AI_OPENAI_API_KEY="sk-..."
export TERMINAL_AI_OPENAI_MODEL="gpt-5-mini"
export TERMINAL_AI_OPENAI_TEMPERATURE="0.7"
export TERMINAL_AI_OPENAI_MAX_TOKENS="2000"

# Cache settings  
export TERMINAL_AI_CACHE_ENABLED="true"
export TERMINAL_AI_CACHE_TTL="5m"
export TERMINAL_AI_CACHE_MAX_SIZE="100"

# Service tier
export TERMINAL_AI_OPENAI_SERVICE_TIER="default"
export TERMINAL_AI_OPENAI_REASONING_EFFORT="low"

# UI settings
export TERMINAL_AI_UI_THEME="dark"
export TERMINAL_AI_UI_STREAMING_ENABLED="true"

# Logging
export TERMINAL_AI_LOGGING_LEVEL="info"
```

## Usage Modes

### Query Mode (`-q`)
Quick one-off questions with concise answers:

```bash
terminal-ai -q "What is Docker?"
terminal-ai -q "How to reverse a string in Python?"
```

### Shell Mode (`-s`) - DEFAULT
Generate and optionally execute shell commands:

```bash
terminal-ai "find all log files larger than 100MB"  # defaults to -s
terminal-ai -s "list docker containers"
# Output:
# find / -name "*.log" -size +100M 2>/dev/null
# Execute? [Enter/E=Execute, N=Refine, Q=Quit]: 
```

Interactive refinement:
- Press Enter or E to execute the command
- Press N to provide feedback and get a new suggestion
- Press Q to quit

### Chat Mode (`-c`)
Interactive conversation with context:

```bash
terminal-ai -c
```

**Chat Commands:**
- `/help` - Show available commands
- `/clear` - Clear conversation history
- `/save` - Save current conversation
- `/load` - Load a saved conversation
- `/export` - Export conversation as markdown
- `/model` - Change the AI model
- `/system` - Set system prompt
- `/multiline` - Toggle multiline input mode
- `/history` - Show conversation history
- `/exit` - Exit chat session

## Global Options

```bash
-q, --query                 Query mode for questions
-s, --shell                 Shell command generator mode (default when text provided)
-c, --chat                  Interactive chat mode
-m, --model string          Override default model
    --service-tier string   Service tier (auto, default, priority, flex, scale)
    --stream                Enable streaming (default true)
    --no-stream             Disable streaming
-v, --verbose               Verbose output
    --no-color              Disable colored output
    --config file           Custom config file path
```

## Legacy Commands

The following commands are still available for backward compatibility:

### `query` - Quick Query

```bash
terminal-ai query "your question" [flags]
```

### `chat` - Interactive Chat

```bash
terminal-ai chat [flags]
```

### `config` - Configuration Management

Manage application configuration:

```bash
terminal-ai config [flags]
terminal-ai config set [key] [value]
terminal-ai config get [key]

Flags:
      --init       Initialize configuration with wizard
      --show       Display current configuration
      --validate   Validate configuration
      --test       Test API connection
      --edit       Edit config in default editor
      --location   Show config file locations

Examples:
  terminal-ai config --init                    # Run setup wizard
  terminal-ai config set openai.model gpt-5-mini    # Set model
  terminal-ai config get openai.model          # Get current model
  terminal-ai config --test                    # Test API connection
```

### `cache` - Cache Management

Manage the response cache for improved performance:

```bash
terminal-ai cache [flags]

Flags:
      --stats         Show cache statistics
      --clear         Clear all cached responses
      --invalidate    Invalidate entries matching pattern
      --size          Show cache size information

Examples:
  terminal-ai cache --stats                    # View hit/miss rates
  terminal-ai cache --clear                    # Clear all cache
  terminal-ai cache --invalidate "chat_*"      # Clear chat cache
```

## Performance

- **Cold Start**: ~27ms
- **Memory Usage**: ~11.5MB idle, <50MB active
- **Cached Response**: <1ms (1500x faster than API calls)
- **First Token Latency**: <2s with streaming
- **Concurrent Requests**: Non-blocking architecture
- **Cache Hit Rate**: Typically >30% in normal usage
- **Connection Pooling**: Reuses connections for efficiency

## Security

### API Key Storage

The application follows security best practices for API key management:

1. **Never Logged**: API keys are automatically masked in all log output
2. **Secure Storage**: Keys stored with 0600 permissions (owner read/write only)
3. **Environment Variables**: Supports secure key injection via environment
4. **Validation**: Keys are validated before use
5. **No Hardcoding**: No default keys or credentials in code

### Best Practices

```bash
# Recommended: Use environment variable
export OPENAI_API_KEY="sk-..."

# Secure file storage (auto-created with proper permissions)
terminal-ai config --init

# Never commit API keys to version control
echo "config.yaml" >> .gitignore
```

## Development

### Building

```bash
# Get dependencies
go mod download

# Build for current platform
make build

# Build for all platforms
make build-all

# Run tests
make test

# Run linters
make lint

# Format code
make fmt

# Run with hot reload (development)
make dev
```

### Project Structure

```
terminal-ai/
├── cmd/           # CLI command implementations
├── internal/      # Internal packages
│   ├── ai/        # OpenAI client with caching
│   ├── config/    # Configuration management
│   ├── ui/        # Terminal UI components
│   └── utils/     # Logging, errors, metrics
├── pkg/           # Public interfaces
│   └── models/    # Data models
├── examples/      # Usage examples
├── docs/          # Documentation
└── tests/         # Test suites
```

### Testing

```bash
# Run all tests
make test

# Run with coverage
go test -cover ./...

# Run specific package tests
go test ./internal/ai/...

# Run benchmarks
go test -bench=. ./...
```

## Examples

### Basic Usage Examples

```bash
# Simple query with concise output
terminal-ai -q "How to check disk usage in Linux?"

# Shell command with priority processing
terminal-ai --service-tier priority "monitor system performance"

# Reasoning model with high effort
terminal-ai -m o3-mini -q "Explain the algorithm for..."
```

### Advanced Examples

```bash
# Interactive refinement in shell mode
terminal-ai "backup database safely"
# Shows command, press N to refine, Enter to execute

# Chat mode with custom model
terminal-ai -c -m gpt-5

# Disable streaming for scripting
terminal-ai --no-stream -q "list git commands" > commands.txt
```

Check out the `examples/` directory for detailed usage examples:

- `examples/client/` - OpenAI client usage patterns
- `examples/ui/` - Terminal UI components demo
- `examples/cache/` - Caching strategies
- `examples/config/` - Configuration examples

## Troubleshooting

### Common Issues

1. **API Key Not Found**
   - Ensure `OPENAI_API_KEY` is set or run `terminal-ai config --init`

2. **Connection Timeout**
   - Check internet connection
   - Verify API endpoint is accessible
   - Adjust timeout in config: `openai.timeout: 60s`

3. **Cache Not Working**
   - Ensure cache is enabled: `cache.enabled: true`
   - Check cache directory permissions
   - Clear corrupted cache: `terminal-ai cache --clear`

4. **High Token Usage**
   - Reduce `max_tokens` in configuration
   - Use more specific prompts
   - Monitor usage with `--tokens` flag

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing`)
5. Open a Pull Request

## License

MIT License - see LICENSE file for details

## Support

- **Issues**: [GitHub Issues](https://github.com/user/terminal-ai/issues)
- **Documentation**: [docs/](./docs/)
- **Examples**: [examples/](./examples/)