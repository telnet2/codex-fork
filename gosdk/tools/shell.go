package tools

import (
	"runtime"

	"github.com/anthropics/codex-fork/gosdk/schema"
)

// ShellTool returns the shell tool specification.
// This mirrors create_shell_tool() in codex-rs/core/src/tools/spec.rs
var ShellTool = func() *ToolSpec {
	properties := map[string]*schema.JSONSchema{
		"command": schema.Array(
			schema.String(""),
			"The command to execute",
		),
		"workdir": schema.String(
			"The working directory to execute the command in",
		),
		"timeout_ms": schema.Number(
			"The timeout for the command in milliseconds",
		),
		"sandbox_permissions": schema.String(
			`Sandbox permissions for the command. Set to "require_escalated" to request running without sandbox restrictions; defaults to "use_default".`,
		),
		"justification": schema.String(
			`Only set if sandbox_permissions is "require_escalated". 1-sentence explanation of why we want to run this command.`,
		),
	}

	var description string
	if runtime.GOOS == "windows" {
		description = `Runs a Powershell command (Windows) and returns its output. Arguments to ` + "`shell`" + ` will be passed to CreateProcessW(). Most commands should be prefixed with ["powershell.exe", "-Command"].

Examples of valid command strings:

- ls -a (show hidden): ["powershell.exe", "-Command", "Get-ChildItem -Force"]
- recursive find by name: ["powershell.exe", "-Command", "Get-ChildItem -Recurse -Filter *.py"]
- recursive grep: ["powershell.exe", "-Command", "Get-ChildItem -Path C:\\myrepo -Recurse | Select-String -Pattern 'TODO' -CaseSensitive"]
- ps aux | grep python: ["powershell.exe", "-Command", "Get-Process | Where-Object { $_.ProcessName -like '*python*' }"]
- setting an env var: ["powershell.exe", "-Command", "$env:FOO='bar'; echo $env:FOO"]
- running an inline Python script: ["powershell.exe", "-Command", "@'\\nprint('Hello, world!')\\n'@ | python -"]`
	} else {
		description = `Runs a shell command and returns its output.
- The arguments to ` + "`shell`" + ` will be passed to execvp(). Most terminal commands should be prefixed with ["bash", "-lc"].
- Always set the ` + "`workdir`" + ` param when using the shell function. Do not use ` + "`cd`" + ` unless absolutely necessary.`
	}

	return &ToolSpec{
		Type:        ToolTypeFunction,
		Name:        "shell",
		Description: description,
		Strict:      false,
		Parameters: schema.Object(
			properties,
			[]string{"command"},
		),
	}
}()

// ShellCommandTool returns the shell_command tool specification.
// This mirrors create_shell_command_tool() in codex-rs/core/src/tools/spec.rs
var ShellCommandTool = func() *ToolSpec {
	properties := map[string]*schema.JSONSchema{
		"command": schema.String(
			"The shell script to execute in the user's default shell",
		),
		"workdir": schema.String(
			"The working directory to execute the command in",
		),
		"login": schema.Boolean(
			"Whether to run the shell with login shell semantics. Defaults to false unless a shell snapshot is available.",
		),
		"timeout_ms": schema.Number(
			"The timeout for the command in milliseconds",
		),
		"sandbox_permissions": schema.String(
			`Sandbox permissions for the command. Set to "require_escalated" to request running without sandbox restrictions; defaults to "use_default".`,
		),
		"justification": schema.String(
			`Only set if sandbox_permissions is "require_escalated". 1-sentence explanation of why we want to run this command.`,
		),
	}

	var description string
	if runtime.GOOS == "windows" {
		description = `Runs a Powershell command (Windows) and returns its output.

Examples of valid command strings:

- ls -a (show hidden): "Get-ChildItem -Force"
- recursive find by name: "Get-ChildItem -Recurse -Filter *.py"
- recursive grep: "Get-ChildItem -Path C:\\myrepo -Recurse | Select-String -Pattern 'TODO' -CaseSensitive"
- ps aux | grep python: "Get-Process | Where-Object { $_.ProcessName -like '*python*' }"
- setting an env var: "$env:FOO='bar'; echo $env:FOO"
- running an inline Python script: "@'\\nprint('Hello, world!')\\n'@ | python -`
	} else {
		description = `Runs a shell command and returns its output.
- Always set the ` + "`workdir`" + ` param when using the shell_command function. Do not use ` + "`cd`" + ` unless absolutely necessary.`
	}

	return &ToolSpec{
		Type:        ToolTypeFunction,
		Name:        "shell_command",
		Description: description,
		Strict:      false,
		Parameters: schema.Object(
			properties,
			[]string{"command"},
		),
	}
}()

// ExecCommandTool returns the exec_command tool specification for unified exec.
// This mirrors create_exec_command_tool() in codex-rs/core/src/tools/spec.rs
var ExecCommandTool = func() *ToolSpec {
	properties := map[string]*schema.JSONSchema{
		"cmd": schema.String(
			"Shell command to execute.",
		),
		"workdir": schema.String(
			"Optional working directory to run the command in; defaults to the turn cwd.",
		),
		"shell": schema.String(
			"Shell binary to launch. Defaults to /bin/bash.",
		),
		"login": schema.Boolean(
			"Whether to run the shell with -l/-i semantics. Defaults to false unless a shell snapshot is available.",
		),
		"yield_time_ms": schema.Number(
			"How long to wait (in milliseconds) for output before yielding.",
		),
		"max_output_tokens": schema.Number(
			"Maximum number of tokens to return. Excess output will be truncated.",
		),
		"sandbox_permissions": schema.String(
			`Sandbox permissions for the command. Set to "require_escalated" to request running without sandbox restrictions; defaults to "use_default".`,
		),
		"justification": schema.String(
			`Only set if sandbox_permissions is "require_escalated". 1-sentence explanation of why we want to run this command.`,
		),
	}

	return &ToolSpec{
		Type:        ToolTypeFunction,
		Name:        "exec_command",
		Description: "Runs a command in a PTY, returning output or a session ID for ongoing interaction.",
		Strict:      false,
		Parameters: schema.Object(
			properties,
			[]string{"cmd"},
		),
	}
}()

// WriteStdinTool returns the write_stdin tool specification.
// This mirrors create_write_stdin_tool() in codex-rs/core/src/tools/spec.rs
var WriteStdinTool = func() *ToolSpec {
	properties := map[string]*schema.JSONSchema{
		"session_id": schema.Number(
			"Identifier of the running unified exec session.",
		),
		"chars": schema.String(
			"Bytes to write to stdin (may be empty to poll).",
		),
		"yield_time_ms": schema.Number(
			"How long to wait (in milliseconds) for output before yielding.",
		),
		"max_output_tokens": schema.Number(
			"Maximum number of tokens to return. Excess output will be truncated.",
		),
	}

	return &ToolSpec{
		Type:        ToolTypeFunction,
		Name:        "write_stdin",
		Description: "Writes characters to an existing unified exec session and returns recent output.",
		Strict:      false,
		Parameters: schema.Object(
			properties,
			[]string{"session_id"},
		),
	}
}()
