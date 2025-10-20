# Crontab Expression Editor

> **Keywords:** cron · crontab · terminal · TUI · bubbletea · CLI · scheduler · Go · developer-tools · interactive

🕐 An interactive terminal-based cron expression editor built with Go and Bubble Tea. Create, edit, and validate crontab schedules with real-time human-readable descriptions and next run time calculations.

![Version](https://img.shields.io/badge/version-1.0.0-blue)
![Go Version](https://img.shields.io/badge/go-1.25.3-00ADD8?logo=go)
![Coverage](https://img.shields.io/badge/coverage-93.6%25-brightgreen)
![Tests](https://img.shields.io/badge/tests-72%20passing-success)
![License](https://img.shields.io/badge/license-MIT-green)

## ✨ Features

- 🎨 **Beautiful TUI Interface** - Clean, colorful terminal interface with responsive design
- ⚡ **Real-time Validation** - Instant feedback as you type with field-aware validation
- 📝 **Human-Readable Descriptions** - Converts cron expressions to natural language
- 📋 **Clipboard Integration** - Copy cron expressions to clipboard with one keystroke
- 🛡️ **Robust Error Handling** - Comprehensive validation prevents crashes and invalid expressions
- ⌨️ **Intuitive Navigation** - Tab, arrow keys, and shortcuts for efficient editing
- 🔍 **Field-Specific Validation** - Smart validation for each cron field (minute, hour, day, month, weekday)
- 📊 **Next Execution Times** - Preview when your cron job will run next

## 🚀 Installation

### 📦 Quick Install (Recommended)

#### Using Go Install

```bash
go install github.com/techquestsdev/crontab-guru@latest
```

Then run with: `x` (or `crontab-guru` if renamed)

#### Using Homebrew (macOS/Linux)

```bash
# Coming soon - will be available after first release
brew tap techquestsdev/tap
brew install crontab-guru
```

#### Download Binary

Download the latest release for your platform from the [Releases page](https://github.com/techquestsdev/crontab-guru/releases):

- **Linux** (amd64, arm64)
- **macOS** (Intel, Apple Silicon)
- **Windows** (amd64)

```bash
# Example: Download and install on Linux/macOS
curl -L https://github.com/techquestsdev/crontab-guru/releases/latest/download/crontab-guru_Linux_x86_64.tar.gz | tar xz
sudo mv crontab-guru /usr/local/bin/
```

### 🛠️ Build from Source

#### Prerequisites

- **Go** 1.25.3 or higher
- **Terminal** with color support
- **Make** (optional, for using Makefile commands)

#### Clone and Build

```bash
# Clone the repository
git clone https://github.com/techquestsdev/crontab-guru.git
cd crontab-guru

# Download dependencies
go mod download

# Build the application
make build
# Or without Make:
go build -o bin/crontab-guru .

# Install to your system
make install
# Or without Make:
go install

# Run it
./bin/crontab-guru
```

#### Using Make (Developer Workflow)

```bash
# See all available commands (23 targets)
make help

# Build the application
make build

# Run tests with coverage
make test-coverage

# Check code quality (fmt + vet + lint)
make check
```

#### Quick Run (Development)

```bash
make run
# Or without Make:
go run main.go
```

## 📖 Usage

### Basic Usage

1. Launch the application
2. Use **Tab/Space/Enter** to navigate between fields
3. Type your cron expression values
4. See the description update in real-time
5. Press **y** to copy the expression to clipboard
6. Press **Esc** or **Ctrl+C** to quit

### Keyboard Shortcuts

| Key                       | Action                             |
| ------------------------- | ---------------------------------- |
| `?`                       | Toggle help text                   |
| `Tab` / `Space` / `Enter` | Navigate between fields (forward)  |
| `Shift+Tab`               | Navigate between fields (backward) |
| `y`                       | Copy cron expression to clipboard  |
| `Esc` / `Ctrl+C`          | Quit application                   |

## 🎯 Cron Expression Format

The editor uses the standard cron format with 5 fields:

```text
┌───────────── minute (0 - 59)
│ ┌───────────── hour (0 - 23)
│ │ ┌───────────── day of month (1 - 31)
│ │ │ ┌───────────── month (1 - 12 or JAN-DEC)
│ │ │ │ ┌───────────── day of week (0 - 6 or SUN-SAT)
│ │ │ │ │
* * * * *
```

### Supported Syntax

- **Wildcards**: `*` (any value)
- **Specific values**: `5` (at minute 5)
- **Ranges**: `1-5` (1 through 5)
- **Steps**: `*/15` (every 15 minutes)
- **Lists**: `1,15,30` (at 1, 15, and 30)
- **Month names**: `JAN`, `FEB`, `MAR`, etc. (month field only)
- **Day names**: `SUN`, `MON`, `TUE`, etc. (weekday field only)

### Examples

| Expression        | Description              |
| ----------------- | ------------------------ |
| `* * * * *`       | Every minute             |
| `0 * * * *`       | Every hour               |
| `0 0 * * *`       | Every day at midnight    |
| `0 9 * * MON-FRI` | At 9:00 AM on weekdays   |
| `*/15 * * * *`    | Every 15 minutes         |
| `0 9,17 * * *`    | At 9:00 AM and 5:00 PM   |
| `0 0 1 * *`       | First day of every month |
| `0 0 * * SUN`     | Every Sunday at midnight |

## 🏗️ Architecture

The application follows the [**Elm Architecture**](https://guide.elm-lang.org/architecture/) (Model-View-Update pattern) via Bubble Tea:

```text
┌───────────────────────────────────────┐
│                 Model                 │
│  (State: 5 text inputs, error, etc.)  │
└────────────┬──────────────────────────┘
             │
             ├──► Update (Handle events)
             │     ├─ Key presses
             │     ├─ Field validation
             │     └─ Navigation
             │
             └──► View (Render UI)
                   ├─ Title
                   ├─ Input fields
                   ├─ Description
                   └─ Help/Error text
```

### Key Components

- **Model**: Holds application state (5 textinput fields, cursor, description, error)
- **Update**: Processes user input and updates state
- **View**: Renders the current state to terminal
- **Validation**: Field-aware validation prevents invalid input

### Field-Aware Validation

The editor implements sophisticated validation rules:

- **Minute/Hour/Day** (fields 0-2): Only numeric values, no letters
- **Month** (field 3): Numbers 1-12 or month abbreviations (JAN-DEC)
- **Weekday** (field 4): Numbers 0-6 or day abbreviations (SUN-SAT)
- **Minimum length**: Letter values must be at least 3 characters
- **Abbreviation validation**: Checks that abbreviations are valid for that field

## 🧪 Testing

The project has comprehensive test coverage (93.6% with 72 test cases) and zero linting issues.

### Run Tests

```bash
# Run all tests
make test
# Or:
go test -v

# Run tests with coverage
make test-coverage
# Or:
go test -v -cover

# Generate detailed coverage report
make test-coverage-view
# Or:
go test -coverprofile=coverage.out
go tool cover -html=coverage.out

# Run benchmarks
make bench
```

### Linting and Formatting

```bash
# Run linter
make lint

# Auto-fix linting issues
make lint-fix

# Format code
make fmt

# Run vet
make vet

# Check everything (format + lint + test)
make check

# Run all checks and build
make all
```

### Test Categories

- ✅ Input validation (numeric, alphabetic, special characters)
- ✅ Navigation (tab, arrows, home, end)
- ✅ Field-specific validation (month/day names)
- ✅ Error handling (invalid expressions)
- ✅ Edge cases (empty fields, bounds, single letters)
- ✅ Clipboard operations
- ✅ Help toggle
- ✅ Window resize
- ✅ Complex cron expressions

## 📦 Dependencies

- [github.com/charmbracelet/bubbletea](https://github.com/charmbracelet/bubbletea) - Terminal UI framework
- [github.com/charmbracelet/bubbles](https://github.com/charmbracelet/bubbles) - Text input components
- [github.com/charmbracelet/lipgloss](https://github.com/charmbracelet/lipgloss) - Terminal styling
- [github.com/lnquy/cron](https://github.com/lnquy/cron) - Cron expression descriptions
- [github.com/robfig/cron/v3](https://github.com/robfig/cron/v3) - Cron expression parsing
- [github.com/atotto/clipboard](https://github.com/atotto/clipboard) - Clipboard integration

## 🔧 Development

### Project Structure

```text
.
├── main.go           # Main application code
├── main_test.go      # Test suite
├── go.mod            # Go module dependencies
├── go.sum            # Dependency checksums
├── LICENSE           # Project license
└── README.md         # This file
```

### Code Quality

- ✅ **Production-Ready**: 9.5/10 code quality score
- ✅ **Well-Tested**: 90%+ test coverage
- ✅ **No Global Pollution**: Minimal global state
- ✅ **Error Handling**: Comprehensive error handling throughout
- ✅ **Documentation**: Clear comments and documentation
- ✅ **Type Safety**: Strong typing with Go

### Coding Standards

- Follow [Effective Go](https://golang.org/doc/effective_go.html) guidelines
- Maintain test coverage above 90%
- Use meaningful variable and function names
- Add comments for complex logic
- Run `go fmt` before committing

## 🐛 Known Limitations

- Requires terminal with color support for best experience
- Clipboard operations require clipboard utilities (xclip on Linux, pbcopy on macOS)
- Advanced cron features (L, W, #) are not fully supported in descriptions

## 🤝 Contributing

Contributions are welcome! Please follow these steps:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'feat: add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a [Pull Request](https://github.com/techquestsdev/crontab-guru/pulls)

### Before Submitting

- Ensure all tests pass (`go test -v`)
- Maintain or improve code coverage
- Follow existing code style
- Add tests for new features
- Update documentation as needed

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## � Publishing & Releases

### For Maintainers

To publish a new release:

```bash
# 1. Create homebrew-tap repository (first time only)
#    On GitHub, create public repo: techquestsdev/homebrew-tap

# 2. Create and push a version tag
git tag -a v1.0.0 -m "Release v1.0.0: Initial stable release"
git push origin v1.0.0

# 3. GitHub Actions automatically handles everything:
#    ✅ Builds multi-platform binaries (Linux, macOS, Windows)
#    ✅ Creates GitHub release with changelog
#    ✅ Publishes Homebrew formula to homebrew-tap
#    ✅ No GITHUB_TOKEN needed (automatically provided)
```

**Full release guide**: See [PUBLISH.md](PUBLISH.md) for step-by-step checklist or [RELEASE.md](RELEASE.md) for complete instructions on:

- First release setup (homebrew-tap repository)
- Testing releases locally with GoReleaser
- Publishing to additional package managers (Snap, Scoop, Docker)
- Local testing (only time you need custom GITHUB_TOKEN)

## �🙏 Acknowledgments

- [crontab.guru](https://crontab.guru/) for the project inspiration
- [Charm](https://charm.sh/) for the amazing Bubble Tea framework
- [robfig](https://github.com/robfig) for the robust cron parser
- [lnquy](https://github.com/lnquy) for the cron description library

## 📧 Contact

André Nogueira - [@aanogueira](https://github.com/aanogueira)

Project Link: [https://github.com/techquestsdev/crontab-guru](https://github.com/techquestsdev/crontab-guru)

---

### Made with ❤️ and Go
