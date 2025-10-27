# 🤖 Hi AI CLI -- Make Ops Smarter, Make Commands Simpler.

Hi AI is a CLI tool designed specifically for server operations and personal host management. By integrating AI capabilities, it helps users quickly find and execute system commands, improving operational efficiency. It can intelligently parse user requirements and provide precise command suggestions and execution solutions, making it an invaluable assistant for operation engineers.

[中文](./README.md) · **English** ·  [Report Bug][github-issues-link] · [Request Feature][github-issues-link]

## ✨ Features

- 🤖 Intelligent Dialogue: Support natural language conversations with AI to quickly find and apply commands
- 📚 Command Manual Mode: `hi man`，use large language model to help you with command manual queries.
- 🔍 Smart Context Awareness: Intelligently perceives the current working environment for more accurate AI responses
- 🖥️ Cross-Platform Compatibility: Supports Linux, MacOS, Windows (arm, amd architectures) platforms
- 🌍 Multi-language Support: Built-in internationalization support, providing multi-language interface (currently supports Chinese and English)
- ⚙️ Configuration Management: Supports custom configuration, including API key settings
- 📝 Logging: Detailed logging for easy troubleshooting

## 📦 Installation

### 📋 Prerequisites

- Go 1.22 or higher
- Git

### 📝 Installation Steps

#### Method 1. 📦 Binary Installation
```bash
# One-line installation (English version)
# Install latest version by default
curl -fSsL https://raw.githubusercontent.com/zdt1013/hi-ai-cli/main/install.sh | bash
# Install latest version with script acceleration
curl -fSsL https://ghproxy.net/https://raw.githubusercontent.com/zdt1013/hi-ai-cli/main/install.sh | bash

# Install specific version with acceleration source
curl -fSsL https://raw.githubusercontent.com/zdt1013/hi-ai-cli/main/install.sh | bash -s -- -v v0.1.1 -m ghproxy
# Install latest version with script acceleration
curl -fSsL https://ghproxy.net/https://raw.githubusercontent.com/zdt1013/hi-ai-cli/main/install.sh | bash -s -- -m ghproxy
```

```bash
# Step-by-step installation
# Download installation script
curl -o install.sh https://raw.githubusercontent.com/zdt1013/hi-ai-cli/main/install.sh
# Download installation script with acceleration source
curl -o install.sh https://ghproxy.net/https://raw.githubusercontent.com/zdt1013/hi-ai-cli/main/install.sh

# Add execution permission
chmod +x install.sh

# Run installation script (install latest version by default)
sudo ./install.sh

# Or install specific version
sudo ./install.sh -v v0.1.1

# Install with acceleration source (options: ghproxy, wgetla)
sudo ./install.sh -v v0.1.1 -m ghproxy
```

#### Method 2. 🚀 Local Compilation
1. Clone repository:
```bash
git clone https://github.com/zdt1013/hi-ai-cli.git
cd hi-ai-cli
```

2. Install dependencies:
```bash
go mod download
```

3. Build project:
```bash
go build
```

4. Add the compiled executable to system PATH (optional)

## 🚀 Usage

### ⌨️ Single Question Mode

```bash
# Basic format
> hi + [input any question]
```

![Single Question Mode Example](docs/example1.png)

### 💬 Start Dialogue Mode
```bash
# Basic format
> hi chat + [input first question]
Or
> hi chat <enter>
  <input any question>
```
![Dialogue Mode Example](docs/example2.png)

### 📚 Command Manual Mode (man mode)
> Use large language model to help you with command manual queries.

```bash
# Basic format
> hi man -c curl [optional additional question]
# Example
> hi man -c curl "focus on fSsL parameter"
```
![man mode example](docs/example3.png)

### 🛠️ Other Commands
```bash
# View configuration
> hi config --help
# View help information
> hi --help
```

### 🔧 Configuration

Before using Hi AI, you need to configure necessary parameters, such as API keys. You can configure through the following commands:

```bash
> hi config --apiKey YOUR_API_KEY --baseUrl YOUR_API_BASE --model YOUR_API_MODEL
or
> hi config -k YOUR_API_KEY -u YOUR_API_BASE -m YOUR_API_MODEL
```

## 📁 Project Structure

```
hi-ai-cli/
├── action/     # Command action implementation
├── assets/     # Static resources
│   └── lang/   # Language packs
├── cmd/        # Command line definitions
├── docs/       # Documentation and attachments
├── common/     # Common components
├── execute/    # Executor
├── logger/     # Logging module
├── model/      # Data models
├── setup/      # Initialization setup
├── validate/   # Validator
├── wenai/      # Core functionality implementation
├── main.go     # Program entry
├── go.mod      # Go module definition
└── go.sum      # Go module dependency verification
```

## ⚠️ Known Issues
  * Non-command line related question responses, reply style not yet optimized

## 🔮 Future Plans
 * Awareness: Running host user, non-sudo user, intelligent command adjustment
 * Awareness: Installed and uninstalled commands on local machine
 * Tool chain (functioncall, mcp) support
 * Thinking model compatibility
 * User system for easy installation and use without manual AI parameter configuration
 * User knowledge base and preferences for saving usage habits, making Hi AI understand you better

## 📚 Dependencies on Open Source Projects
 * [urfave/cli](https://github.com/urfave/cli) - Command line application framework
 * [cloudwego/eino](https://github.com/cloudwego/eino) - ByteDance's open-source Large Language Model (LLM) application development framework
 * [gookit/config](https://github.com/gookit/config) - Configuration management library
 * [gookit/i18n](https://github.com/gookit/i18n) - Internationalization support
 * [shirou/gopsutil](https://github.com/shirou/gopsutil) - System information collection
 * [sashabaranov/go-openai](https://github.com/sashabaranov/go-openai) - OpenAI API client
 * [gookit/slog](https://github.com/gookit/slog) - Logging library
 * [go-cmd/cmd](https://github.com/go-cmd/cmd) - Command execution library
 * [manifoldco/promptui](https://github.com/manifoldco/promptui) - Interactive command line interface

## 🤝 Contribution Guidelines

Welcome to submit Issues and Pull Requests to help improve this project. Before submitting code, please ensure:

1. Code complies with project coding standards
2. Necessary tests have been added
3. Related documentation has been updated

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details

## 📞 Contact

If you have any questions or suggestions, please contact through:

- Submit an Issue
