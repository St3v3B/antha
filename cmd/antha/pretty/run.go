package pretty

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/antha-lang/antha/execute"
	"github.com/antha-lang/antha/target"
	"github.com/antha-lang/antha/target/auto"
	"golang.org/x/net/context"
)

func shouldWait(inst target.Inst) bool {
	switch inst.(type) {
	case *target.Run:
		return true
	}
	return false
}

// Run executes an execute.Result against the given auto target.
func Run(out io.Writer, in io.Reader, a *auto.Auto, result *execute.Result) error {
	if _, err := fmt.Fprintf(out, "== Running Workflow:\n"); err != nil {
		return err
	}

	bin := bufio.NewReader(in)
	ctx := context.Background()
	for _, inst := range result.Insts {
		if _, err := fmt.Fprintf(out, "    * %s", a.Pretty(inst)); err != nil {
			return err
		}

		var skip bool
		if shouldWait(inst) {
			fmt.Fprintf(out, " (Run? [yes,skip]) ") // nolint
			s, err := bin.ReadString('\n')
			if err != nil {
				return err
			}
			skip = true
			if strings.HasPrefix(s, "yes") {
				skip = false
			}
		}

		if !skip {
			if err := a.Execute(ctx, inst); err != nil {
				fmt.Fprintf(out, " [FAIL]\n") // nolint
				return err
			}
		}

		if _, err := fmt.Fprintf(out, " [OK]\n"); err != nil {
			return err
		}
	}
	return nil
}
