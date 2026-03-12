# Test Plan: `sq completion zsh`

## Goal
Verify that `sq completion zsh`:
- prints the expected zsh completion script
- advertises the correct help/usage surface
- supports the documented flags for this tuple
- can be loaded in a clean zsh session
- returns usable completion candidates for both top-level `sq` commands and `sq completion` shell names

## Initialization Steps
1. Create a disposable workspace and enter the repository root.
2. Build the CLI:
   ```bash
   mkdir -p bin
   go build -o ./bin/sq ./cmd/sq
   ```
3. Confirm the target command exists:
   ```bash
   ./bin/sq completion zsh --help
   ```
4. Ensure `zsh` is available in `PATH`:
   ```bash
   command -v zsh
   ```
5. Use a clean shell session for behavior checks so user dotfiles do not affect results:
   ```bash
   zsh -f
   ```

## Testing Steps

### 1. Help surface
1. Verify help exits successfully:
   ```bash
   ./bin/sq completion zsh --help
   ```
2. Confirm output contains:
   - `Generate the autocompletion script for the zsh shell.`
   - `Usage:`
   - `sq completion zsh [flags]`
   - `--no-descriptions`
   - `Global Flags:`
3. Verify unknown flags fail with usage-style output:
   ```bash
   ./bin/sq completion zsh --wat
   ```

### 2. Script generation
1. Generate the script:
   ```bash
   ./bin/sq completion zsh > /tmp/sq-completion-zsh
   ```
2. Confirm the file is non-empty and starts with zsh completion headers:
   ```bash
   head -20 /tmp/sq-completion-zsh
   ```
3. Confirm the generated script includes:
   - `#compdef sq`
   - `compdef _sq sq`
   - `_sq()`
   - `compadd -- $commands`
   - `compadd -- $shells`

### 3. Functional top-level completion behavior
1. Start a clean zsh process:
   ```bash
   zsh -f
   ```
2. In that shell, initialize completion and source the generated script:
   ```zsh
   autoload -Uz compinit && compinit
   source /tmp/sq-completion-zsh
   ```
3. Stub `compadd` to print candidates, then invoke the completer for a top-level command prefix:
   ```zsh
   compadd() { shift; printf '%s\n' "$@"; }
   words=(sq up)
   CURRENT=2
   _sq
   ```
4. Confirm the results include `update`.
5. Confirm the output does **not** include help descriptions such as `Update a task`.

### 4. Functional shell-subcommand completion behavior
1. In the same clean zsh shell, request shell-name completion:
   ```zsh
   words=(sq completion z)
   CURRENT=3
   _sq
   ```
2. Confirm the results include `zsh`.
3. Confirm the results are raw shell names only, not descriptive text.
4. Repeat with:
   ```zsh
   words=(sq completion p)
   CURRENT=3
   _sq
   ```
5. Confirm the result includes `powershell`.

### 5. Flag tolerance checks
1. Verify documented no-op flag acceptance still exits successfully:
   ```bash
   ./bin/sq completion zsh --no-descriptions
   ```
2. Verify global compatibility flags are accepted:
   ```bash
   ./bin/sq completion zsh --actor tester --db /tmp/sq-test.db --dolt-auto-commit off
   ```
3. Confirm both commands still emit a valid completion script.

### 6. Regression checks against recent completion fixes
1. Confirm top-level command completion includes commonly used commands such as:
   - `completion`
   - `update`
   - `ready`
   - `show`
2. Confirm `sq completion zsh` does not rely on user shell startup files.
3. Confirm command names with hyphens are offered intact, e.g. `rename-prefix` and `import-beads`.

## Cleanup Steps
1. Remove generated temporary script files:
   ```bash
   rm -f /tmp/sq-completion-zsh
   ```
2. Remove any temporary database or sandbox files created during testing.
3. Exit the clean zsh shell.
4. If a disposable worktree or temp directory was created only for this test, remove it.

## Expected Result
- Help output matches the documented command surface.
- Generated zsh completion script is loadable.
- Completion candidates are functional and value-oriented.
- Top-level command completion and `completion` shell completion both work in a clean zsh environment.
- No cleanup artifacts remain after the test session.
