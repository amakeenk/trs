package cmd

import (
	"github.com/spf13/cobra"
)

// newCompletionCmd creates the completion command
func newCompletionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completion script",
		Long: `Generate shell completion script for trs.

To load completions:

Bash:
  source <(trs completion bash)

  # To load completions for each session, add to ~/.bashrc:
  echo 'source <(trs completion bash)' >> ~/.bashrc

  # Or install system-wide:
  trs completion bash > /etc/bash_completion.d/trs

Zsh:
  # If shell completion is not already enabled in your environment,
  # you will need to enable it. Add to ~/.zshrc:
  autoload -Uz compinit && compinit

  # To load completions for each session, add to ~/.zshrc:
  source <(trs completion zsh)

  # Or install system-wide:
  trs completion zsh > "${fpath[1]}/_trs"

Fish:
  trs completion fish | source

  # To load completions for each session, add to ~/.config/fish/config.fish:
  trs completion fish > ~/.config/fish/completions/trs.fish

PowerShell:
  trs completion powershell | Out-String | Invoke-Expression

  # To load completions for each session, add to $PROFILE:
  trs completion powershell >> $PROFILE
`,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.ExactValidArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			switch args[0] {
			case "bash":
				cmd.Root().GenBashCompletion(cmd.OutOrStdout())
			case "zsh":
				cmd.Root().GenZshCompletion(cmd.OutOrStdout())
			case "fish":
				cmd.Root().GenFishCompletion(cmd.OutOrStdout(), true)
			case "powershell":
				cmd.Root().GenPowerShellCompletionWithDesc(cmd.OutOrStdout())
			}
		},
	}
}
