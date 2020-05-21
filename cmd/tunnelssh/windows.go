// +build windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/balibuild/tunnelssh/cli"
	"github.com/balibuild/winio"
	"github.com/mattn/go-isatty"
	"golang.org/x/crypto/ssh/terminal"
	"golang.org/x/sys/windows/registry"
)

// HKCU\Software\Microsoft\Windows\CurrentVersion\Internet Settings

// ResolveRegistryProxy todo
func ResolveRegistryProxy() (*ProxySettings, error) {
	k, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Internet Settings`, registry.QUERY_VALUE)
	if err != nil {
		return nil, err
	}
	defer k.Close()
	ps := &ProxySettings{sep: ";"}
	if d, _, err := k.GetIntegerValue("ProxyEnable"); err == nil && d == 1 {
		if s, _, err := k.GetStringValue("ProxyServer"); err == nil && len(s) > 0 {
			ps.ProxyServer = s
		}
	} else {
		if s, _, err := k.GetStringValue("AutoConfigURL"); err == nil && len(s) > 0 {
			ps.ProxyServer = s
		}
	}
	if s, _, err := k.GetStringValue("ProxyOverride"); err == nil && len(s) > 0 {
		ps.ProxyOverride = s
	}
	if ps.ProxyServer != "" {
		return ps, nil
	}
	return nil, ErrProxyNotConfigured
}

// feature read proxy from registry

// ResolveProxy todo
func ResolveProxy() (*ProxySettings, error) {
	if s, err := ResolveRegistryProxy(); err == nil {
		return s, nil
	}
	ps := &ProxySettings{sep: ","}
	ps.ProxyOverride = os.Getenv("NO_PROXY")
	if ps.ProxyServer = os.Getenv("SSH_PROXY"); len(ps.ProxyServer) > 0 {
		return ps, nil
	}
	if ps.ProxyServer = os.Getenv("HTTPS_PROXY"); len(ps.ProxyServer) > 0 {
		return ps, nil
	}
	if ps.ProxyServer = os.Getenv("HTTP_PROXY"); len(ps.ProxyServer) > 0 {
		return ps, nil
	}
	if ps.ProxyServer = os.Getenv("ALL_PROXY"); len(ps.ProxyServer) > 0 {
		return ps, nil
	}
	return nil, ErrProxyNotConfigured
}

// MakeAgent make agent
// Windows use pipe now
// https://github.com/PowerShell/openssh-portable/blob/latestw_all/contrib/win32/win32compat/ssh-agent/agent.c#L40
func (ka *KeyAgent) MakeAgent() error {
	if len(os.Getenv("SSH_AUTH_SOCK")) == 0 {
		return cli.ErrorCat("ssh agent not initialized")
	}
	// \\\\.\\pipe\\openssh-ssh-agent
	conn, err := winio.DialPipe("\\\\.\\pipe\\openssh-ssh-agent", nil)
	if err != nil {
		return err
	}
	ka.conn = conn
	return nil
}

// Windows Terminaol WT_SESSION todo
// Mintty TTY

// IsTerminal todo
func IsTerminal(fd *os.File) bool {
	return isatty.IsTerminal(fd.Fd()) || isatty.IsCygwinTerminal(fd.Fd())
}

// ReadPassPhrase todo
// openssh-portable-8.1.0.0\contrib\win32\win32compat\msic.c

// save_state=$(stty -g)
//

// http://man7.org/linux/man-pages/man1/stty.1.html

// \Device\NamedPipe\msys-1888ae32e00d56aa-pty0-echoloop
// Cygwin uses special named pipes to simulate TTY, and Mintty listens to these obvious
// pipes to control the terminal emulator, adjust its size, mode, and so on.
//
// If we parse these pipelines to control Mintty, this is a little more complicated.
// In short, we can use stty to complete this work.
// https://github.com/git/git/blob/master/compat/terminal.c#L90

// enableEchoInput only support cygterminal
func enableEchoInput() (string, error) {
	cmd := exec.Command("stty", "-g")
	cmd.Stdin = os.Stdin
	buf, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	cmd2 := exec.Command("stty", "-echo")
	cmd2.Stdin = os.Stdin
	cmd2.Stderr = os.Stderr
	if err := cmd2.Run(); err != nil {
		return "", err
	}
	return strings.TrimSpace(string(buf)), nil
}

// restoreInput only support cygterminal
func restoreInput(state string) error {
	cmd := exec.Command("stty", state)
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

//AskPassword ask password
func AskPassword(prompt string) (string, error) {
	if fd := int(os.Stdin.Fd()); terminal.IsTerminal(fd) {
		fmt.Fprintf(os.Stderr, "%s: ", prompt)
		pwd, err := terminal.ReadPassword(fd)
		if err != nil {
			return "", err
		}
		return string(pwd), nil
	}
	if isatty.IsCygwinTerminal(os.Stdin.Fd()) {
		state, err := enableEchoInput()
		if err != nil {
			return "", err
		}
		defer restoreInput(state)
		fmt.Fprintf(os.Stderr, "%s: ", prompt)
		pwd, err := ReadInput(os.Stdin, true)
		if err != nil {
			return "", err
		}
		return string(pwd), nil
	}
	return readAskPass(prompt, AskEcho)
}

// AskPrompt todo
func AskPrompt(prompt string) (string, error) {
	if fd := int(os.Stdin.Fd()); terminal.IsTerminal(fd) {
		fmt.Fprintf(os.Stderr, "%s: ", prompt)
		respond, err := ReadInput(os.Stdin, false)
		if err != nil {
			return "", err
		}
		return string(respond), nil
	}
	if isatty.IsCygwinTerminal(os.Stdin.Fd()) {
		fmt.Fprintf(os.Stderr, "%s: ", prompt)
		respond, err := ReadInput(os.Stdin, true)
		if err != nil {
			return "", err
		}
		return string(respond), nil
	}
	return readAskPass(prompt, AskNone)
}

// DefaultKnownHosts todo
func DefaultKnownHosts() string {
	return os.ExpandEnv("$USERPROFILE\\.ssh\\known_hosts")
}

// Initialize todo
func (ks *KeySearcher) Initialize() error {
	ks.home = os.ExpandEnv("$USERPROFILE")
	return nil
}
