package ast

import (
	"fmt"

	"github.com/antha-lang/antha/antha/anthalib/wtype"
	"github.com/antha-lang/antha/graph"
)

// Options for computing dependencies
const (
	AllDeps  = iota // Follow all AST edges
	DataDeps        // Follow only consumer-producer edges
)

// A Location is a physical place
type Location interface{}

// A Node is the input to code generation. An abstract syntax tree generated
// via execution of an Antha protocol.
//
// The basic design philosophy is to capture the semantics of the Antha
// language while reducing the cases for code generation. A secondary goal is
// to ease the creation of the AST at runtime (e.g., online, incremental
// generation of nodes).
//
// Conveniently, a tree naturally expresses the single-use (i.e., destructive
// update) aspect of physical things, so the code generation keeps this
// representation longer than a traditional compiler flow would.
type Node interface {
	graph.Node
	NodeString() string
}

// A Command is high-level instruction.
type Command struct {
	From     []Node      // Inputs
	Requests []Request   // Requirements for device selection
	Inst     interface{} // Command-specific data
	Output   interface{} // Output from compilation
}

// NodeString implements graph pretty printing
func (a *Command) NodeString() string {
	return fmt.Sprintf("%+v", struct {
		Requests interface{}
		Inst     string
	}{
		Requests: a.Requests,
		Inst:     fmt.Sprintf("%T", a.Inst),
	})
}

// A UseComp is a use of a liquid component
type UseComp struct {
	From  []Node
	Value *wtype.LHComponent
}

// NodeString implements graph pretty printing
func (a *UseComp) NodeString() string {
	return fmt.Sprintf("%+v", struct {
		Value interface{}
	}{
		Value: a.Value,
	})
}

// A Bundle is an unordered collection of expressions
type Bundle struct {
	From []Node
}

// NodeString implements graph pretty printing
func (a *Bundle) NodeString() string {
	return ""
}

// A Move is a low-level move instruction
type Move struct {
	From   []*UseComp
	ToLoc  Location
	Output interface{}
}

// NodeString implements graph pretty printing
func (a *Move) NodeString() string {
	return ""
}

// A Graph is a view of the AST as a graph
type Graph struct {
	Nodes     []Node
	whichDeps int
}

// NumNodes implements a Graph
func (a *Graph) NumNodes() int {
	return len(a.Nodes)
}

// Node implements a Graph
func (a *Graph) Node(i int) graph.Node {
	return a.Nodes[i]
}

// matching returns a subset of nodes that match the predicate
func matching(pred func(Node) bool, nodes ...Node) (r []Node) {
	for _, n := range nodes {
		if !pred(n) {
			continue
		}
		r = append(r, n)
	}
	return
}

func notNil(n Node) bool {
	return n != nil
}

func setOut(n Node, i, deps int, x Node) {
	switch n := n.(type) {
	case *UseComp:
		n.From[i] = x
	case *Bundle:
		n.From[i] = x
	case *Command:
		n.From[i] = x
	case *Move:
		n.From[i] = x.(*UseComp)
	default:
		panic(fmt.Sprintf("ast.setOut: unknown node type %T", n))
	}
}

func getOut(n Node, i, deps int) Node {
	switch n := n.(type) {
	case *UseComp:
		return n.From[i]
	case *Bundle:
		return n.From[i]
	case *Command:
		return n.From[i]
	case *Move:
		return n.From[i]
	default:
		panic(fmt.Sprintf("ast.getOut: unknown node type %T", n))
	}
}

func numOuts(n Node, deps int) int {
	switch n := n.(type) {
	case *UseComp:
		return len(n.From)
	case *Bundle:
		return len(n.From)
	case *Command:
		return len(n.From)
	case *Move:
		return len(n.From)
	default:
		panic(fmt.Sprintf("ast.numOuts: unknown node type %T", n))
	}
}

// NumOuts implements a Graph
func (a *Graph) NumOuts(n graph.Node) int {
	return numOuts(n.(Node), a.whichDeps)
}

// Out implements a Graph
func (a *Graph) Out(n graph.Node, i int) graph.Node {
	return getOut(n.(Node), i, a.whichDeps)
}

// SetOut updates the ith output of node n
func (a *Graph) SetOut(n Node, i int, x Node) {
	setOut(n.(Node), a.whichDeps, i, x)
}

// A ToGraphOpt are options for ToGraph
type ToGraphOpt struct {
	Roots     []Node // Roots of program
	WhichDeps int    // Edges to follow when building graph
}

// ToGraph creates a graph from a list of roots. Include any referenced ast
// nodes in the resulting graph.
func ToGraph(opt ToGraphOpt) *Graph {
	g := &Graph{
		whichDeps: opt.WhichDeps,
	}

	seen := make(map[graph.Node]bool)
	for _, root := range opt.Roots {
		// Traverse doesn't use Graph.NumNodes() or Graph.Node(int), so we can pass
		// in our partially constructed graph to extract the reachable nodes in the
		// AST
		results, _ := graph.Visit(graph.VisitOpt{
			Graph: g,
			Root:  root,
			Visitor: func(n graph.Node) error {
				if seen[n] {
					return graph.ErrNextNode
				}
				return nil
			},
		})

		for _, k := range results.Seen.Values() {
			if seen[k] {
				continue
			}
			g.Nodes = append(g.Nodes, k.(Node))
			seen[k] = true
		}
	}

	return g
}

// Deps constructs the data dependencies between a set of commands.
func Deps(roots []Node) graph.Graph {
	g := ToGraph(ToGraphOpt{Roots: roots, WhichDeps: DataDeps})
	root := make(map[graph.Node]bool)
	for _, r := range roots {
		root[r] = true
	}
	return graph.Eliminate(graph.EliminateOpt{
		Graph: g,
		In: func(n graph.Node) bool {
			return root[n]
		},
	})
}

// FindReachingCommands returns the set of commands that have a path to the
// given nodes without any intervening commands.
func FindReachingCommands(nodes []Node) []*Command {
	g := ToGraph(ToGraphOpt{Roots: nodes, WhichDeps: DataDeps})

	var cmds []*Command
	var queue []graph.Node

	// Add immediate children to queue
	for _, node := range nodes {
		for i := 0; i < g.NumOuts(node); i++ {
			queue = append(queue, g.Out(node, i))
		}
	}

	// Breath-first search on queue
	seen := make(map[graph.Node]bool)
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]

		// Check if we've been here before
		if seen[node] {
			continue
		}
		seen[node] = true

		cmd, ok := node.(*Command)
		if ok {
			// Found a command, stop here
			cmds = append(cmds, cmd)
		} else {
			// Keep looking
			for i := 0; i < g.NumOuts(node); i++ {
				queue = append(queue, g.Out(node, i))
			}
		}
	}
	return cmds
}
