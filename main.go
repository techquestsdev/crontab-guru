// Copyright (c) 2025 Andre Nogueira
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

// Package main implements a terminal-based cron expression editor.
// It provides an interactive UI for creating and editing crontab schedule expressions
// with real-time validation and human-readable descriptions.

package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/cockroachdb/errors"
	crondesc "github.com/lnquy/cron"
	"github.com/mattn/go-isatty"
	cronparser "github.com/robfig/cron/v3"
)

const (
	inputCharLimit     = 10               // Maximum characters per input field
	inputWidth         = 5                // Visual width of each input field
	initialCron        = "20 4 * * *"     // Default cron expression (4:20 AM daily)
	numCronFields      = 5                // Number of cron fields: minute, hour, day, month, weekday
	minAbbrevLength    = 3                // Minimum length for month/day abbreviations (e.g., "JAN", "MON")
	fieldIndexMonth    = 3                // Index of the month field in the cron expression
	fieldIndexWeekday  = 4                // Index of the weekday field in the cron expression
	stepValueMinLength = 2                // Minimum length for step values (e.g., "*/5" has "/" at index 1)
	labelWidth         = 12               // Width for field labels in the UI
	copyMessageText    = "Copied!"        // Success message when copying to clipboard
	copyFailedText     = "Failed to copy" // Error message when clipboard copy fails
	cronParserOptions  = cronparser.Minute | cronparser.Hour | cronparser.Dom | cronparser.Month | cronparser.Dow
)

//nolint:gochecknoglobals
var (
	// ErrInvalidValue is returned when a cron field contains an invalid value
	ErrInvalidValue = errors.New("invalid value in field")
	// ErrCronDescriptor is returned when the cron descriptor fails to initialize
	ErrCronDescriptor = errors.New("failed to create cron descriptor")
	// ErrCronParse is returned when the cron expression fails to parse
	ErrCronParse = errors.New("failed to parse cron expression")
)

// clipboardAvailable checks if clipboard operations are available in the current environment
func clipboardAvailable() bool {
	// On Linux, clipboard requires DISPLAY environment variable and clipboard utilities (xclip/xsel)
	if runtime.GOOS == "linux" {
		if os.Getenv("DISPLAY") == "" {
			return false
		}
	}

	// Check if we're running in a non-TTY environment (like CI)
	if !isatty.IsTerminal(os.Stdout.Fd()) {
		return false
	}

	return true
}

//nolint:gochecknoglobals
var (
	// Cron field names used for error messages and UI labels
	fieldNames = []string{"minute", "hour", "day", "month", "weekday"}

	// UI color palette
	colorYellow    = lipgloss.Color("#FFFF00") // Highlighted/focused elements
	colorWhite     = lipgloss.Color("#FFFFFF") // Primary text
	colorGray      = lipgloss.Color("#888888") // Help text and secondary info
	colorLightGray = lipgloss.Color("#AAAAAA") // Labels and subtle text
	colorRed       = lipgloss.Color("#FF0000") // Errors and invalid input
	colorCyan      = lipgloss.Color("#00FFFF") // Info messages (next run time)
)

//nolint:gochecknoglobals
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorYellow).
			MarginTop(1).
			MarginBottom(1)

	descriptionStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorWhite).
				MarginBottom(1).
				Italic(true)

	inputBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(0, 1)

	focusedInputBoxStyle = inputBoxStyle.
				BorderForeground(colorYellow)

	errorInputBoxStyle = inputBoxStyle.
				BorderForeground(colorRed)

	helpStyle = lipgloss.NewStyle().
			Foreground(colorGray).
			MarginTop(1)

	labelStyle = lipgloss.NewStyle().
			Foreground(colorLightGray)

	focusedLabelStyle = lipgloss.NewStyle().
				Foreground(colorYellow).
				Bold(true)

	infoStyle = lipgloss.NewStyle().
			Foreground(colorCyan)
)

// clearCopyMessage is sent after a delay to hide the clipboard copy message
type clearCopyMessage struct{}

// model represents the application state for the Bubble Tea TUI
type model struct {
	inputs       []textinput.Model             // Input fields for the 5 cron parts
	description  string                        // Human-readable description of the cron expression
	nextRun      string                        // Next scheduled execution time
	err          error                         // Current validation or parsing error
	width        int                           // Terminal width
	height       int                           // Terminal height
	cronDesc     crondesc.ExpressionDescriptor // Cron expression descriptor
	focusIndex   int                           // Index of currently focused input field
	copyMessage  string                        // Message shown after copying to clipboard
	showHelp     bool                          // Whether help text is visible
	lastCronExpr string                        // Last processed cron expression (for caching)
}

// initialModel creates and initializes a new model with default values
func initialModel() *model {
	m := model{
		inputs:     make([]textinput.Model, numCronFields),
		focusIndex: 0,
		showHelp:   false,
	}

	placeholders := []string{"*", "*", "*", "*", "*"}
	initialValues := strings.Fields(initialCron)

	for i := range numCronFields {
		t := textinput.New()

		t.Placeholder = placeholders[i]
		if i < len(initialValues) {
			t.SetValue(initialValues[i])
		}

		t.CharLimit = inputCharLimit
		t.Width = inputWidth
		m.inputs[i] = t
	}

	m.inputs[0].Focus()

	cronDescriptor, err := crondesc.NewDescriptor()
	if err != nil {
		m.err = fmt.Errorf("%w: %w", ErrCronDescriptor, err)

		return &m
	}

	m.cronDesc = *cronDescriptor

	m.updateDescription()

	return &m
}

// validateMonthAbbreviation checks if a letter part contains valid month abbreviations
func validateMonthAbbreviation(letterPart string) bool {
	validMonths := []string{
		"JAN", "FEB", "MAR", "APR", "MAY", "JUN",
		"JUL", "AUG", "SEP", "OCT", "NOV", "DEC",
	}
	for _, month := range validMonths {
		if strings.Contains(letterPart, month) || strings.HasPrefix(letterPart, month) {
			return true
		}
	}

	return false
}

// validateWeekdayAbbreviation checks if a letter part contains valid weekday abbreviations
func validateWeekdayAbbreviation(letterPart string) bool {
	validDays := []string{"SUN", "MON", "TUE", "WED", "THU", "FRI", "SAT"}
	for _, day := range validDays {
		if strings.Contains(letterPart, day) || strings.HasPrefix(letterPart, day) {
			return true
		}
	}

	return false
}

// extractLetterPart extracts the leading letter sequence from a value
func extractLetterPart(value string) string {
	upper := strings.ToUpper(value)

	var letterPart strings.Builder

	for _, char := range upper {
		if char >= 'A' && char <= 'Z' {
			letterPart.WriteRune(char)
		} else {
			break
		}
	}

	return letterPart.String()
}

// hasLetters checks if a string contains any letter characters
func hasLetters(value string) bool {
	for _, char := range value {
		if (char >= 'A' && char <= 'Z') || (char >= 'a' && char <= 'z') {
			return true
		}
	}

	return false
}

// validateStepValue validates wildcard step values like */5
func validateStepValue(value string) bool {
	if !strings.HasPrefix(value, "*") || len(value) <= 1 {
		return true
	}

	// Only valid if followed by / for step values
	if value[1] != '/' {
		return false
	}

	// If it's "*/" it must have a number after the slash
	if len(value) == stepValueMinLength {
		return false
	}

	// Check that everything after */ is numeric
	for index := stepValueMinLength; index < len(value); index++ {
		if value[index] < '0' || value[index] > '9' {
			return false
		}
	}

	return true
}

// isValidCharForField checks if all characters in value are valid for the field
func isValidCharForField(value string, fieldIndex int) bool {
	validChars := "0123456789*,-/"
	if fieldIndex == fieldIndexMonth || fieldIndex == fieldIndexWeekday {
		validChars += "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	}

	for _, char := range value {
		if !strings.ContainsRune(validChars, char) {
			return false
		}
	}

	return true
}

// validateLetterValue validates letter-containing values for month/weekday fields
func validateLetterValue(value string, fieldIndex int) bool {
	if fieldIndex != fieldIndexMonth && fieldIndex != fieldIndexWeekday {
		return false
	}

	letterPart := extractLetterPart(value)
	if len(letterPart) < minAbbrevLength {
		return false
	}

	if fieldIndex == fieldIndexMonth {
		return validateMonthAbbreviation(letterPart)
	}

	return validateWeekdayAbbreviation(letterPart)
}

// isValidCronPart validates a cron field value based on its field index.
// It checks for valid characters, letter abbreviations (for month/weekday),
// and proper step value syntax (e.g., "*/5").
//
// fieldIndex mapping: 0=minute, 1=hour, 2=day, 3=month, 4=weekday
//
// Returns true if the value is valid for the specified field, false otherwise.
func isValidCronPart(value string, fieldIndex int) bool {
	if value == "" || value == "*" {
		return true
	}

	if !isValidCharForField(value, fieldIndex) {
		return false
	}

	if hasLetters(value) {
		return validateLetterValue(value, fieldIndex)
	}

	return validateStepValue(value)
}

// Init initializes the model and returns the initial command (text cursor blink)
func (m *model) Init() tea.Cmd {
	return textinput.Blink
}

// View renders the complete UI by assembling all visual components
func (m *model) View() string {
	var builder strings.Builder

	builder.WriteString(m.renderHeader())
	builder.WriteString(m.renderDescription())
	builder.WriteString(m.renderNextRun())
	builder.WriteString(m.renderInputs())
	builder.WriteString(m.renderLabels())
	builder.WriteString(m.renderAllowedValues())
	builder.WriteString(m.renderHelp())
	builder.WriteString(m.renderFooter())

	return builder.String()
}

// Update handles all messages (keyboard input, window resize, timer events)
// and updates the model state accordingly
func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Ensure m.focusIndex is always within valid bounds
	if m.focusIndex < 0 || m.focusIndex >= len(m.inputs) {
		m.focusIndex = 0
	}

	switch msg := msg.(type) {
	case clearCopyMessage:
		m.copyMessage = ""

		return m, nil

	case tea.KeyMsg:
		if model, cmd := m.handleKeyMessage(msg); model != nil {
			return model, cmd
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		return m, nil
	}

	cmd := m.updateInputs(msg)
	m.updateDescription()

	return m, cmd
}

// handleKeyMessage processes keyboard input
func (m *model) handleKeyMessage(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "esc":
		return m, tea.Quit
	case "y":
		return m, m.handleCopyToClipboard()
	case "?":
		m.showHelp = !m.showHelp

		return m, nil
	case "tab", " ", "enter":
		return m, m.handleTabNavigation()
	case "shift+tab":
		return m, m.handleShiftTabNavigation()
	case "backspace":
		if cmd := m.handleBackspaceNavigation(); cmd != nil {
			return m, cmd
		}
	}

	return nil, nil
}

// updateDescription validates the cron expression and updates the human-readable
// description and next run time. Uses caching to avoid redundant processing.
func (m *model) updateDescription() {
	cronExpr := m.buildCronExpression()

	// Optimization: Only update if cron expression has changed
	if cronExpr == m.lastCronExpr {
		return
	}

	m.lastCronExpr = cronExpr

	if strings.TrimSpace(cronExpr) == "" {
		m.clearDescription()

		return
	}

	// Validate all parts before attempting to parse
	if err := m.validateCronParts(); err != nil {
		m.err = err
		m.description = ""
		m.nextRun = ""

		return
	}

	m.updateCronDescription(cronExpr)
	m.updateNextRunTime(cronExpr)
}

// buildCronExpression constructs the cron expression string from input fields
func (m *model) buildCronExpression() string {
	cronParts := make([]string, 0, numCronFields)

	for _, input := range m.inputs {
		value := input.Value()
		if value == "" {
			value = "*"
		}

		cronParts = append(cronParts, value)
	}

	return strings.Join(cronParts, " ")
}

// clearDescription resets the description, next run time, and error
func (m *model) clearDescription() {
	m.description = ""
	m.nextRun = ""
	m.err = nil
}

// validateCronParts validates all cron field values
func (m *model) validateCronParts() error {
	for index, input := range m.inputs {
		if !isValidCronPart(input.Value(), index) {
			return fmt.Errorf("%w: %s", ErrInvalidValue, fieldNames[index])
		}
	}

	return nil
}

// updateCronDescription generates the human-readable description
func (m *model) updateCronDescription(cronExpr string) {
	desc, err := m.cronDesc.ToDescription(cronExpr, crondesc.Locale_en)
	if err != nil {
		m.err = err
		m.description = ""
		m.nextRun = ""

		return
	}

	m.description = desc
	m.err = nil
}

// updateNextRunTime calculates the next scheduled execution time
func (m *model) updateNextRunTime(cronExpr string) {
	parser := cronparser.NewParser(cronParserOptions)

	schedule, err := parser.Parse(cronExpr)
	if err != nil {
		m.nextRun = ""
		m.err = fmt.Errorf("%w: %w", ErrCronParse, err)

		return
	}

	next := schedule.Next(time.Now())
	m.nextRun = next.Format("2006-01-02 15:04:05")
}

// updateInputs updates the focused input field
func (m *model) updateInputs(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd

	for index := range m.inputs {
		if m.inputs[index].Focused() {
			m.inputs[index], cmd = m.inputs[index].Update(msg)
		}
	}

	return cmd
}

// handleCopyToClipboard handles copying the cron expression to clipboard
func (m *model) handleCopyToClipboard() tea.Cmd {
	cronParts := make([]string, 0, numCronFields)
	for _, input := range m.inputs {
		cronParts = append(cronParts, input.Value())
	}

	cronExpr := strings.Join(cronParts, " ")

	// Check if clipboard is available in the current environment
	if !clipboardAvailable() {
		m.copyMessage = "Clipboard not available"
	} else if err := clipboard.WriteAll(cronExpr); err != nil {
		m.copyMessage = copyFailedText
	} else {
		m.copyMessage = copyMessageText
	}

	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return clearCopyMessage{}
	})
}

// handleTabNavigation handles tab key navigation between fields
func (m *model) handleTabNavigation() tea.Cmd {
	m.inputs[m.focusIndex].Blur()
	m.focusIndex = (m.focusIndex + 1) % len(m.inputs)
	m.inputs[m.focusIndex].Focus()

	return textinput.Blink
}

// handleShiftTabNavigation moves focus to the previous input field (with wraparound)
func (m *model) handleShiftTabNavigation() tea.Cmd {
	m.inputs[m.focusIndex].Blur()

	m.focusIndex--
	if m.focusIndex < 0 {
		m.focusIndex = len(m.inputs) - 1
	}
	// Ensure m.focusIndex is valid before calling Focus()
	if m.focusIndex >= 0 && m.focusIndex < len(m.inputs) {
		m.inputs[m.focusIndex].Focus()
	} else {
		m.inputs[0].Focus()
		m.focusIndex = 0
	}

	return textinput.Blink
}

// handleBackspaceNavigation moves to the previous field when backspace is pressed
// on an empty input field (convenience feature for editing flow)
func (m *model) handleBackspaceNavigation() tea.Cmd {
	if m.inputs[m.focusIndex].Value() == "" && m.focusIndex > 0 {
		m.inputs[m.focusIndex].Blur()

		m.focusIndex--
		if m.focusIndex >= 0 && m.focusIndex < len(m.inputs) {
			m.inputs[m.focusIndex].Focus()
		} else {
			m.inputs[0].Focus()
			m.focusIndex = 0
		}

		return textinput.Blink
	}

	return nil
}

// renderHeader renders the title and subtitle
func (m *model) renderHeader() string {
	var builder strings.Builder

	title := titleStyle.Render("crontab guru")
	builder.WriteString(lipgloss.Place(m.width, 0, lipgloss.Center, lipgloss.Top, title))
	builder.WriteString("\n")

	subtitle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#AAAAAA")).
		Render("The quick and simple editor for cron schedule expressions")
	builder.WriteString(lipgloss.Place(m.width, 0, lipgloss.Center, lipgloss.Top, subtitle))
	builder.WriteString("\n\n")

	return builder.String()
}

// renderDescription renders the description or error message
func (m *model) renderDescription() string {
	switch {
	case m.description != "":
		desc := descriptionStyle.Render(fmt.Sprintf("\"%s\"", m.description))

		return lipgloss.Place(m.width, 0, lipgloss.Center, lipgloss.Top, desc) + "\n"
	case m.err != nil:
		errmsg := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).Bold(true).Render("Error: " + m.err.Error())

		return lipgloss.Place(m.width, 0, lipgloss.Center, lipgloss.Top, errmsg) + "\n"
	default:
		return "\n"
	}
}

// renderNextRun displays the next scheduled execution time if available
func (m *model) renderNextRun() string {
	if m.nextRun != "" {
		nextInfo := infoStyle.Render("next at " + m.nextRun)

		return lipgloss.Place(m.width, 0, lipgloss.Center, lipgloss.Top, nextInfo) + "\n\n"
	}

	return "\n\n"
}

// renderInputs renders the five input fields with appropriate styling
// based on focus state and validation errors
func (m *model) renderInputs() string {
	inputViews := make([]string, 0, len(m.inputs))

	for index := range m.inputs {
		var style lipgloss.Style

		switch {
		case m.err != nil:
			style = errorInputBoxStyle
		case m.inputs[index].Focused():
			style = focusedInputBoxStyle
		default:
			style = inputBoxStyle
		}

		inputViews = append(inputViews, style.Render(m.inputs[index].View()))
	}

	inputs := lipgloss.JoinHorizontal(lipgloss.Top, inputViews...)

	return lipgloss.Place(m.width, 0, lipgloss.Center, lipgloss.Top, inputs) + "\n"
}

// renderLabels renders the field labels
func (m *model) renderLabels() string {
	styledLabels := make([]string, 0, len(fieldNames))
	baseLabelStyle := lipgloss.NewStyle().Width(labelWidth).Align(lipgloss.Center)

	safeFocusIndex := m.focusIndex
	if safeFocusIndex < 0 || safeFocusIndex >= len(m.inputs) {
		safeFocusIndex = 0
	}

	for index, label := range fieldNames {
		var style lipgloss.Style
		if index == safeFocusIndex {
			style = focusedLabelStyle
		} else {
			style = labelStyle
		}

		styledLabels = append(styledLabels, baseLabelStyle.Render(style.Render(label)))
	}

	labelRow := lipgloss.JoinHorizontal(lipgloss.Top, styledLabels...)

	return lipgloss.Place(m.width, 0, lipgloss.Center, lipgloss.Top, labelRow) + "\n"
}

// renderAllowedValues shows the valid value range for the currently focused field
func (m *model) renderAllowedValues() string {
	availableValues := []string{
		"Allowed values: 0-59",
		"Allowed values: 0-23",
		"Allowed values: 1-31",
		"Allowed values: 1-12 or JAN-DEC",
		"Allowed values: 0-6 or SUN-SAT (7 is also Sunday)",
	}

	if m.focusIndex >= 0 && m.focusIndex < len(availableValues) && m.focusIndex < len(m.inputs) {
		availVals := lipgloss.NewStyle().Foreground(lipgloss.Color("#666666")).Render(availableValues[m.focusIndex])

		return lipgloss.Place(m.width, 0, lipgloss.Center, lipgloss.Top, availVals) + "\n\n"
	}

	return "\n\n"
}

// renderHelp displays the help panel with cron syntax and keyboard shortcuts
func (m *model) renderHelp() string {
	if !m.showHelp {
		return ""
	}

	helpText := []string{
		"*    any value",
		",    value list separator",
		"-    range of values",
		"/    step values",
		"---------------------------",
		"tab/space/enter: next field",
		"shift+tab: previous field",
		"y: copy expression",
		"esc/ctrl+c: quit",
	}

	help := helpStyle.Render(strings.Join(helpText, "\n"))

	return lipgloss.Place(m.width, 0, lipgloss.Center, lipgloss.Top, help) + "\n\n"
}

// renderFooter renders the instructions and copy message
func (m *model) renderFooter() string {
	var builder strings.Builder

	instructions := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666666")).
		Render("Press ? for help, y to copy, Esc to quit")
	builder.WriteString(lipgloss.Place(m.width, 0, lipgloss.Center, lipgloss.Top, instructions))
	builder.WriteString("\n")

	if m.copyMessage != "" {
		copyMsg := lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00")).Render(m.copyMessage)
		builder.WriteString(lipgloss.Place(m.width, 0, lipgloss.Center, lipgloss.Top, copyMsg))
	} else {
		builder.WriteString(lipgloss.Place(m.width, 0, lipgloss.Center, lipgloss.Top, ""))
	}

	return builder.String()
}

// app is a package-level variable to allow tests to send quit messages
//
//nolint:gochecknoglobals
var app *tea.Program

// run initializes and runs the Bubble Tea app
func run() error {
	if !isatty.IsTerminal(os.Stdout.Fd()) {
		return nil // Exit gracefully when no TTY is available
	}

	m := initialModel()

	app = tea.NewProgram(m)
	if _, err := app.Run(); err != nil {
		return fmt.Errorf("app execution failed: %w", err)
	}

	return nil
}

// runWithOptions initializes and runs the Bubble Tea app with custom options
// This function is useful for testing as it allows running without a TTY
func runWithOptions(opts ...tea.ProgramOption) error {
	// Check if we have a TTY available, exit gracefully if not
	if !isatty.IsTerminal(os.Stdout.Fd()) {
		return nil
	}

	m := initialModel()

	app = tea.NewProgram(m, opts...)
	if _, err := app.Run(); err != nil {
		return fmt.Errorf("app execution failed: %w", err)
	}

	return nil
}

// main is the entry point of the application
func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
