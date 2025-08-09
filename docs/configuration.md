# Configuration Management System

The Terminal AI application uses a robust, multi-source configuration system built with Viper that supports environment variables, configuration files, and profiles.

## Configuration Sources (Priority Order)

1. **Command-line flags** (highest priority) - When implemented via cobra
2. **Environment variables** - `TERMINAL_AI_*` prefix or standard vars like `OPENAI_API_KEY`
3. **Configuration file** - YAML format at multiple locations
4. **Default values** (lowest priority)

## Configuration File Locations

The system searches for configuration files in the following order:
- `./.terminal-ai.yaml` - Project-specific configuration
- `~/.terminal-ai/config.yaml` - User-specific configuration
- `~/.terminal-ai.yaml` - Alternative user configuration
- `/etc/terminal-ai/config.yaml` - System-wide configuration

## Environment Variables

All configuration values can be set via environment variables with the `TERMINAL_AI_` prefix:

```bash
# OpenAI settings
export TERMINAL_AI_OPENAI_API_KEY="your-key"
export TERMINAL_AI_OPENAI_MODEL="gpt-5-mini"
export TERMINAL_AI_OPENAI_MAX_TOKENS="4000"
export TERMINAL_AI_OPENAI_TEMPERATURE="1.0"
export TERMINAL_AI_OPENAI_SERVICE_TIER="default"
export TERMINAL_AI_OPENAI_REASONING_EFFORT="low"

# Cache settings
export TERMINAL_AI_CACHE_ENABLED="true"
export TERMINAL_AI_CACHE_TTL="10m"
export TERMINAL_AI_CACHE_MAX_SIZE="200"

# UI settings
export TERMINAL_AI_UI_THEME="dark"
export TERMINAL_AI_UI_STREAMING_ENABLED="true"

# Logging
export TERMINAL_AI_LOGGING_LEVEL="debug"
export TERMINAL_AI_LOGGING_FORMAT="json"
```

The system also supports standard OpenAI environment variables:
- `OPENAI_API_KEY` - Automatically detected if `TERMINAL_AI_OPENAI_API_KEY` is not set
- `OPENAI_ORG_ID` - Organization ID for OpenAI API

## Configuration File Format

Create a `config.yaml` file with the following structure:

```yaml
# Profile selection
profile: prod  # Options: dev, prod, custom

# OpenAI Configuration
openai:
  api_key: ${OPENAI_API_KEY}  # Use env var reference for security
  model: gpt-5-mini            # Default reasoning model
  max_tokens: 2000
  temperature: 1.0             # Must be 1.0 for reasoning models
  reasoning_effort: low        # low, medium, high (for reasoning models)
  service_tier: default        # auto, default, priority, flex, scale
  top_p: 1.0
  n: 1
  timeout: 30s
  base_url: https://api.openai.com/v1
  org_id: ""
  stop: []

# Model Types:
# Reasoning (temp=1.0): gpt-5, gpt-5-mini, gpt-5-nano, o1, o1-mini, o3, o3-mini, o4-mini
# Non-reasoning: gpt-4.1, gpt-4.1-mini, gpt-4.1-nano, gpt-4o, gpt-4o-mini

# Service Tiers:
# - auto: Uses project settings
# - default: Standard processing (default for all models)
# - priority: Faster performance (enterprise only)
# - flex: Non-time-sensitive tasks
# - scale: Dedicated capacity

# Cache Configuration
cache:
  enabled: true
  ttl: 5m  # Duration format: 5m, 1h, 30s
  max_size: 100  # MB
  strategy: lru  # Options: lru, lfu, fifo
  dir: ${HOME}/.terminal-ai/cache

# UI Configuration
ui:
  streaming_enabled: true
  color_output: true
  markdown_rendering: true
  syntax_highlighting: true
  theme: auto  # Options: dark, light, auto
  spinner: dots  # Options: dots, line, star, arrow
  width: 0  # 0 for auto-detect

# Logging Configuration
logging:
  level: info  # Options: debug, info, warn, error, fatal, panic
  format: json  # Options: json, text, pretty
  file: ""  # Log file path (empty for stdout)
  no_api: true  # Never log API keys
```

## Configuration Profiles

The system supports different profiles for different environments:

### Development Profile (`dev`)
- Debug logging enabled
- Pretty log formatting
- Extended timeouts
- Cache may be disabled for testing

### Production Profile (`prod`)
- Info-level logging
- JSON log formatting
- Standard timeouts
- Cache enabled with LRU strategy
- API keys always masked

### Custom Profile
Define your own profile settings in the configuration file.

To use a specific profile:
```bash
export TERMINAL_AI_PROFILE=dev
# Or in code:
config.LoadWithProfile("", "dev")
```

## Secure API Key Management

### Option 1: Environment Variable
```bash
export OPENAI_API_KEY="sk-..."
```

### Option 2: Secure File
```bash
# Create a file with restricted permissions
echo "sk-..." > ~/.terminal-ai/api.key
chmod 600 ~/.terminal-ai/api.key

# Reference the file
export TERMINAL_AI_API_KEY_FILE=~/.terminal-ai/api.key
```

### Option 3: Configuration File
Use environment variable references in your config:
```yaml
openai:
  api_key: ${OPENAI_API_KEY}
```

## Validation

The configuration system performs comprehensive validation:

- **API Key**: Format validation, presence check
- **Model**: Validates against supported OpenAI models (including GPT-5 and O-series)
- **Temperature**: Must be between 0 and 2 (automatically set to 1.0 for reasoning models)
- **Reasoning Effort**: Must be low, medium, or high for reasoning models
- **Service Tier**: Must be auto, default, priority, flex, or scale
- **Top-p**: Must be between 0 and 1
- **Max Tokens**: Model-specific limits enforced
- **Timeout**: Minimum 5 seconds, maximum 5 minutes
- **Cache Size**: Maximum 10GB
- **UI Theme**: Must be dark, light, or auto
- **Log Level**: Valid log levels only
- **File Permissions**: API key files must have 0600 permissions

## Usage in Code

### Basic Usage
```go
import "github.com/user/terminal-ai/internal/config"

// Load configuration
cfg, err := config.Load("")
if err != nil {
    log.Fatal(err)
}

// Access values
model := cfg.OpenAI.Model
cacheEnabled := cfg.Cache.Enabled
```

### With Custom Config File
```go
cfg, err := config.Load("/path/to/config.yaml")
```

### With Profile
```go
cfg, err := config.LoadWithProfile("", "dev")
```

### Saving Configuration
```go
// Save to default location
err := cfg.Save()

// Save to specific location
err := cfg.SaveTo("/path/to/config.yaml")
```

### Accessing Nested Values
```go
// Use the GetString helper
model := cfg.GetString("openai.model")
logLevel := cfg.GetString("logging.level")
```

## Best Practices

1. **Never commit API keys** - Always use environment variables or secure files
2. **Use profiles** - Different settings for dev/prod environments
3. **Validate early** - Configuration is validated on load
4. **Secure file permissions** - API key files should be 0600
5. **Mask sensitive data** - API keys are automatically masked in logs and saved configs
6. **Use environment variables** - Override settings without changing files
7. **Project-specific configs** - Use `.terminal-ai.yaml` for project overrides

## Troubleshooting

### Missing API Key
```
Error: OpenAI API key is required
Solution: Set OPENAI_API_KEY environment variable
```

### Invalid Model
```
Error: unsupported model: xxx
Solution: Use a valid OpenAI model name (gpt-5-mini, gpt-5, o1, o3, gpt-4o, etc.)
```

### Permission Denied
```
Error: API key file has insecure permissions
Solution: chmod 600 /path/to/api.key
```

### Cache Directory Issues
```
Error: cache directory is not writable
Solution: Ensure the cache directory exists and is writable
```