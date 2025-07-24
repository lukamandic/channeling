package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/spf13/cobra"
)

type ChannelInfo struct {
	Name         string
	Type         string
	Location     string
	Declaration  string
	SendOps      []string
	ReceiveOps   []string
	ReturnedFrom []string
	PassedTo     []string
	UsedInFiles  []string
	mu           sync.RWMutex
}

func main() {
	var rootCmd = &cobra.Command{
		Use:   "channeling",
		Short: "A tool to analyze Go code and detect channel usage",
		Long:  `A CLI tool that analyzes Go code and detects channel declarations and usage patterns.`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				fmt.Println("Please provide a directory path to analyze")
				return
			}
			analyzeDirectory(args[0])
		},
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func analyzeDirectory(dirPath string) {
	fset := token.NewFileSet()
	channels := make(map[string]*ChannelInfo)
	var wg sync.WaitGroup
	var mu sync.Mutex

	fileChan := make(chan string, 100)

	numWorkers := 4
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for filePath := range fileChan {
				analyzeFile(fset, filePath, channels, &mu)
			}
		}()
	}

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".go") {
			fileChan <- path
		}
		return nil
	})

	close(fileChan)
	wg.Wait()

	if err != nil {
		fmt.Printf("Error walking directory: %v\n", err)
		return
	}

	printChannelInfo(channels)
}

func analyzeFile(fset *token.FileSet, filePath string, channels map[string]*ChannelInfo, mu *sync.Mutex) {
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		fmt.Printf("Error parsing file %s: %v\n", filePath, err)
		return
	}

	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.AssignStmt:
			for i, v := range x.Rhs {
				if call, ok := v.(*ast.CallExpr); ok {
					if fun, ok := call.Fun.(*ast.Ident); ok && fun.Name == "make" {
						if len(call.Args) > 0 {
							if chanType, ok := call.Args[0].(*ast.ChanType); ok {
								if len(x.Lhs) > i {
									if ident, ok := x.Lhs[i].(*ast.Ident); ok {
										pos := fset.Position(x.Pos())
										mu.Lock()
										channels[ident.Name] = &ChannelInfo{
											Name:        ident.Name,
											Type:        fmt.Sprintf("chan %s", getTypeString(chanType.Value)),
											Location:    fmt.Sprintf("%s:%d", filePath, pos.Line),
											Declaration: fmt.Sprintf("Declared at %s:%d", filePath, pos.Line),
											SendOps:     make([]string, 0, 10),
											ReceiveOps:  make([]string, 0, 10),
											ReturnedFrom: make([]string, 0, 5),
											PassedTo:    make([]string, 0, 5),
											UsedInFiles: []string{filePath},
										}
										mu.Unlock()
									}
								}
							}
						}
					}
				}
			}
		case *ast.SendStmt:
			if ident, ok := x.Chan.(*ast.Ident); ok {
				if channel, exists := channels[ident.Name]; exists {
					pos := fset.Position(x.Pos())
					channel.mu.Lock()
					channel.SendOps = append(channel.SendOps, fmt.Sprintf("%s:%d", filePath, pos.Line))
					channel.UsedInFiles = appendIfNotExists(channel.UsedInFiles, filePath)
					channel.mu.Unlock()
				}
			}
		case *ast.UnaryExpr:
			if x.Op == token.ARROW {
				if ident, ok := x.X.(*ast.Ident); ok {
					if channel, exists := channels[ident.Name]; exists {
						pos := fset.Position(x.Pos())
						channel.mu.Lock()
						channel.ReceiveOps = append(channel.ReceiveOps, fmt.Sprintf("%s:%d", filePath, pos.Line))
						channel.UsedInFiles = appendIfNotExists(channel.UsedInFiles, filePath)
						channel.mu.Unlock()
					}
				}
			}
		case *ast.SelectStmt:
			for _, caseClause := range x.Body.List {
				if caseClause, ok := caseClause.(*ast.CommClause); ok {
					if comm, ok := caseClause.Comm.(*ast.AssignStmt); ok {
						// Handle receive in select
						if unary, ok := comm.Rhs[0].(*ast.UnaryExpr); ok && unary.Op == token.ARROW {
							if ident, ok := unary.X.(*ast.Ident); ok {
								if channel, exists := channels[ident.Name]; exists {
									pos := fset.Position(comm.Pos())
									channel.mu.Lock()
									channel.ReceiveOps = append(channel.ReceiveOps, fmt.Sprintf("%s:%d (select)", filePath, pos.Line))
									channel.UsedInFiles = appendIfNotExists(channel.UsedInFiles, filePath)
									channel.mu.Unlock()
								}
							}
						}
					} else if comm, ok := caseClause.Comm.(*ast.SendStmt); ok {
						if ident, ok := comm.Chan.(*ast.Ident); ok {
							if channel, exists := channels[ident.Name]; exists {
								pos := fset.Position(comm.Pos())
								channel.mu.Lock()
								channel.SendOps = append(channel.SendOps, fmt.Sprintf("%s:%d (select)", filePath, pos.Line))
								channel.UsedInFiles = appendIfNotExists(channel.UsedInFiles, filePath)
								channel.mu.Unlock()
							}
						}
					}
				}
			}
		case *ast.FuncDecl:
			if x.Type.Results != nil {
				for _, result := range x.Type.Results.List {
					if _, ok := result.Type.(*ast.ChanType); ok {
						if len(result.Names) > 0 {
							if ident := result.Names[0]; ident != nil {
								if channel, exists := channels[ident.Name]; exists {
									pos := fset.Position(x.Pos())
									channel.mu.Lock()
									channel.ReturnedFrom = append(channel.ReturnedFrom, fmt.Sprintf("%s:%d", filePath, pos.Line))
									channel.UsedInFiles = appendIfNotExists(channel.UsedInFiles, filePath)
									channel.mu.Unlock()
								}
							}
						}
					}
				}
			}
			if x.Type.Params != nil {
				for _, param := range x.Type.Params.List {
					if _, ok := param.Type.(*ast.ChanType); ok {
						for _, name := range param.Names {
							if channel, exists := channels[name.Name]; exists {
								pos := fset.Position(x.Pos())
								channel.mu.Lock()
								channel.PassedTo = append(channel.PassedTo, fmt.Sprintf("%s:%d", filePath, pos.Line))
								channel.UsedInFiles = appendIfNotExists(channel.UsedInFiles, filePath)
								channel.mu.Unlock()
							}
						}
					}
				}
			}
		}
		return true
	})
}

func getTypeString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		return fmt.Sprintf("%s.%s", getTypeString(t.X), t.Sel.Name)
	default:
		return "unknown"
	}
}

func appendIfNotExists(slice []string, str string) []string {
	for _, s := range slice {
		if s == str {
			return slice
		}
	}
	return append(slice, str)
}

func printChannelInfo(channels map[string]*ChannelInfo) {
	if len(channels) == 0 {
		fmt.Println("No channels found in the analyzed code.")
		return
	}

	fmt.Println("\nChannel Analysis Results:")
	fmt.Println("========================")
	for _, channel := range channels {
		fmt.Printf("\nChannel: %s\n", channel.Name)
		fmt.Printf("Type: %s\n", channel.Type)
		fmt.Printf("Declaration: %s\n", channel.Declaration)
		
		if len(channel.SendOps) > 0 {
			fmt.Println("\nSend Operations:")
			for _, op := range channel.SendOps {
				fmt.Printf("  - %s\n", op)
			}
		}
		
		if len(channel.ReceiveOps) > 0 {
			fmt.Println("\nReceive Operations:")
			for _, op := range channel.ReceiveOps {
				fmt.Printf("  - %s\n", op)
			}
		}

		if len(channel.ReturnedFrom) > 0 {
			fmt.Println("\nReturned From Functions:")
			for _, fn := range channel.ReturnedFrom {
				fmt.Printf("  - %s\n", fn)
			}
		}

		if len(channel.PassedTo) > 0 {
			fmt.Println("\nPassed To Functions:")
			for _, fn := range channel.PassedTo {
				fmt.Printf("  - %s\n", fn)
			}
		}

		if len(channel.UsedInFiles) > 1 {
			fmt.Println("\nUsed In Files:")
			for _, file := range channel.UsedInFiles {
				fmt.Printf("  - %s\n", file)
			}
		}
		
		fmt.Println("------------------------")
	}

	visualizeChannels(channels)

	startWebServer(channels)
} 