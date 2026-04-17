// Package shell parses shell command strings and flattens them into individual
// commands split by pipes, &&, ||, ;, subshells, and command substitutions.
//
// The logic is adapted from github.com/gurisugi/shs.
package shell

import (
	"strings"

	"mvdan.cc/sh/v3/syntax"
)

// SplitCommands parses a shell command string and returns individual commands
// flattened from chains (pipes, &&, ||, ;) and command substitutions.
func SplitCommands(command string) ([]string, error) {
	_, commands, err := parseCommands(command)
	return commands, err
}

// CommandNames parses a shell command string and returns the name (first word)
// of each command.
func CommandNames(command string) ([]string, error) {
	calls, _, err := parseCommands(command)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, call := range calls {
		names = append(names, extractName(call))
	}
	return names, nil
}

func parseCommands(command string) ([]*syntax.CallExpr, []string, error) {
	parser := syntax.NewParser()
	prog, err := parser.Parse(strings.NewReader(command), "")
	if err != nil {
		return nil, nil, err
	}

	var calls []*syntax.CallExpr
	var commands []string
	for _, stmt := range prog.Stmts {
		collectFromStmt(stmt, &calls, &commands)
	}
	return calls, commands, nil
}

func collectFromStmt(stmt *syntax.Stmt, calls *[]*syntax.CallExpr, out *[]string) {
	if stmt == nil {
		return
	}
	collectFromCmd(stmt.Cmd, calls, out)
}

func collectFromCmd(cmd syntax.Command, calls *[]*syntax.CallExpr, out *[]string) {
	if cmd == nil {
		return
	}
	switch c := cmd.(type) {
	case *syntax.BinaryCmd:
		collectFromStmt(c.X, calls, out)
		collectFromStmt(c.Y, calls, out)
	case *syntax.Subshell:
		for _, stmt := range c.Stmts {
			collectFromStmt(stmt, calls, out)
		}
	case *syntax.CallExpr:
		*calls = append(*calls, c)
		*out = append(*out, printRedacted(cmd))
		collectSubstitutions(cmd, calls, out)
	default:
		*calls = append(*calls, nil)
		*out = append(*out, printRedacted(cmd))
		collectSubstitutions(cmd, calls, out)
	}
}

// collectSubstitutions walks the node to find command substitutions $()
// and collects the commands within them.
func collectSubstitutions(node syntax.Node, calls *[]*syntax.CallExpr, out *[]string) {
	syntax.Walk(node, func(n syntax.Node) bool {
		cs, ok := n.(*syntax.CmdSubst)
		if !ok {
			return true
		}
		for _, stmt := range cs.Stmts {
			collectFromStmt(stmt, calls, out)
		}
		return false
	})
}

// extractName extracts the command name (first word) from a CallExpr.
func extractName(call *syntax.CallExpr) string {
	if call == nil || len(call.Args) == 0 {
		return ""
	}
	return wordToLiteral(call.Args[0])
}

// wordToLiteral returns the literal string from a Word node.
// Returns an empty string if the word contains variable expansions or command substitutions.
func wordToLiteral(word *syntax.Word) string {
	var sb strings.Builder
	for _, part := range word.Parts {
		lit, ok := part.(*syntax.Lit)
		if !ok {
			return ""
		}
		sb.WriteString(lit.Value)
	}
	return sb.String()
}

// printRedacted prints a node with command substitution contents replaced by $(...).
func printRedacted(node syntax.Node) string {
	var sb strings.Builder
	printer := syntax.NewPrinter(syntax.Minify(true))

	// temporarily clear Stmts in command substitutions for printing
	var saved []savedSubst
	syntax.Walk(node, func(n syntax.Node) bool {
		cs, ok := n.(*syntax.CmdSubst)
		if !ok {
			return true
		}
		saved = append(saved, savedSubst{cs: cs, stmts: cs.Stmts})
		cs.Stmts = nil
		return true
	})

	_ = printer.Print(&sb, node)

	// restore Stmts
	for _, s := range saved {
		s.cs.Stmts = s.stmts
	}

	return strings.TrimSpace(sb.String())
}

type savedSubst struct {
	cs    *syntax.CmdSubst
	stmts []*syntax.Stmt
}
