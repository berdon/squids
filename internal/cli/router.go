package cli

import (
	"fmt"
	"os"
)

// Run executes the sq CLI with pre-sliced args (excluding argv[0])
// and returns a process exit code.
func Run(args []string) int {
	if len(args) < 1 {
		usage()
		return 2
	}

	switch args[0] {
	case "-h", "--help":
		usage()
		return 0
	case "help":
		return cmdHelp(args[1:])
	case "-V", "--version":
		return cmdVersion(nil)
	case "init":
		return cmdInit()
	case "ready":
		return cmdReady()
	case "create":
		return cmdCreate(args[1:])
	case "q":
		return cmdQ(args[1:])
	case "show":
		return cmdShow(args[1:])
	case "list":
		return cmdList(args[1:])
	case "update":
		return cmdUpdate(args[1:])
	case "close":
		return cmdClose(args[1:])
	case "reopen":
		return cmdReopen(args[1:])
	case "delete":
		return cmdDelete(args[1:])
	case "label":
		return cmdLabel(args[1:])
	case "dep":
		return cmdDep(args[1:])
	case "comments":
		return cmdComments(args[1:])
	case "todo":
		return cmdTodo(args[1:])
	case "children":
		return cmdChildren(args[1:])
	case "blocked":
		return cmdBlocked(args[1:])
	case "defer":
		return cmdDefer(args[1:])
	case "undefer":
		return cmdUndefer(args[1:])
	case "set-state":
		return cmdSetState(args[1:])
	case "rename":
		return cmdRename(args[1:])
	case "rename-prefix":
		return cmdRenamePrefix(args[1:])
	case "duplicate":
		return cmdDuplicate(args[1:])
	case "supersede":
		return cmdSupersede(args[1:])
	case "types":
		return cmdTypes(args[1:])
	case "query":
		return cmdQuery(args[1:])
	case "stale":
		return cmdStale(args[1:])
	case "orphans":
		return cmdOrphans(args[1:])
	case "search":
		return cmdSearch(args[1:])
	case "gate":
		return cmdGate(args[1:])
	case "backup":
		return cmdBackup(args[1:])
	case "purge":
		return cmdPurge(args[1:])
	case "restore":
		return cmdRestore(args[1:])
	case "count":
		return cmdCount(args[1:])
	case "status", "stats":
		return cmdStatus()
	case "version":
		return cmdVersion(args[1:])
	case "where":
		return cmdWhere(args[1:])
	case "info":
		return cmdInfo(args[1:])
	case "human":
		return cmdHuman(args[1:])
	case "quickstart":
		return cmdQuickstart(args[1:])
	case "mail":
		return cmdMail(args[1:])
	case "mol":
		return cmdMol(args[1:])
	case "setup":
		return cmdSetup(args[1:])
	case "history":
		return cmdHistory(args[1:])
	case "audit":
		return cmdAudit(args[1:])
	case "swarm":
		return cmdSwarm(args[1:])
	case "hooks":
		return cmdHooks(args[1:])
	case "completion":
		return cmdCompletion(args[1:])
	case "onboard":
		return cmdOnboard(args[1:])
	case "import-beads":
		return cmdImportBeads(args[1:])
	case "gitlab":
		return cmdGitLab(args[1:])
	case "memories":
		return cmdMemories(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", args[0])
		usage()
		return 2
	}
}
