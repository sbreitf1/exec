package exec

import (
	"strings"
	"testing"

	"github.com/sbreitf1/errors"
	"github.com/stretchr/testify/assert"
)

/* ############################################# */
/* ###                  Run                  ### */
/* ############################################# */

func TestRunSuccess(t *testing.T) {
	out, code, err := Run(path("success.sh"))
	assert.NoError(t, err)
	assert.Equal(t, 0, code)
	assert.True(t, strings.Contains(out, "some test output here"))
}

func TestRunSuccessArgs(t *testing.T) {
	out, code, err := Run(path("args.sh"), "foo test space", "bar")
	assert.NoError(t, err)
	assert.Equal(t, 0, code)
	assert.True(t, strings.Contains(out, "1foo test space"))
	assert.True(t, strings.Contains(out, "2bar"))
}

func TestRunFail(t *testing.T) {
	out, code, err := Run(path("fail.sh"))
	assert.NoError(t, err)
	assert.Equal(t, 1, code)
	assert.True(t, strings.Contains(out, "error output"))
}

func TestRunError(t *testing.T) {
	_, _, err := Run(path("noexec.txt"))
	assert.True(t, errors.InstanceOf(err, RunError))
}

/* ############################################# */
/* ###                Parser                 ### */
/* ############################################# */

func TestParse(t *testing.T) {
	cmd, args, err := Parse(`newcommand -d foo -m bar`)
	assert.NoError(t, err)
	assert.Equal(t, "newcommand", cmd)
	assert.Equal(t, []string{"-d", "foo", "-m", "bar"}, args)
}

func TestParseCommandOnly(t *testing.T) {
	cmd, args, err := Parse(`newcommand`)
	assert.NoError(t, err)
	assert.Equal(t, "newcommand", cmd)
	assert.Nil(t, args)
}

func TestParseSpaces(t *testing.T) {
	cmd, args, err := Parse(`newcommand    -d    foo    -m   bar    `)
	assert.NoError(t, err)
	assert.Equal(t, "newcommand", cmd)
	assert.Equal(t, []string{"-d", "foo", "-m", "bar"}, args)
}

func TestParseEscapeSpace(t *testing.T) {
	cmd, args, err := Parse(`new\ command -d f\ oo -m bar\ `)
	assert.NoError(t, err)
	assert.Equal(t, "new command", cmd)
	assert.Equal(t, []string{"-d", "f oo", "-m", "bar "}, args)
}

func TestParseEscapeSingleSpace(t *testing.T) {
	cmd, args, err := Parse(`newcommand \ \\\ `)
	assert.NoError(t, err)
	assert.Equal(t, "newcommand", cmd)
	assert.Equal(t, []string{" \\ "}, args)
}

func TestParseEscapeQuotes(t *testing.T) {
	cmd, args, err := Parse(`newcommand \" test \'`)
	assert.NoError(t, err)
	assert.Equal(t, "newcommand", cmd)
	assert.Equal(t, []string{"\"", "test", "'"}, args)
}

func TestParseQuotes(t *testing.T) {
	cmd, args, err := Parse(`newcommand -d "foo ' bar" -m 'single test " quotes'`)
	assert.NoError(t, err)
	assert.Equal(t, "newcommand", cmd)
	assert.Equal(t, []string{"-d", "foo ' bar", "-m", "single test \" quotes"}, args)
}

func TestParseEmptyQuotes(t *testing.T) {
	cmd, args, err := Parse(`newcommand -d "" -m ''`)
	assert.NoError(t, err)
	assert.Equal(t, "newcommand", cmd)
	assert.Equal(t, []string{"-d", "", "-m", ""}, args)
}

func TestParseCombinedQuotes(t *testing.T) {
	cmd, args, err := Parse(`newcommand -d "asdf"'qwert'foo'bar'\ "test""1234"\ `)
	assert.NoError(t, err)
	assert.Equal(t, "newcommand", cmd)
	assert.Equal(t, []string{"-d", "asdfqwertfoobar test1234 "}, args)
}

func TestParseQuoteEscape(t *testing.T) {
	cmd, args, err := Parse(`newcommand a\\b a\;b "a\;b" 'a\;b' "\'\n\"blub" '\"\n\ blub' "foo\\bar\ \t\n\0" 'foo\\bar'`)
	assert.NoError(t, err)
	assert.Equal(t, "newcommand", cmd)
	assert.Equal(t, []string{"a\\b", "a;b", "a\\;b", "a\\;b", "\\'\\n\"blub", "\\\"\\n\\ blub", "foo\\bar\\ \\t\\n\\0", "foo\\\\bar"}, args)
}

func TestParseEmpty(t *testing.T) {
	_, _, err := Parse(``)
	assert.True(t, errors.InstanceOf(err, ParseError))
}

func TestParseEscapeEnd(t *testing.T) {
	_, _, err := Parse(`newcommand \`)
	assert.True(t, errors.InstanceOf(err, ParseError))
}

func TestParseDoubleQuoteFail(t *testing.T) {
	_, _, err := Parse(`newcommand "test`)
	assert.True(t, errors.InstanceOf(err, ParseError))
}

func TestParseSingleQuoteFail(t *testing.T) {
	_, _, err := Parse(`newcommand 'test`)
	assert.True(t, errors.InstanceOf(err, ParseError))
}

func TestParseZeroRune(t *testing.T) {
	_, _, err := Parse("newcommand test \000")
	assert.True(t, errors.InstanceOf(err, ParseError))
}

/* ############################################# */
/* ###                RunLine                ### */
/* ############################################# */

func TestRunLineSuccess(t *testing.T) {
	out, code, err := RunLine(Quote(path("success.sh")))
	assert.NoError(t, err)
	assert.Equal(t, 0, code)
	assert.True(t, strings.Contains(out, "some test output here"))
}

func TestRunLineSuccessArgs(t *testing.T) {
	out, code, err := RunLine(Quote(path("args.sh")) + ` "foo test space" bar`)
	assert.NoError(t, err)
	assert.Equal(t, 0, code)
	assert.True(t, strings.Contains(out, "1foo test space"))
	assert.True(t, strings.Contains(out, "2bar"))
}

func TestRunLineFail(t *testing.T) {
	out, code, err := RunLine(Quote(path("fail.sh")))
	assert.NoError(t, err)
	assert.Equal(t, 1, code)
	assert.True(t, strings.Contains(out, "error output"))
}

func TestRunLineError(t *testing.T) {
	_, _, err := RunLine(Quote(path("noexec.txt")))
	assert.NotNil(t, err)
}

func TestRunLineParseError(t *testing.T) {
	_, _, err := RunLine(Quote(path("args.sh")) + ` "foo test space" "bar`)
	assert.True(t, errors.InstanceOf(err, ParseError))
}

/* ############################################# */
/* ###            GetCommandLine             ### */
/* ############################################# */

func TestGetCommandLine(t *testing.T) {
	cmdLine := GetCommandLine("newcommand", "blub", "", "foo bar", `"test  `, `blub''\`, `"""`)
	assert.Equal(t, `newcommand blub "" foo\ bar '"test  ' "blub''\\" '"""'`, cmdLine)
}

func TestGetCommandLineNoQuotes(t *testing.T) {
	cmdLine := GetCommandLine("newcommand", "blub", "foobar")
	assert.Equal(t, "newcommand blub foobar", cmdLine)
}

/* ############################################# */
/* ###                Objects                ### */
/* ############################################# */

func TestLocalExecutorRun(t *testing.T) {
	e := NewLocalExecutor()
	out, code, err := e.Run(path("success.sh"))
	assert.NoError(t, err)
	assert.Equal(t, 0, code)
	assert.True(t, strings.Contains(out, "some test output here"))
}

func TestLocalExecutorRunLine(t *testing.T) {
	e := NewLocalExecutor()
	out, code, err := e.RunLine(Quote(path("success.sh")))
	assert.NoError(t, err)
	assert.Equal(t, 0, code)
	assert.True(t, strings.Contains(out, "some test output here"))
}

func TestMockExecutorRun(t *testing.T) {
	var lastCommand string
	var lastArgs []string
	e := NewMockExecutor(func(command string, args ...string) (string, int, errors.Error) {
		lastCommand = command
		lastArgs = args
		return "foobar", 42, errors.GenericError.Make()
	})
	out, code, err := e.Run(path("success.sh"), "foo", "bar")
	assert.Equal(t, "foobar", out)
	assert.True(t, errors.InstanceOf(err, errors.GenericError))
	assert.Equal(t, 42, code)
	assert.Equal(t, path("success.sh"), lastCommand)
	assert.Equal(t, []string{"foo", "bar"}, lastArgs)
}

func TestMockExecutorRunLine(t *testing.T) {
	var lastCommand string
	var lastArgs []string
	e := NewMockExecutor(func(command string, args ...string) (string, int, errors.Error) {
		lastCommand = command
		lastArgs = args
		return "foobar", 42, errors.GenericError.Make()
	})
	out, code, err := e.RunLine(path("success.sh") + " foo bar")
	assert.Equal(t, "foobar", out)
	assert.True(t, errors.InstanceOf(err, errors.GenericError))
	assert.Equal(t, 42, code)
	assert.Equal(t, path("success.sh"), lastCommand)
	assert.Equal(t, []string{"foo", "bar"}, lastArgs)
}

func TestMockExecutorRunLineParseFail(t *testing.T) {
	e := NewMockExecutor(func(command string, args ...string) (string, int, errors.Error) {
		assert.Fail(t, "Callback should not be executed on parse fail")
		return "", 0, nil
	})
	_, _, err := e.RunLine(Quote(path("args.sh")) + ` "foo test space" "bar`)
	assert.True(t, errors.InstanceOf(err, ParseError))
}

/* ############################################# */
/* ###                Helper                 ### */
/* ############################################# */

func path(cmd string) string {
	return "./test/" + cmd
}
