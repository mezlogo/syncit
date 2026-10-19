package processhandlers

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	port "syncit/port/processHandlers"
)

type fakeRunner struct {
	calls []string
	err   error
}

func (f *fakeRunner) RunCommand(dryRun bool, name string, args []string) error {
	f.calls = append(f.calls, name+" "+strings.Join(args, " "))
	return f.err
}

func TestUseCase_ProcessPreFileHandlers(t *testing.T) {
	tests := []struct {
		name      string
		cmd       port.ProcessHandlersCommand
		runnerErr error
		wantCalls []string
		wantCodes []port.ProcessDecisionCode
	}{
		{
			name: "pacman grouped unique",
			cmd: port.ProcessHandlersCommand{
				Handlers:     map[string]string{"pacman": "sudo pacman -S --needed $ALL_UNIQUE_VALUES_JOIN_WHITESPACE"},
				Declarations: map[string][]string{"pacman": {"nvim", "fish", "nvim"}},
			},
			wantCalls: []string{"sudo pacman -S --needed fish nvim"},
			wantCodes: []port.ProcessDecisionCode{port.ProcessDecisionCodeExecuted},
		},
		{
			name: "aur grouped",
			cmd: port.ProcessHandlersCommand{
				Handlers:     map[string]string{"aur": "yay -S --needed $ALL_UNIQUE_VALUES_JOIN_WHITESPACE"},
				Declarations: map[string][]string{"aur": {"vscode", "leaf"}},
			},
			wantCalls: []string{"yay -S --needed leaf vscode"},
			wantCodes: []port.ProcessDecisionCode{port.ProcessDecisionCodeExecuted},
		},
		{
			name: "dry run",
			cmd: port.ProcessHandlersCommand{
				Handlers:     map[string]string{"pacman": "sudo pacman -S --needed $ALL_UNIQUE_VALUES_JOIN_WHITESPACE"},
				Declarations: map[string][]string{"pacman": {"nvim"}},
				DryRun:       true,
			},
			wantCalls: []string{"sudo pacman -S --needed nvim"},
			wantCodes: []port.ProcessDecisionCode{port.ProcessDecisionCodeDryRun},
		},
		{
			name: "runner error",
			cmd: port.ProcessHandlersCommand{
				Handlers:     map[string]string{"pacman": "sudo pacman -S --needed $ALL_UNIQUE_VALUES_JOIN_WHITESPACE"},
				Declarations: map[string][]string{"pacman": {"nvim"}},
			},
			runnerErr: errors.New("boom"),
			wantCalls: []string{"sudo pacman -S --needed nvim"},
			wantCodes: []port.ProcessDecisionCode{port.ProcessDecisionCodeFailed},
		},
		{
			name: "skips missing handler and empty values",
			cmd: port.ProcessHandlersCommand{
				Handlers:     map[string]string{"pacman": "sudo pacman -S $ALL_UNIQUE_VALUES_JOIN_WHITESPACE"},
				Declarations: map[string][]string{"aur": {"vscode"}},
			},
			wantCalls: nil,
			wantCodes: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := &fakeRunner{err: tt.runnerErr}
			result := New(runner).ProcessPreFileHandlers(tt.cmd)

			if !reflect.DeepEqual(runner.calls, tt.wantCalls) {
				t.Errorf("calls: got %#v, want %#v", runner.calls, tt.wantCalls)
			}
			if len(result.Decisions) != len(tt.wantCodes) {
				t.Fatalf("decisions: got %d, want %d", len(result.Decisions), len(tt.wantCodes))
			}
			for i, wantCode := range tt.wantCodes {
				if result.Decisions[i].Code != wantCode {
					t.Errorf("decision %d: got %q, want %q", i, result.Decisions[i].Code, wantCode)
				}
			}
		})
	}
}

func TestUseCase_ProcessPostFileHandlers_PerValue(t *testing.T) {
	runner := &fakeRunner{}
	cmd := port.ProcessHandlersCommand{
		Handlers: map[string]string{
			"systemd":     "sudo systemctl enable $UNIQUE_SINGLE_VALUE",
			"usersystemd": "systemctl --user enable $UNIQUE_SINGLE_VALUE",
		},
		Declarations: map[string][]string{
			"systemd":     {"netd.service", "resolved.service"},
			"usersystemd": {"ssh-agent.service"},
		},
	}

	result := New(runner).ProcessPostFileHandlers(cmd)

	wantCalls := []string{
		"sudo systemctl enable netd.service",
		"sudo systemctl enable resolved.service",
		"systemctl --user enable ssh-agent.service",
	}
	if !reflect.DeepEqual(runner.calls, wantCalls) {
		t.Errorf("calls: got %#v, want %#v", runner.calls, wantCalls)
	}
	if len(result.Decisions) != 3 {
		t.Errorf("decisions: got %d, want 3", len(result.Decisions))
	}
}
