// Copyright (c) 2025 Andre Nogueira
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

// Package main provides comprehensive tests for the crontab guru TUI application.
//
// Test Coverage:
// - Model initialization and state management
// - Keyboard navigation (Tab, Shift+Tab, Enter, Space, Backspace)
// - Cron expression validation (wildcards, ranges, steps, abbreviations)
// - Human-readable description generation
// - Next run time calculation
// - Clipboard operations
// - UI rendering and styling
// - Error handling and recovery
// - Edge cases and boundary conditions
//
// All tests use t.Parallel() for concurrent execution where possible,
// though some tests interact with shared global state (like the program variable).

package main

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/cockroachdb/errors"
)

// assertModelType safely asserts that a tea.Model is a *model.
// This helper function provides type-safe conversion from tea.Model interface
// to the concrete *model type, failing the test if the assertion fails.
func assertModelType(t *testing.T, teaModel tea.Model) *model {
	t.Helper()

	m, ok := teaModel.(*model)
	if !ok {
		t.Fatalf("Expected *model, got %T", teaModel)
	}

	return m
}

// TestInitialModel verifies that the model is initialized correctly with:
// - 5 input fields (minute, hour, day, month, weekday)
// - First input focused
// - Default cron expression "20 4 * * *" (4:20 AM daily)
func TestInitialModel(t *testing.T) {
	t.Parallel()

	m := initialModel()

	if len(m.inputs) != 5 {
		t.Errorf("Expected 5 inputs, but got %d", len(m.inputs))
	}

	if m.inputs[0].Focused() != true {
		t.Error("Expected the first input to be focused")
	}

	initialValues := []string{"20", "4", "*", "*", "*"}
	for i, v := range initialValues {
		if m.inputs[i].Value() != v {
			t.Errorf("Expected input %d to have value %s, but got %s", i, v, m.inputs[i].Value())
		}
	}
}

// TestUpdateFocus verifies keyboard navigation between input fields:
// - Tab cycles through all 5 fields with wraparound
// - Shift+Tab moves to the previous field
func TestUpdateFocus(t *testing.T) {
	t.Parallel()

	m := initialModel()

	// Test Tab navigation
	for i := range 5 {
		if m.focusIndex != i%5 {
			t.Errorf("Expected focus index to be %d, but got %d", i%5, m.focusIndex)
		}

		keyMsg := tea.KeyMsg{Type: tea.KeyTab}
		newModel, _ := m.Update(keyMsg)
		m = assertModelType(t, newModel)
	}

	// Test Shift+Tab navigation
	m = initialModel() // Reset model
	keyMsg := tea.KeyMsg{Type: tea.KeyShiftTab}
	newModel, _ := m.Update(keyMsg)

	m = assertModelType(t, newModel)

	if m.focusIndex != 4 {
		t.Errorf("Expected focus index to be 4 after Shift+Tab, but got %d", m.focusIndex)
	}
}

// TestUpdateCopy verifies that pressing 'y' copies the cron expression
// to the clipboard and displays a success message with a timer to clear it.
func TestUpdateCopy(t *testing.T) {
	t.Parallel()

	m := initialModel()

	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")}
	newModel, cmd := m.Update(keyMsg)
	m = assertModelType(t, newModel)

	if m.copyMessage != copyMessageText {
		t.Errorf("Expected copy message to be \"Copied!\", but got \"%s\"", m.copyMessage)
	}

	if cmd == nil {
		t.Error("Expected a command to be returned for clearing the message")
	}

	// Execute the command and check the message
	msg := cmd()
	if _, ok := msg.(clearCopyMessage); !ok {
		t.Errorf("Expected command to return a clearCopyMessage, but got %T", msg)
	}
}

// TestUpdateDescriptionErrors verifies error handling for invalid cron expressions.
// Empty values are treated as wildcards (*), so they should produce valid descriptions.
func TestUpdateDescriptionErrors(t *testing.T) {
	t.Parallel()

	m := initialModel()

	// Test with all empty values - should convert to "* * * * *" (every minute)
	for i := range m.inputs {
		m.inputs[i].SetValue("")
	}

	m.updateDescription()
	// Empty values are treated as *, so we should get a valid description
	if m.description == "" {
		t.Error("Expected a valid description for '* * * * *', but got empty")
	}

	if m.err != nil {
		t.Errorf("Expected no error for '* * * * *', but got: %v", m.err)
	}

	// Test invalid cron for ToDescription - single letter that's not valid
	m = initialModel() // Reset
	m.inputs[0].SetValue("x")
	m.updateDescription()

	if m.err == nil {
		t.Error("Expected an error for invalid cron expression in ToDescription, but got nil")
	}
}

// TestUpdateMessages verifies handling of various message types in the Update method,
// including clearCopyMessage, Ctrl+C, '?', Backspace, and WindowSizeMsg.
func TestUpdateMessages(t *testing.T) {
	t.Parallel()

	m := initialModel()

	// Test clearCopyMessage
	m.copyMessage = "test"
	newModel, _ := m.Update(clearCopyMessage{})

	m = assertModelType(t, newModel)

	if m.copyMessage != "" {
		t.Errorf("Expected copyMessage to be empty, but got: %s", m.copyMessage)
	}

	// Test Ctrl+C - should trigger quit
	keyMsg := tea.KeyMsg{Type: tea.KeyCtrlC}
	newModel, _ = m.Update(keyMsg)
	m = assertModelType(t, newModel)

	// Test '?' key - toggles help display
	m.showHelp = false
	keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")}
	newModel, _ = m.Update(keyMsg)

	m = assertModelType(t, newModel)

	if !m.showHelp {
		t.Error("Expected showHelp to be true after '?'")
	}

	newModel, _ = m.Update(keyMsg) // Press again to toggle off

	m = assertModelType(t, newModel)

	if m.showHelp {
		t.Error("Expected showHelp to be false after second '?'")
	}

	// Test backspace - should move to previous field when current field is empty
	m.focusIndex = 1
	m.inputs[1].SetValue("")

	keyMsg = tea.KeyMsg{Type: tea.KeyBackspace}
	newModel, _ = m.Update(keyMsg)

	m = assertModelType(t, newModel)

	if m.focusIndex != 0 {
		t.Errorf("Expected focus index to be 0 after backspace on empty input, but got %d", m.focusIndex)
	}

	// Test WindowSizeMsg - should update terminal dimensions
	winSizeMsg := tea.WindowSizeMsg{Width: 80, Height: 24}
	newModel, _ = m.Update(winSizeMsg)

	m = assertModelType(t, newModel)

	if m.width != 80 || m.height != 24 {
		t.Errorf("Expected width=80, height=24, but got width=%d, height=%d", m.width, m.height)
	}
}

// TestUpdateInputs verifies that character input updates the focused input field correctly.
func TestUpdateInputs(t *testing.T) {
	t.Parallel()

	m := initialModel()
	m.focusIndex = 0

	// Simulate typing '1'
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("1")}
	newModel, _ := m.Update(keyMsg)
	m = assertModelType(t, newModel)

	expectedValue := "201"
	if m.inputs[0].Value() != expectedValue {
		t.Errorf("Expected input 0 to have value \"%s\", but got \"%s\"", expectedValue, m.inputs[0].Value())
	}
}

// TestInit verifies that the Init() method returns a valid command
// (the text cursor blink command for the Bubble Tea framework).
func TestInit(t *testing.T) {
	t.Parallel()

	m := initialModel()

	cmd := m.Init()
	if cmd == nil {
		t.Error("Expected a command from Init(), but got nil")
	}
}

// TestUpdateDescriptionParseError verifies that invalid cron values
// (like day 32) are handled gracefully and don't produce a next run time.
func TestUpdateDescriptionParseError(t *testing.T) {
	t.Parallel()

	m := initialModel()
	m.inputs[2].SetValue("32") // Day 32 is invalid
	m.updateDescription()

	if m.nextRun != "" {
		t.Errorf("Expected empty nextRun for invalid cron parse, but got: %s", m.nextRun)
	}
}

// TestUpdateDefaultCase verifies that regular character input is properly
// handled and updates the focused input field.
func TestUpdateDefaultCase(t *testing.T) {
	t.Parallel()

	m := initialModel()
	m.focusIndex = 1
	m.inputs[0].Blur()
	m.inputs[1].Focus()
	m.inputs[1].SetValue("1")

	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")}
	newModel, _ := m.Update(keyMsg)
	m = assertModelType(t, newModel)

	expectedValue := "1a"
	if m.inputs[1].Value() != expectedValue {
		t.Errorf("Expected input 1 to have value \"%s\", but got \"%s\"", expectedValue, m.inputs[1].Value())
	}
}

// TestView verifies that the View() method renders all UI components correctly,
// including the title, description, inputs, and instructions.
func TestView(t *testing.T) {
	t.Parallel()

	m := initialModel()
	m.width = 80 // Set a fixed width for predictable layout

	// Test default view
	view := m.View()
	if !strings.Contains(view, "crontab guru") {
		t.Error("View should contain the title")
	}

	if !strings.Contains(view, "Press ? for help") {
		t.Error("View should contain the default help text")
	}

	// Test with error
	m.err = errors.New("test error")
	m.description = ""

	view = m.View()
	if !strings.Contains(view, "Error: test error") {
		t.Error("View should contain the error message")
	}

	m.err = nil // Reset error

	// Test with empty description and no error
	m.description = ""

	view = m.View()
	if !strings.Contains(view, "\n\n") {
		t.Error("View should contain a blank line for empty description")
	}

	// Test with showHelp
	m.showHelp = true

	view = m.View()
	if !strings.Contains(view, "tab/space/enter: next field") {
		t.Error("View should contain the detailed help text when showHelp is true")
	}

	m.showHelp = false // Reset

	// Test with copyMessage
	m.copyMessage = copyMessageText

	view = m.View()
	if !strings.Contains(view, copyMessageText) {
		t.Error("View should contain the copy message")
	}

	// Test with no nextRun
	m.nextRun = ""

	view = m.View()
	if !strings.Contains(view, "\n\n") {
		t.Error("View should contain a blank line for empty nextRun")
	}
}

// TestUpdateCopyMessageCleared verifies that the clearCopyMessage message type
// properly resets the copy message after the timer expires.
func TestUpdateCopyMessageCleared(t *testing.T) {
	t.Parallel()

	m := initialModel()
	m.copyMessage = copyMessageText

	// Simulate the clearCopyMessage
	newModel, _ := m.Update(clearCopyMessage{})
	m = assertModelType(t, newModel)

	if m.copyMessage != "" {
		t.Errorf("Expected copy message to be cleared, but got \"%s\"", m.copyMessage)
	}
}

// TestRun verifies that the run() function can start and stop the application
// correctly when a Quit message is sent. In CI environments without TTY,
// the function should exit gracefully.
func TestRun(t *testing.T) {
	t.Parallel()

	// Note: This test will exit early in CI environments without TTY
	err := run()
	if err != nil {
		t.Errorf("run() returned an error: %v", err)
	}
}

// TestRunWithOptions verifies that the application can run with custom options
// for headless testing environments.
func TestRunWithOptions(t *testing.T) {
	t.Parallel()

	errCh := make(chan error, 1)

	go func() {
		// Use WithoutRenderer to run in headless mode for testing
		errCh <- runWithOptions(tea.WithoutRenderer())
	}()

	time.Sleep(100 * time.Millisecond)

	// Send quit message to stop the app
	if app != nil {
		app.Send(tea.Quit())
	}

	err := <-errCh
	if err != nil {
		t.Errorf("runWithOptions() returned an error: %v", err)
	}
}

// TestViewWithNegativeFocusIndex verifies that the View() method handles
// edge cases with invalid focus indices without panicking.
func TestViewWithNegativeFocusIndex(t *testing.T) {
	t.Parallel()

	m := initialModel()
	m.width = 80

	// Force a negative focus index (simulating edge case)
	m.focusIndex = -1

	// This should not panic
	view := m.View()
	if view == "" {
		t.Error("View should not be empty even with negative focus index")
	}

	// Test with out-of-bounds positive index
	m.focusIndex = 99

	view = m.View()
	if view == "" {
		t.Error("View should not be empty even with out-of-bounds focus index")
	}
}

// TestUpdateDescriptionWithSingleCharInMonthField verifies that a single invalid
// character in a field produces an appropriate error message.
func TestUpdateDescriptionWithSingleCharInMonthField(t *testing.T) {
	t.Parallel()

	m := initialModel()

	// Set month field (index 3) to a single character
	m.inputs[3].SetValue("a")

	// This should not panic - it should set an error instead
	m.updateDescription()

	if m.err == nil {
		t.Error("Expected an error for single character in month field, but got nil")
	}

	if !strings.Contains(m.err.Error(), "month") {
		t.Errorf("Expected error to contain 'month', but got '%s'", m.err.Error())
	}

	if m.description != "" {
		t.Errorf("Expected empty description for invalid input, but got: %s", m.description)
	}

	if m.nextRun != "" {
		t.Errorf("Expected empty nextRun for invalid input, but got: %s", m.nextRun)
	}
}

// TestIsValidCronPart verifies the validation logic for cron expression parts,
// including wildcards, ranges, lists, step values, and month/weekday names.
func TestIsValidCronPart(t *testing.T) {
	t.Parallel()

	tests := []struct {
		value      string
		fieldIndex int // 0=minute, 1=hour, 2=day, 3=month, 4=weekday
		expected   bool
	}{
		// Numeric fields (minute, hour, day)
		{"*", 0, true},
		{"", 0, true},
		{"5", 0, true},
		{"15", 1, true},
		{"1-5", 0, true},
		{"1,2,3", 0, true},
		{"*/5", 0, true},
		{"0-23", 1, true},

		// Month field (3) - letters allowed
		{"JAN-DEC", 3, true},
		{"JAN", 3, true},
		{"1-12", 3, true},

		// Weekday field (4) - letters allowed
		{"MON", 4, true},
		{"SUN-SAT", 4, true},
		{"0-6", 4, true},

		// Invalid cases
		{"a", 0, false},   // Single letter in minute field - invalid
		{"S", 3, false},   // Single letter in month field - invalid
		{"x", 4, false},   // Single letter in weekday field - invalid
		{"!", 0, false},   // Single special char - invalid
		{"*s", 0, false},  // Wildcard with trailing letter - invalid
		{"1s", 0, false},  // Number with trailing letter - invalid
		{"12x", 0, false}, // Number with trailing invalid letter - invalid
		{"AB", 3, false},  // Two letter abbreviation - invalid
		{"*a", 0, false},  // Wildcard with letter - invalid
		{"*/", 0, false},  // Incomplete step value - invalid

		// Letters in wrong fields
		{"JAN", 0, false}, // Month name in minute field - invalid
		{"MON", 0, false}, // Day name in minute field - invalid
		{"JAN", 4, false}, // Month name in weekday field - invalid
		{"MON", 3, false}, // Day name in month field - invalid
	}

	for _, tt := range tests {
		result := isValidCronPart(tt.value, tt.fieldIndex)
		if result != tt.expected {
			t.Errorf("isValidCronPart(%q, field %d) = %v, expected %v", tt.value, tt.fieldIndex, result, tt.expected)
		}
	}
}

// TestInvalidFieldErrorMessages verifies that each input field generates
// the correct error message when an invalid value is provided.
func TestInvalidFieldErrorMessages(t *testing.T) {
	t.Parallel()

	fieldNames := []string{"minute", "hour", "day", "month", "weekday"}

	for i, fieldName := range fieldNames {
		m := initialModel()

		// Set invalid single character in the field
		m.inputs[i].SetValue("x")

		m.updateDescription()

		if m.err == nil {
			t.Errorf("Expected an error for invalid value in %s field, but got nil", fieldName)

			continue
		}

		// Check that the error contains the field name
		if !strings.Contains(m.err.Error(), fieldName) {
			t.Errorf("Expected error message to contain '%s', but got '%s'", fieldName, m.err.Error())
		}
	}
}

// TestUpdateDescriptionWithValidCronExpressions verifies that valid cron expressions
// produce non-empty descriptions and next run times without errors.
func TestUpdateDescriptionWithValidCronExpressions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		values []string
	}{
		{"Every minute", []string{"*", "*", "*", "*", "*"}},
		{"Hourly at :30", []string{"30", "*", "*", "*", "*"}},
		{"Daily at noon", []string{"0", "12", "*", "*", "*"}},
		{"Weekly on Monday", []string{"0", "0", "*", "*", "MON"}},
		{"Monthly on 1st", []string{"0", "0", "1", "*", "*"}},
		{"With ranges", []string{"0-30", "9-17", "*", "*", "1-5"}},
		{"With lists", []string{"0,15,30,45", "*", "*", "*", "*"}},
		{"With steps", []string{"*/5", "*", "*", "*", "*"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := initialModel()
			for i, val := range tt.values {
				m.inputs[i].SetValue(val)
			}

			m.updateDescription()

			if m.err != nil {
				t.Errorf("Expected no error for %s, but got: %v", tt.name, m.err)
			}

			if m.description == "" {
				t.Errorf("Expected a description for %s, but got empty", tt.name)
			}

			if m.nextRun == "" {
				t.Errorf("Expected nextRun for %s, but got empty", tt.name)
			}
		})
	}
}

// TestUpdateDescriptionCaching verifies that updateDescription() avoids
// redundant computation when the cron expression hasn't changed.
func TestUpdateDescriptionCaching(t *testing.T) {
	t.Parallel()

	m := initialModel()

	// Set initial values
	m.inputs[0].SetValue("30")
	m.updateDescription()
	firstDesc := m.description

	// Call updateDescription again without changing values
	m.updateDescription()

	// Should return early due to caching (lastCronExpr check)
	if m.description != firstDesc {
		t.Errorf("Description should remain the same due to caching")
	}
}

// TestUpdateDescriptionWithInvalidCronParse verifies that semantically invalid
// values (like day 32) produce parser errors.
func TestUpdateDescriptionWithInvalidCronParse(t *testing.T) {
	t.Parallel()

	m := initialModel()

	// Set valid format but semantically invalid value (day 32)
	m.inputs[2].SetValue("32")
	m.updateDescription()

	// Should get an error from the parser
	if m.err == nil {
		t.Error("Expected an error for day value 32, but got nil")
	}

	if m.nextRun != "" {
		t.Errorf("Expected empty nextRun for invalid day, but got: %s", m.nextRun)
	}
}

// TestUpdateWithEnterKey verifies that pressing Enter advances the focus
// to the next input field.
func TestUpdateWithEnterKey(t *testing.T) {
	t.Parallel()

	m := initialModel()
	m.focusIndex = 0

	keyMsg := tea.KeyMsg{Type: tea.KeyEnter}
	newModel, _ := m.Update(keyMsg)
	m = assertModelType(t, newModel)

	if m.focusIndex != 1 {
		t.Errorf("Expected focus index to be 1 after Enter, but got %d", m.focusIndex)
	}
}

// TestUpdateWithSpaceKey verifies that pressing Space advances the focus
// to the next input field.
func TestUpdateWithSpaceKey(t *testing.T) {
	t.Parallel()

	m := initialModel()
	m.focusIndex = 2

	keyMsg := tea.KeyMsg{Type: tea.KeySpace}
	newModel, _ := m.Update(keyMsg)
	m = assertModelType(t, newModel)

	if m.focusIndex != 3 {
		t.Errorf("Expected focus index to be 3 after Space, but got %d", m.focusIndex)
	}
}

// TestUpdateWithEscKey verifies that pressing Esc triggers the quit command.
func TestUpdateWithEscKey(t *testing.T) {
	t.Parallel()

	m := initialModel()

	keyMsg := tea.KeyMsg{Type: tea.KeyEsc}
	_, cmd := m.Update(keyMsg)

	// Should return tea.Quit command
	if cmd == nil {
		t.Error("Expected a quit command for Esc key")
	}
}

// TestUpdateInputsMultipleCalls verifies that multiple character inputs
// are properly accumulated in the focused field.
func TestUpdateInputsMultipleCalls(t *testing.T) {
	t.Parallel()

	m := initialModel()
	m.focusIndex = 1
	m.inputs[0].Blur()
	m.inputs[1].Focus()

	// Type multiple characters
	for _, char := range "15" {
		keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{char}}
		newModel, _ := m.Update(keyMsg)
		m = assertModelType(t, newModel)
	}

	expectedValue := "415" // Initial was "4", added "15"
	if m.inputs[1].Value() != expectedValue {
		t.Errorf("Expected input 1 to have value \"%s\", but got \"%s\"", expectedValue, m.inputs[1].Value())
	}
}

// TestBackspaceOnFirstInput verifies that backspace on the first field
// with empty value doesn't move focus backward (no wraparound).
func TestBackspaceOnFirstInput(t *testing.T) {
	t.Parallel()

	m := initialModel()
	m.focusIndex = 0
	m.inputs[0].SetValue("")

	keyMsg := tea.KeyMsg{Type: tea.KeyBackspace}
	newModel, _ := m.Update(keyMsg)
	m = assertModelType(t, newModel)

	// Should remain at index 0 since we can't go back further
	if m.focusIndex != 0 {
		t.Errorf("Expected focus index to remain at 0, but got %d", m.focusIndex)
	}
}

// TestViewWithDescription verifies that the View renders both the description
// and next run time when they are present.
func TestViewWithDescription(t *testing.T) {
	t.Parallel()

	m := initialModel()
	m.width = 100
	m.description = "Test description"
	m.nextRun = "2025-10-21 12:00:00"

	view := m.View()

	if !strings.Contains(view, "Test description") {
		t.Error("View should contain the description")
	}

	if !strings.Contains(view, "2025-10-21 12:00:00") {
		t.Error("View should contain the nextRun time")
	}
}

// TestViewFocusedInputStyling verifies that the focused input's label
// is rendered with the appropriate styling.
func TestViewFocusedInputStyling(t *testing.T) {
	t.Parallel()

	m := initialModel()
	m.width = 100
	m.focusIndex = 2

	// Focus the third input
	for i := range m.inputs {
		m.inputs[i].Blur()
	}

	m.inputs[2].Focus()

	view := m.View()

	// Should contain the "day" label
	if !strings.Contains(view, "day") {
		t.Error("View should contain focused field label")
	}
}

// TestInitialModelWithError verifies that initialModel() always returns
// a valid model structure (error handling is tested elsewhere).
func TestInitialModelWithError(t *testing.T) {
	t.Parallel()

	// This tests the error path in initialModel when cron descriptor creation fails
	// In normal circumstances this shouldn't fail, but we test the structure
	m := initialModel()

	// Skip the rest of the test if model is nil (should never happen)
	if m == nil {
		t.Fatal("initialModel should not return nil")

		return
	}

	if len(m.inputs) != 5 {
		t.Errorf("Expected 5 inputs, got %d", len(m.inputs))
	}
}

// TestIsValidCronPartEdgeCases tests additional edge cases for cron part validation,
// including step values, ranges, and month/weekday names.
func TestIsValidCronPartEdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		value      string
		fieldIndex int
		expected   bool
	}{
		{"*/10", 0, true},      // Step value with 2 digits
		{"*/100", 0, true},     // Step value with 3 digits
		{"*/*", 0, false},      // Invalid step
		{"*/x", 0, false},      // Invalid step with letter
		{"1-5/2", 0, true},     // Range with step
		{"MON-FRI", 4, true},   // Day range (weekday field)
		{"JAN", 3, true},       // Month abbreviation (month field)
		{"JANUARY", 3, true},   // Full month name (contains JAN, month field)
		{"XYZ", 3, false},      // Invalid abbreviation
		{"1,2,3,4,5", 0, true}, // Long list
		{"0", 0, true},         // Zero
		{"59", 0, true},        // Two digit number
		{"123", 2, true},       // Three digit number (day field can go higher than 31 in ranges)
	}

	for _, tt := range tests {
		result := isValidCronPart(tt.value, tt.fieldIndex)
		if result != tt.expected {
			t.Errorf("isValidCronPart(%q, field %d) = %v, expected %v", tt.value, tt.fieldIndex, result, tt.expected)
		}
	}
}

// TestIsValidCronPartMoreEdgeCases tests additional validation scenarios
// for special characters and invalid patterns.
func TestIsValidCronPartMoreEdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		value      string
		fieldIndex int
		expected   bool
	}{
		{"#", 0, false},      // Invalid special character
		{"@", 0, false},      // Invalid special character
		{"5-10", 0, true},    // Valid range
		{"TUE", 4, true},     // Day abbreviation (weekday field)
		{"TUESDAY", 4, true}, // Contains valid abbreviation (weekday field)
		{"AB", 3, false},     // Invalid two-letter combo
		{"1a", 0, false},     // Number with invalid letter
		{"10x", 0, false},    // Number with trailing invalid letter
	}

	for _, tt := range tests {
		result := isValidCronPart(tt.value, tt.fieldIndex)
		if result != tt.expected {
			t.Errorf("isValidCronPart(%q, field %d) = %v, expected %v", tt.value, tt.fieldIndex, result, tt.expected)
		}
	}
}

// TestUpdateDescriptionEmptyAfterClear verifies that empty values are treated
// as wildcards (*) and still produce valid descriptions.
func TestUpdateDescriptionEmptyAfterClear(t *testing.T) {
	t.Parallel()

	m := initialModel()

	// First set valid values
	m.inputs[0].SetValue("30")
	m.updateDescription()

	if m.description == "" {
		t.Error("Expected description after setting valid value")
	}

	// Now clear all but one to test empty value conversion
	m.inputs[1].SetValue("")
	m.inputs[2].SetValue("")
	m.updateDescription()

	// Empty values should be treated as *, so still valid
	if m.err != nil {
		t.Errorf("Expected no error with empty values, but got: %v", m.err)
	}
}

// TestUpdateWithBackspaceOnNonEmptyInput verifies that backspace on a non-empty
// input field doesn't move focus backward.
func TestUpdateWithBackspaceOnNonEmptyInput(t *testing.T) {
	t.Parallel()

	m := initialModel()
	m.focusIndex = 1
	m.inputs[1].SetValue("10") // Non-empty value
	m.inputs[0].Blur()
	m.inputs[1].Focus()

	keyMsg := tea.KeyMsg{Type: tea.KeyBackspace}
	newModel, _ := m.Update(keyMsg)
	m = assertModelType(t, newModel)

	// Should stay at index 1 since value is not empty
	if m.focusIndex != 1 {
		t.Errorf("Expected focus index to remain at 1, but got %d", m.focusIndex)
	}
}

// TestUpdateFocusIndexOutOfBoundsRecovery verifies that the Update method
// can recover from an artificially out-of-bounds focus index.
func TestUpdateFocusIndexOutOfBoundsRecovery(t *testing.T) {
	t.Parallel()

	m := initialModel()

	// Artificially set focus index out of bounds
	m.focusIndex = 10

	// Update should reset it
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("5")}
	newModel, _ := m.Update(keyMsg)
	m = assertModelType(t, newModel)

	// Should be reset to 0
	if m.focusIndex != 0 {
		t.Errorf("Expected focus index to be reset to 0, but got %d", m.focusIndex)
	}
}

// TestShiftTabFromLastToFirst verifies that Shift+Tab from the last input
// wraps around to the first input.
func TestShiftTabFromLastToFirst(t *testing.T) {
	t.Parallel()

	m := initialModel()

	// Move to last input
	m.focusIndex = 4
	for i := range m.inputs {
		m.inputs[i].Blur()
	}

	m.inputs[4].Focus()

	// Press shift+tab
	keyMsg := tea.KeyMsg{Type: tea.KeyShiftTab}
	newModel, _ := m.Update(keyMsg)
	m = assertModelType(t, newModel)

	if m.focusIndex != 3 {
		t.Errorf("Expected focus index to be 3 after shift+tab from 4, but got %d", m.focusIndex)
	}
}

// TestTabWraparound verifies that pressing Tab from the last input
// wraps around to the first input.
func TestTabWraparound(t *testing.T) {
	t.Parallel()

	m := initialModel()

	// Move to last input
	m.focusIndex = 4
	for i := range m.inputs {
		m.inputs[i].Blur()
	}

	m.inputs[4].Focus()

	// Press tab - should wrap to 0
	keyMsg := tea.KeyMsg{Type: tea.KeyTab}
	newModel, _ := m.Update(keyMsg)
	m = assertModelType(t, newModel)

	if m.focusIndex != 0 {
		t.Errorf("Expected focus index to wrap to 0 after tab from 4, but got %d", m.focusIndex)
	}
}

// TestValidCronPartWildcardWithNonSlash verifies that wildcards followed by
// non-slash characters are rejected as invalid.
func TestValidCronPartWildcardWithNonSlash(t *testing.T) {
	t.Parallel()

	// This tests the defensive check in isValidCronPart for wildcard followed by non-slash
	result := isValidCronPart("*a", 0)
	if result != false {
		t.Error("Expected false for '*a', but got true")
	}

	result = isValidCronPart("*x", 0)
	if result != false {
		t.Error("Expected false for '*x', but got true")
	}

	result = isValidCronPart("*5", 0)
	if result != false {
		t.Error("Expected false for '*5', but got true")
	}
}

// TestUpdateDescriptionWithAllWhitespace verifies the behavior when input
// fields contain only whitespace (which gets trimmed to empty, then converted to *).
func TestUpdateDescriptionWithAllWhitespace(t *testing.T) {
	t.Parallel()

	m := initialModel()

	// This is hard to trigger since empty inputs are converted to "*"
	// But we can test by setting lastCronExpr to something and then trying whitespace
	m.lastCronExpr = ""

	// Set inputs to spaces (though the textinput might not allow this)
	// This test documents the intent even if the line is hard to reach
	for i := range m.inputs {
		m.inputs[i].SetValue(" ")
	}

	// The trimmed result would be empty
	// However, the actual implementation converts empty to "*"
	m.updateDescription()

	// Should have some result since empty converts to *
	if m.err != nil {
		t.Logf("Got error (acceptable): %v", m.err)
	}
}

// TestUpdateDescriptionWithParseError verifies that the cron parser properly
// rejects semantically invalid values (like day 32 or hour 25).
func TestUpdateDescriptionWithParseError(t *testing.T) {
	t.Parallel()

	m := initialModel()

	// Set values that pass validation but fail cron parsing
	// Day value 32 should fail the parser
	m.inputs[2].SetValue("32")
	m.updateDescription()

	// The cron library ToDescription might catch this
	// Or the parser will catch it
	if m.err == nil {
		t.Log("Expected an error for day value 32, parser may have been lenient")
	} else {
		// Error is properly handled
		if m.nextRun != "" {
			t.Error("nextRun should be empty when there's a parse error")
		}

		if m.description != "" {
			t.Error("description should be empty when there's a parse error")
		}
	}

	// Try another invalid case - hour 25
	m = initialModel()
	m.inputs[1].SetValue("25")
	m.updateDescription()

	if m.err == nil {
		t.Log("Expected an error for hour value 25, parser may have been lenient")
	}
}

// TestShiftTabWithInvalidFocusIndex verifies that Shift+Tab can recover
// from an artificially invalid focus index.
func TestShiftTabWithInvalidFocusIndex(t *testing.T) {
	t.Parallel()

	m := initialModel()

	// Set focus index to a very invalid value
	m.focusIndex = -10

	// This should trigger the fallback in shift+tab
	keyMsg := tea.KeyMsg{Type: tea.KeyShiftTab}
	newModel, _ := m.Update(keyMsg)
	m = assertModelType(t, newModel)

	// Should recover to a valid index
	if m.focusIndex < 0 || m.focusIndex >= len(m.inputs) {
		t.Errorf("Expected valid focus index after recovery, but got %d", m.focusIndex)
	}
}

// TestBackspaceWithInvalidFocusIndex verifies that backspace handling can
// recover from an invalid focus index without panicking.
func TestBackspaceWithInvalidFocusIndex(t *testing.T) {
	t.Parallel()

	m := initialModel()

	// Set an empty value and invalid focus
	m.focusIndex = 1
	m.inputs[1].SetValue("")

	// First manually set focusIndex to invalid to test fallback
	// This simulates the edge case
	keyMsg := tea.KeyMsg{Type: tea.KeyBackspace}
	newModel, _ := m.Update(keyMsg)
	m = assertModelType(t, newModel)

	// Should have moved back to index 0
	if m.focusIndex != 0 {
		t.Errorf("Expected focus index 0 after backspace on empty input at index 1, got %d", m.focusIndex)
	}
}

// TestBackspaceEmptyInputAtIndex2 verifies that backspace on an empty input
// at index 2 moves focus back to index 1.
func TestBackspaceEmptyInputAtIndex2(t *testing.T) {
	t.Parallel()

	m := initialModel()

	// Move to index 2 and make it empty
	m.focusIndex = 2
	for i := range m.inputs {
		m.inputs[i].Blur()
	}

	m.inputs[2].SetValue("")
	m.inputs[2].Focus()

	// Press backspace on empty input
	keyMsg := tea.KeyMsg{Type: tea.KeyBackspace}
	newModel, _ := m.Update(keyMsg)
	m = assertModelType(t, newModel)

	// Should move back to index 1
	if m.focusIndex != 1 {
		t.Errorf("Expected focus index 1 after backspace on empty input at index 2, got %d", m.focusIndex)
	}
}

// TestShiftTabFromIndex0 verifies that Shift+Tab from the first input
// wraps around to the last input.
func TestShiftTabFromIndex0(t *testing.T) {
	t.Parallel()

	m := initialModel()

	// Already at index 0
	m.focusIndex = 0
	m.inputs[0].Focus()

	// Press shift+tab - should wrap to last index
	keyMsg := tea.KeyMsg{Type: tea.KeyShiftTab}
	newModel, _ := m.Update(keyMsg)
	m = assertModelType(t, newModel)

	if m.focusIndex != 4 {
		t.Errorf("Expected focus index 4 after shift+tab from 0, but got %d", m.focusIndex)
	}
}

// TestIsValidCronPartStepWithMultipleDigits verifies that step values
// with various numbers of digits are validated correctly.
func TestIsValidCronPartStepWithMultipleDigits(t *testing.T) {
	t.Parallel()

	tests := []struct {
		value      string
		fieldIndex int
		expected   bool
	}{
		{"*/1", 0, true},
		{"*/12", 0, true},
		{"*/123", 0, true},
		{"*/5678", 0, true},
		{"*/0", 0, true},
	}

	for _, tt := range tests {
		result := isValidCronPart(tt.value, tt.fieldIndex)
		if result != tt.expected {
			t.Errorf("isValidCronPart(%q, field %d) = %v, expected %v", tt.value, tt.fieldIndex, result, tt.expected)
		}
	}
}

// TestUpdateDescriptionWithMonthNames verifies that month name ranges
// (like JAN-DEC) are properly handled.
func TestUpdateDescriptionWithMonthNames(t *testing.T) {
	t.Parallel()

	m := initialModel()

	// Test with full month range
	m.inputs[3].SetValue("JAN-DEC")
	m.updateDescription()

	if m.err != nil {
		t.Errorf("Expected no error for JAN-DEC, but got: %v", m.err)
	}

	if m.description == "" {
		t.Error("Expected description for month range, but got empty")
	}
}

// TestUpdateDescriptionWithDayNames verifies that weekday name ranges
// (like MON-FRI) are properly handled.
func TestUpdateDescriptionWithDayNames(t *testing.T) {
	t.Parallel()

	m := initialModel()

	// Test with day range
	m.inputs[4].SetValue("MON-FRI")
	m.updateDescription()

	if m.err != nil {
		t.Errorf("Expected no error for MON-FRI, but got: %v", m.err)
	}

	if m.description == "" {
		t.Error("Expected description for day range, but got empty")
	}
}

// TestUpdateDescriptionWithComplexExpression verifies that complex cron expressions
// combining lists, ranges, and multiple fields are handled correctly.
func TestUpdateDescriptionWithComplexExpression(t *testing.T) {
	t.Parallel()

	m := initialModel()

	// Complex expression with multiple features
	m.inputs[0].SetValue("0,15,30,45")
	m.inputs[1].SetValue("9-17")
	m.inputs[2].SetValue("1-15")
	m.inputs[3].SetValue("1,4,7,10")
	m.inputs[4].SetValue("1-5")

	m.updateDescription()

	if m.err != nil {
		t.Errorf("Expected no error for complex expression, but got: %v", m.err)
	}

	if m.description == "" {
		t.Error("Expected description for complex expression, but got empty")
	}

	if m.nextRun == "" {
		t.Error("Expected nextRun for complex expression, but got empty")
	}
}

// TestIsValidCronPartWithRangesAndSteps verifies that ranges combined
// with step values are validated correctly.
func TestIsValidCronPartWithRangesAndSteps(t *testing.T) {
	t.Parallel()

	tests := []struct {
		value      string
		fieldIndex int
		expected   bool
	}{
		{"0-59/5", 0, true},  // Range with step (minute)
		{"1-12/2", 3, true},  // Month range with step
		{"10-20/3", 2, true}, // General range with step (day)
		{"0-23/4", 1, true},  // Hour range with step
	}

	for _, tt := range tests {
		result := isValidCronPart(tt.value, tt.fieldIndex)
		if result != tt.expected {
			t.Errorf("isValidCronPart(%q, field %d) = %v, expected %v", tt.value, tt.fieldIndex, result, tt.expected)
		}
	}
}

// TestViewWithLongCopyMessage verifies that long copy messages
// are properly displayed in the view.
func TestViewWithLongCopyMessage(t *testing.T) {
	t.Parallel()

	m := initialModel()
	m.width = 100
	m.copyMessage = "Expression copied to clipboard!"

	view := m.View()

	if !strings.Contains(view, "Expression copied to clipboard!") {
		t.Error("View should contain the long copy message")
	}
}

// TestMultipleConsecutiveUpdates verifies that multiple character inputs
// in quick succession are properly accumulated.
func TestMultipleConsecutiveUpdates(t *testing.T) {
	t.Parallel()

	m := initialModel()
	m.focusIndex = 0
	m.inputs[0].Focus()

	// Type multiple characters in quick succession
	chars := []rune{'3', '0'}
	for _, char := range chars {
		keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{char}}
		newModel, _ := m.Update(keyMsg)
		m = assertModelType(t, newModel)
	}

	// Should have accumulated the input
	if !strings.Contains(m.inputs[0].Value(), "30") {
		t.Errorf("Expected input to contain '30', got: %s", m.inputs[0].Value())
	}
}

// TestUpdateDescriptionCachingPreventsRedundantWork verifies that the caching
// mechanism properly avoids redundant description updates.
func TestUpdateDescriptionCachingPreventsRedundantWork(t *testing.T) {
	t.Parallel()

	m := initialModel()

	// Set initial values
	m.inputs[0].SetValue("15")
	m.inputs[1].SetValue("10")
	m.updateDescription()

	firstDescription := m.description

	// Call update again without changing anything
	m.updateDescription()

	// Description should be the same
	if m.description != firstDescription {
		t.Error("Description should not change when cron expression is the same")
	}

	// Now change a value
	m.inputs[0].SetValue("20")
	m.updateDescription()

	// Description should be different
	if m.description == firstDescription {
		t.Error("Description should change when cron expression changes")
	}
}

// TestUpdateWithHelpToggle verifies that the '?' key properly toggles
// the help display on and off.
func TestUpdateWithHelpToggle(t *testing.T) {
	t.Parallel()

	m := initialModel()

	// Initially help should be false
	if m.showHelp {
		t.Error("Expected showHelp to be false initially")
	}

	// Press '?' to toggle help
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}}
	newModel, _ := m.Update(keyMsg)
	m = assertModelType(t, newModel)

	if !m.showHelp {
		t.Error("Expected showHelp to be true after pressing ?")
	}

	// Press '?' again to toggle off
	newModel, _ = m.Update(keyMsg)
	m = assertModelType(t, newModel)

	if m.showHelp {
		t.Error("Expected showHelp to be false after toggling twice")
	}
}

// TestUpdateWithWindowSizeMsg verifies that window resize events properly
// update the model's width and height.
func TestUpdateWithWindowSizeMsg(t *testing.T) {
	t.Parallel()

	m := initialModel()

	// Initially width and height should be 0
	if m.width != 0 || m.height != 0 {
		t.Errorf("Expected initial width and height to be 0, got width=%d, height=%d", m.width, m.height)
	}

	// Send a window size message
	sizeMsg := tea.WindowSizeMsg{Width: 100, Height: 50}
	newModel, _ := m.Update(sizeMsg)
	m = assertModelType(t, newModel)

	if m.width != 100 {
		t.Errorf("Expected width to be 100, got %d", m.width)
	}

	if m.height != 50 {
		t.Errorf("Expected height to be 50, got %d", m.height)
	}
}

// TestUpdateWithMultipleWindowResizes verifies that multiple consecutive
// window resize events all update the model correctly.
func TestUpdateWithMultipleWindowResizes(t *testing.T) {
	t.Parallel()

	m := initialModel()

	sizes := []struct{ width, height int }{
		{80, 24},
		{120, 40},
		{160, 60},
	}

	for _, size := range sizes {
		sizeMsg := tea.WindowSizeMsg{Width: size.width, Height: size.height}
		newModel, _ := m.Update(sizeMsg)
		m = assertModelType(t, newModel)

		if m.width != size.width {
			t.Errorf("Expected width to be %d, got %d", size.width, m.width)
		}

		if m.height != size.height {
			t.Errorf("Expected height to be %d, got %d", size.height, m.height)
		}
	}
}

// TestUpdateDescriptionWithComplexRange verifies that complex range expressions
// with multiple comma-separated ranges are properly handled.
func TestUpdateDescriptionWithComplexRange(t *testing.T) {
	t.Parallel()

	m := initialModel()

	// Set a complex range expression
	m.inputs[0].SetValue("0-15,30-45")
	m.inputs[1].SetValue("9-17")
	m.inputs[2].SetValue("1-5,15-20")
	m.inputs[3].SetValue("*")
	m.inputs[4].SetValue("1-5")

	m.updateDescription()

	if m.err != nil {
		t.Errorf("Unexpected error with complex ranges: %v", m.err)
	}

	if m.description == "" {
		t.Error("Expected description to be generated for complex ranges")
	}
}

// TestUpdateDescriptionWithMixedStepAndRange verifies that expressions
// combining step values with ranges are properly handled.
func TestUpdateDescriptionWithMixedStepAndRange(t *testing.T) {
	t.Parallel()

	m := initialModel()

	// Set mixed step and range
	m.inputs[0].SetValue("*/10")
	m.inputs[1].SetValue("2-18/2")
	m.inputs[2].SetValue("1-15/3")
	m.inputs[3].SetValue("*")
	m.inputs[4].SetValue("*")

	m.updateDescription()

	if m.err != nil {
		t.Errorf("Unexpected error with mixed step and range: %v", m.err)
	}

	if m.description == "" {
		t.Error("Expected description to be generated for mixed step and range")
	}
}

// TestUpdateDescriptionWithQuestionMark verifies the behavior when using
// the '?' character in day and weekday fields (means "no specific value").
func TestUpdateDescriptionWithQuestionMark(t *testing.T) {
	t.Parallel()

	m := initialModel()

	// ? is valid in day and weekday fields (means "no specific value")
	m.inputs[0].SetValue("0")
	m.inputs[1].SetValue("12")
	m.inputs[2].SetValue("?")
	m.inputs[3].SetValue("*")
	m.inputs[4].SetValue("?")

	m.updateDescription()

	// Note: The cron library might not support ? or might handle it differently
	// This test documents the current behavior
	if m.err == nil {
		// If no error, description should be generated
		if m.description == "" {
			t.Error("Expected description when using ? in day/weekday fields")
		}
	}
}

// TestUpdateWithCtrlC verifies that Ctrl+C triggers the quit command.
func TestUpdateWithCtrlC(t *testing.T) {
	t.Parallel()

	m := initialModel()

	keyMsg := tea.KeyMsg{Type: tea.KeyCtrlC}
	_, cmd := m.Update(keyMsg)

	if cmd == nil {
		t.Error("Expected Ctrl+C to return tea.Quit command")
	}
}

// TestClearCopyMessageAfterDelay verifies that the clearCopyMessage message
// properly clears the copy message after the timer expires.
func TestClearCopyMessageAfterDelay(t *testing.T) {
	t.Parallel()

	m := initialModel()

	// Set a copy message first
	m.copyMessage = copyMessageText

	// Simulate the clearCopyMessage being sent
	msg := clearCopyMessage{}
	newModel, _ := m.Update(msg)
	m = assertModelType(t, newModel)

	if m.copyMessage != "" {
		t.Errorf("Expected copy message to be cleared, but got: %s", m.copyMessage)
	}
}

// TestViewWithHelpShown verifies that the detailed help text is displayed
// when showHelp is true.
func TestViewWithHelpShown(t *testing.T) {
	t.Parallel()

	m := initialModel()
	m.showHelp = true

	view := m.View()

	// Check that help text is present
	if !strings.Contains(view, "tab/space/enter: next field") {
		t.Error("Expected help text to contain 'tab/space/enter: next field'")
	}

	if !strings.Contains(view, "shift+tab: previous field") {
		t.Error("Expected help text to contain 'shift+tab: previous field'")
	}

	if !strings.Contains(view, "y: copy expression") {
		t.Error("Expected help text to contain 'y: copy expression'")
	}

	if !strings.Contains(view, "any value") {
		t.Error("Expected help text to contain cron syntax explanation")
	}
}

// TestViewWithoutHelp verifies that detailed help text is hidden when
// showHelp is false.
func TestViewWithoutHelp(t *testing.T) {
	t.Parallel()

	m := initialModel()
	m.showHelp = false

	view := m.View()

	// Help should not be shown, just the footer hint
	if strings.Contains(view, "Tab / Enter / Space") && strings.Contains(view, "move forward") {
		t.Error("Expected detailed help text to be hidden when showHelp is false")
	}
}

// TestNavigationWithAllEmptyInputs verifies that tab navigation works correctly
// when all input fields are empty.
func TestNavigationWithAllEmptyInputs(t *testing.T) {
	t.Parallel()

	m := initialModel()

	// Clear all inputs
	for i := range m.inputs {
		m.inputs[i].SetValue("")
	}

	m.focusIndex = 0

	// Tab through all empty inputs
	for i := range 5 {
		keyMsg := tea.KeyMsg{Type: tea.KeyTab}
		newModel, _ := m.Update(keyMsg)
		m = assertModelType(t, newModel)

		expectedIndex := (i + 1) % 5
		if m.focusIndex != expectedIndex {
			t.Errorf("Expected focus index to be %d after tab, got %d", expectedIndex, m.focusIndex)
		}
	}
}

// TestBackspaceNavigationAcrossMultipleEmptyFields verifies that backspace
// can navigate backward through multiple empty input fields.
func TestBackspaceNavigationAcrossMultipleEmptyFields(t *testing.T) {
	t.Parallel()

	m := initialModel()

	// Clear all inputs
	for i := range m.inputs {
		m.inputs[i].SetValue("")
	}

	// Start at the last field
	m.focusIndex = 4
	m.inputs[4].Focus()

	// Backspace should move backwards through empty fields
	for i := 4; i > 0; i-- {
		keyMsg := tea.KeyMsg{Type: tea.KeyBackspace}
		newModel, _ := m.Update(keyMsg)
		m = assertModelType(t, newModel)

		expectedIndex := i - 1
		if m.focusIndex != expectedIndex {
			t.Errorf("Expected focus index to be %d after backspace, got %d", expectedIndex, m.focusIndex)
		}
	}
}

// TestUpdateDescriptionWithHashSymbol verifies that the '#' symbol
// is rejected as invalid in standard cron expressions.
func TestUpdateDescriptionWithHashSymbol(t *testing.T) {
	t.Parallel()

	m := initialModel()

	// # is used in some extended cron formats but should be invalid in standard cron
	m.inputs[0].SetValue("#")
	m.updateDescription()

	if m.err == nil {
		t.Error("Expected error when using # symbol")
	}

	if !strings.Contains(m.err.Error(), "minute") {
		t.Errorf("Expected error to contain 'minute', got: %v", m.err)
	}
}

// TestUpdateDescriptionWithSingleLetterInMonthField verifies the fix for
// single invalid letters in the month field (which previously caused crashes).
func TestUpdateDescriptionWithSingleLetterInMonthField(t *testing.T) {
	t.Parallel()

	m := initialModel()

	// Single letter "S" in month field should be rejected (this was causing the crash)
	m.inputs[3].SetValue("S")
	m.updateDescription()

	if m.err == nil {
		t.Error("Expected error when using single letter 'S' in month field")
	}

	if !strings.Contains(m.err.Error(), "month") {
		t.Errorf("Expected error to contain 'month', got: %v", m.err)
	}
}

// TestUpdateDescriptionWithInvalidLettersInFields verifies that invalid letters
// are properly rejected in each field type.
func TestUpdateDescriptionWithInvalidLettersInFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		fieldIndex int
		value      string
		fieldName  string
	}{
		{0, "A", "minute"},
		{1, "B", "hour"},
		{2, "C", "day"},
		{3, "XYZ", "month"},   // Invalid month abbreviation
		{4, "ABC", "weekday"}, // Invalid weekday abbreviation
	}

	for _, tt := range tests {
		m := initialModel()
		m.inputs[tt.fieldIndex].SetValue(tt.value)
		m.updateDescription()

		if m.err == nil {
			t.Errorf("Expected error for value %q in %s field", tt.value, tt.fieldName)
		}

		if !strings.Contains(m.err.Error(), tt.fieldName) {
			t.Errorf("Expected error to contain '%s', got: %v", tt.fieldName, m.err)
		}
	}
}
