package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

const wasmInputDir = "/hook-input"

// Exec resolves a configured command and runs it with its full argv and environment.
type Exec interface {
	Resolve(command, dir string) (string, error)
	CombinedOutput(path string, args []string, dir string, env []string, inputDir string) ([]byte, error)
}

func hookExecutor(name string) (Exec, error) {
	switch name {
	case "", "process":
		return processExec{}, nil
	case "wasm":
		return wasmExec{}, nil
	default:
		return nil, fmt.Errorf("unknown command-executor %q", name)
	}
}

func commandPath(command, dir string) string {
	if filepath.IsAbs(command) || dir == "" {
		return command
	}
	return filepath.Join(dir, command)
}

type processExec struct{}

func (processExec) Resolve(command, dir string) (string, error) {
	return exec.LookPath(commandPath(command, dir))
}

func (processExec) CombinedOutput(path string, args []string, dir string, env []string, _ string) ([]byte, error) {
	cmd := exec.Command(path)
	cmd.Args = args
	cmd.Dir = dir
	cmd.Env = env
	return cmd.CombinedOutput()
}

type wasmExec struct{}

func (wasmExec) Resolve(command, dir string) (string, error) {
	if command == "" {
		return "", fmt.Errorf("missing WebAssembly module path")
	}
	path, err := filepath.Abs(commandPath(command, dir))
	if err != nil {
		return "", err
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("WebAssembly module %q is not a regular file", path)
	}
	return path, nil
}

func (wasmExec) CombinedOutput(path string, args []string, dir string, env []string, inputDir string) ([]byte, error) {
	// Use the server's PATH to locate Wasmer, just as the process executor
	// resolves executables before applying request-derived environment values.
	runtimeArgs := []string{"run", "--no-tty", "--net", "--forward-host-env"}
	if dir != "" {
		absoluteDir, err := filepath.Abs(dir)
		if err != nil {
			return nil, err
		}
		runtimeArgs = append(runtimeArgs, "--volume", absoluteDir+":/work", "--cwd", "/work")
	}
	if inputDir != "" {
		absoluteInputDir, err := filepath.Abs(inputDir)
		if err != nil {
			return nil, err
		}
		runtimeArgs = append(runtimeArgs, "--volume", absoluteInputDir+":"+wasmInputDir)
	}
	runtimeArgs = append(runtimeArgs, path, "--")
	// Wasmer sets the guest's argv[0] to the module name.
	runtimeArgs = append(runtimeArgs, args[1:]...)
	cmd := exec.Command("wasmer", runtimeArgs...)
	cmd.Dir = dir
	cmd.Env = env
	return cmd.CombinedOutput()
}
