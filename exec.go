package exec

import (
	"os/exec"
	"strings"
	"syscall"
	"unicode"

	"github.com/sbreitf1/errors"
)

const (
	eol = '\000'
	esc = '\\'
	sqt = '\''
	dqt = '"'
)

var (
	// ErrRun occurs when a command could not be executed.
	ErrRun = errors.New("Could not execute command")
	// ErrReturnCode occurs when a command was executed but returned with a non-zero exit code.
	ErrReturnCode = errors.New("Process returned with code %d")
	// ErrParse occurs when a malformed command line was encountered.
	ErrParse = errors.New("Unable to parse command line")
	// DefaultExecutor denotes the Executor that is used by default for Run and RunLine commands.
	DefaultExecutor Executor
)

func init() {
	DefaultExecutor = NewLocalExecutor()
}

// Executor represents the interface for shell command execution.
type Executor interface {
	// RunLine executes an escaped single string command line.
	RunLine(commandLine string) (string, int, errors.Error)
	// Run executes a command line with separated arguments.
	Run(command string, args ...string) (string, int, errors.Error)
}

// LocalExecutor is used to execute commands on the local shell.
type LocalExecutor struct {
}

// RunLine executes an escaped single string command line.
func (e *LocalExecutor) RunLine(commandLine string) (string, int, errors.Error) {
	return runLine(commandLine)
}

// Run executes a command line with separated arguments.
func (e *LocalExecutor) Run(command string, args ...string) (string, int, errors.Error) {
	return run(command, args...)
}

// NewLocalExecutor returns an executor for the local shell.
func NewLocalExecutor() *LocalExecutor {
	return &LocalExecutor{}
}

// MockExecutor offers functionality to mock and debug executed commands.
type MockExecutor struct {
	RunCallback func(command string, args ...string) (string, int, errors.Error)
}

// RunLine parses the command and calls RunCallback.
func (e *MockExecutor) RunLine(commandLine string) (string, int, errors.Error) {
	command, args, err := Parse(commandLine)
	if err != nil {
		return "", 0, err
	}

	return e.RunCallback(command, args...)
}

// Run calls runCallback.
func (e *MockExecutor) Run(command string, args ...string) (string, int, errors.Error) {
	return e.RunCallback(command, args...)
}

// NewMockExecutor returns an executor for the local shell.
func NewMockExecutor(runCallback func(command string, args ...string) (string, int, errors.Error)) *MockExecutor {
	return &MockExecutor{runCallback}
}

// ShouldRunLine executes the given command using RunLine but returns an error for non-zero return codes.
func ShouldRunLine(commandLine string) (string, errors.Error) {
	result, code, err := RunLine(commandLine)
	if err != nil {
		return "", err
	}
	if code != 0 {
		return result, ErrReturnCode.Args(code).Make()
	}
	return result, nil
}

// RunLine parses the given command line and runs it using the DefaultExecutor.
func RunLine(commandLine string) (string, int, errors.Error) {
	return DefaultExecutor.RunLine(commandLine)
}

func runLine(commandLine string) (string, int, errors.Error) {
	command, args, err := Parse(commandLine)
	if err != nil {
		return "", 0, err
	}

	return run(command, args...)
}

// ShouldRun executes the given command using Run but returns an error for non-zero return codes.
func ShouldRun(command string, args ...string) (string, errors.Error) {
	result, code, err := Run(command, args...)
	if err != nil {
		return "", err
	}
	if code != 0 {
		return result, ErrReturnCode.Args(code).Make()
	}
	return result, nil
}

// Run executes a command with given arguments using the DefaultExecutor.
func Run(command string, args ...string) (string, int, errors.Error) {
	return DefaultExecutor.Run(command, args...)
}

func run(command string, args ...string) (string, int, errors.Error) {
	cmd := exec.Command(command, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		switch e := err.(type) {
		case *exec.ExitError:
			switch s := e.Sys().(type) {
			case syscall.WaitStatus:
				return string(output), s.ExitStatus(), nil
			}
		}
		return string(output), 0, ErrRun.Make().Cause(err)
	}

	return string(output), 0, nil
}

const (
	parseDefault = iota
	parseSingleQuote
	parseDoubleQuote
)

// Parse returns the command and arguments from a command line.
func Parse(commandLine string) (string, []string, errors.Error) {
	parts, err := split(commandLine)
	if err != nil {
		return "", nil, err
	}
	if len(parts) == 0 {
		return "", nil, ErrParse.Make().Msg("Unexpected end of command line")
	} else if len(parts) == 1 {
		return parts[0], nil, nil
	} else {
		return parts[0], parts[1:], nil
	}
}

func split(str string) ([]string, errors.Error) {
	// array that holds all seen string parts
	parts := make([]string, 0)

	// parser state to handle quotes and escape sequences
	state := parseDefault
	escape := false
	// the builder assembles the currently processed string part
	var sb strings.Builder
	partStart := 0

	// append EOL (end of line) to command line string for easier processing
	runes := []rune(str + string(eol))
	for i, r := range runes {
		if r == eol {
			if i < (len(runes) - 1) {
				// EOL is ONLY allowed as last char
				return nil, ErrParse.Make().Msg("Invalid 0 char in command line")
			} else if state != parseDefault || escape {
				// last char (EOL) reached but still in quote?
				return nil, ErrParse.Make().Msg("Unexpected end of command line")
			}
		}

		switch state {
		case parseDefault:
			if escape {
				escape = false
				sb.WriteRune(r)
			} else {
				// space runes in default context (not quoted) end the current part
				if unicode.IsSpace(r) || r == eol {
					// ignore multiple consecutive spaces
					if partStart < (i - 1) {
						// append to parts and begin new one
						parts = append(parts, sb.String())
						sb.Reset()
					}
					// keep track of part begin to detect large spaces inbetween arguments
					partStart = i
				} else if r == sqt {
					// do not end current part -> quotes can be combined
					state = parseSingleQuote
				} else if r == dqt {
					// do not end current part -> quotes can be combined
					state = parseDoubleQuote
				} else if r == esc {
					escape = true
				} else {
					sb.WriteRune(r)
				}
			}

		case parseSingleQuote:
			if r == sqt {
				state = parseDefault
			} else {
				sb.WriteRune(r)
			}

		case parseDoubleQuote:
			if escape {
				escape = false
				if r != esc && r != dqt {
					sb.WriteRune(esc)
				}
				sb.WriteRune(r)
			} else {
				if r == dqt {
					state = parseDefault
				} else if r == esc {
					escape = true
				} else {
					sb.WriteRune(r)
				}
			}
		}
	}

	return parts, nil
}

// GetCommandLine is the inverse function of Parse. It assembles a single command line that is equivalent to the given command and arguments by escaping and quoting.
func GetCommandLine(command string, args ...string) string {
	var sb strings.Builder
	sb.WriteString(Quote(command))
	for _, arg := range args {
		sb.WriteRune(' ')
		sb.WriteString(Quote(arg))
	}
	return sb.String()
}

// Quote returns a safe representation of the given string for command line calls.
func Quote(str string) string {
	if len(str) == 0 {
		return `""`
	}

	raw := quoteRaw(str)
	single := quoteSingle(str)
	double := quoteDouble(str)
	if len(raw) < len(double) {
		if len(single) < len(raw) {
			return single
		}
		return raw
	}
	if len(single) < len(double) {
		return single
	}
	return double
}

func quoteRaw(str string) string {
	var sb strings.Builder
	for _, r := range []rune(str) {
		if unicode.IsSpace(r) || r == sqt || r == dqt || r == esc {
			sb.WriteRune(esc)
		}
		sb.WriteRune(r)
	}
	return sb.String()
}

func quoteSingle(str string) string {
	var sb strings.Builder
	sb.WriteRune(sqt)
	for _, r := range []rune(str) {
		if r == sqt {
			// no escaping possible in single quotes: switch to raw
			sb.WriteRune(sqt)
			sb.WriteRune(esc)
			sb.WriteRune(sqt)
			sb.WriteRune(sqt)
		} else {
			sb.WriteRune(r)
		}
	}
	sb.WriteRune(sqt)
	return sb.String()
}

func quoteDouble(str string) string {
	var sb strings.Builder
	sb.WriteRune(dqt)
	for _, r := range []rune(str) {
		if r == dqt || r == esc {
			sb.WriteRune(esc)
		}
		sb.WriteRune(r)
	}
	sb.WriteRune(dqt)
	return sb.String()
}
