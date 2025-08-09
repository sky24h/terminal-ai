# Terminal AI Project Configuration

## Project Overview
Terminal AI is a high-performance CLI tool for interacting with OpenAI models, focusing on speed and efficiency with special support for GPT-5 series and reasoning models.

## Project-Specific Guidelines

### Code Standards
- **Language**: Go 1.21+
- **Framework**: Cobra for CLI, OpenAI official Go SDK v2
- **Configuration**: Viper for config management
- **UI Libraries**: Charmbracelet (bubbles, lipgloss) for terminal UI
- **Logging**: Zerolog with suppression in simple mode

### Model Configuration
- **Default Model**: gpt-5-mini
- **Reasoning Models**: gpt-5 series, o1/o3/o4 series
- **Temperature**: Always 1.0 for reasoning models
- **Service Tier**: "default" for all models unless overridden via --service-tier flag

### Key Features to Maintain
1. **Command Modes**:
   - `-q`: Query mode with concise responses
   - `-s`: Shell mode (default) with execution confirmation
   - `-c`: Interactive chat mode

2. **Service Tier Support**:
   - Options: auto, default, priority, flex, scale
   - Command flag: --service-tier
   - Config: service_tier in YAML

3. **Visual Design**:
   - User input: Cyan color (#86)
   - AI responses: Rose red (#211)
   - No "You:" or "AI:" labels in shell mode
   - Suppressed logs in simple mode

### Configuration Paths
- **Config File**: `~/.opt/terminal-ai-config.yaml`
- **Cache Directory**: `~/.terminal-ai/cache`
- **API Key**: Via OPENAI_API_KEY env or config

### Testing Requirements
- Test all service tier options when modifying API calls
- Verify reasoning_effort parameter for reasoning models
- Check color output in different terminal environments
- Validate command execution in shell mode

### Development Workflow
1. **Before Changes**:
   - Review existing patterns in codebase
   - Check imports and dependencies
   - Understand current model configuration

2. **During Development**:
   - Maintain existing code style
   - Preserve color theming
   - Keep logging suppressed in simple mode
   - Test with both GPT-5 and non-reasoning models

3. **After Changes**:
   - Run `go build` to verify compilation
   - Test all command modes (-q, -s, -c)
   - Verify service tier behavior
   - Update documentation if needed

### Critical Files
- `cmd/simple.go`: Main command interface with flags
- `internal/config/config.go`: Configuration and model logic
- `internal/ai/client.go`: OpenAI client implementation
- `internal/ai/stream.go`: Streaming response handler
- `config.example.yaml`: Configuration template

### Dependencies
```go
github.com/openai/openai-go/v2  // OpenAI official SDK
github.com/spf13/cobra          // CLI framework
github.com/spf13/viper          // Configuration
github.com/charmbracelet/lipgloss // Terminal styling
github.com/rs/zerolog           // Logging
```

### Common Tasks

#### Adding a New Model
1. Update `IsReasoningModel()` in `internal/config/config.go`
2. Add to model lists in documentation
3. Test temperature and reasoning_effort handling

#### Modifying Service Tier Logic
1. Update `GetRecommendedServiceTier()` in config
2. Modify client.go and stream.go for API calls
3. Test all tier options

#### Changing Visual Output
1. Update Theme in `internal/ui/styles.go`
2. Modify display logic in `cmd/simple.go` or `cmd/chat.go`
3. Test in both light and dark terminals

### Do NOT
- Change default service tier from "default"
- Add automatic priority for any models
- Show logs in simple mode
- Add emoji unless explicitly requested
- Create unnecessary files
- Modify config path from ~/.opt/terminal-ai-config.yaml

### Testing Checklist
- [ ] Build succeeds: `go build -o terminal-ai main.go`
- [ ] Query mode works: `./terminal-ai -q "test"`
- [ ] Shell mode works: `./terminal-ai -s "ls"`
- [ ] Chat mode works: `./terminal-ai -c`
- [ ] Service tier override works: `./terminal-ai -q "test" --service-tier priority`
- [ ] Colors display correctly
- [ ] No logs appear in output