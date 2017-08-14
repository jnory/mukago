package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Args struct {
	path   string
	prefix string
}

func isStdLib(name, srcDir string) bool {
	pkg, err := build.Default.Import(name, srcDir, build.FindOnly)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	return pkg.Goroot
}

func getArgs() Args {
	f := flag.String("file", "", "path too a file.")
	prefix := flag.String("prefix", "", "prefix of the local packages.")
	flag.Parse()

	path := *f
	if path == "" {
		flag.Usage()
		os.Exit(1)
	}

	return Args{
		path:   path,
		prefix: *prefix,
	}
}

func loadFile(path string) (*token.FileSet, *ast.File, error) {
	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, path, nil, parser.ImportsOnly)
	if err != nil {
		return nil, nil, err
	}

	return fileSet, file, nil
}

func reorderImports(prefix string, srcDir string, file *ast.File) {
	for _, d := range file.Decls {
		genDecl, ok := d.(*ast.GenDecl)
		if !ok {
			continue
		}
		if genDecl.Tok != token.IMPORT {
			continue
		}
		var stdLibAst []ast.Spec
		var localLibAst []ast.Spec
		var otherLibAst []ast.Spec
		for _, s := range genDecl.Specs {
			spec := s.(*ast.ImportSpec)
			name := strings.Trim(spec.Path.Value, `"`)

			if isStdLib(name, srcDir) {
				stdLibAst = append(stdLibAst, spec)
			} else if strings.HasPrefix(name, prefix) {
				localLibAst = append(localLibAst, spec)
			} else {
				otherLibAst = append(otherLibAst, spec)
			}
		}

		sort.Slice(stdLibAst, func(i, j int) bool {
			return stdLibAst[i].(*ast.ImportSpec).Path.Value < stdLibAst[i].(*ast.ImportSpec).Path.Value
		})
		stdLibAst = append(stdLibAst, &ast.ImportSpec{Path: &ast.BasicLit{}})

		sort.Slice(otherLibAst, func(i, j int) bool {
			return otherLibAst[i].(*ast.ImportSpec).Path.Value < otherLibAst[i].(*ast.ImportSpec).Path.Value
		})
		otherLibAst = append(otherLibAst, &ast.ImportSpec{Path: &ast.BasicLit{}})

		sort.Slice(localLibAst, func(i, j int) bool {
			return localLibAst[i].(*ast.ImportSpec).Path.Value < localLibAst[i].(*ast.ImportSpec).Path.Value
		})

		genDecl.Specs = append(append(stdLibAst, otherLibAst...), localLibAst...)
	}

}

func generate(file *ast.File, cm ast.CommentMap) ([]string, map[int]int) {
	importStmts := make([]string, 0)
	parenMap := make(map[int]int)
	for _, d := range file.Decls {
		genDecl, ok := d.(*ast.GenDecl)
		if !ok {
			continue
		}
		if genDecl.Tok != token.IMPORT {
			continue
		}

		if genDecl.Lparen.IsValid() {
			// start from zero.
			parenMap[int(genDecl.Lparen)-1] = int(genDecl.Rparen)
		}

		buf := bytes.NewBufferString("import (\n")
		for _, s := range genDecl.Specs {
			spec := s.(*ast.ImportSpec)

			comments, ok := cm[spec]
			if ok {
				for _, comment := range comments {
					buf.WriteString("\t")
					buf.WriteString(comment.Text())
					buf.WriteString("\n")
				}
			}

			buf.WriteString("\t")
			if spec.Name != nil {
				buf.WriteString(spec.Name.String())
				buf.WriteString(" ")
			}
			buf.WriteString(spec.Path.Value)
			buf.WriteString("\n")
		}
		buf.WriteString(")\n")
		importStmts = append(importStmts, buf.String())
	}

	return importStmts, parenMap
}

func replaceImports(path string, importStmts []string, parenMap map[int]int) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	bodyBytes, err := ioutil.ReadAll(file)
	if err != nil {
		return "", err
	}

	buf := bytes.NewBufferString("")
	for _, imp := range importStmts {
		pos := bytes.Index(bodyBytes, []byte("import"))
		buf.Write(bodyBytes[:pos])

		end := func() int {
			next := string(bodyBytes[pos+6 : pos+7])
			if next != " " && next != "\n" && next != "(" && next != "\t" {
				return -1
			}

			for i := pos + 6; i < len(bodyBytes); i++ {
				c := string(bodyBytes[i : i+1])
				if c == "(" {
					e, ok := parenMap[i]
					if !ok {
						return -1
					}
					return e
				} else if c == " " || c == "\t" || c == "\n" {
					continue
				} else {
					for j := i; j < len(bodyBytes); j++ {
						if string(bodyBytes[j:j+1]) == "\n" {
							return j + 1
						}
					}
				}
			}
			return -1
		}()

		if end < 0 {
			return "", errors.New("Failed to find import statements.")
		}

		buf.WriteString(imp)
		buf.WriteString("\n")
		bodyBytes = bytes.TrimLeft(bodyBytes[end:], " \t\n")
	}

	buf.Write(bodyBytes)

	return buf.String(), nil
}

func main() {
	args := getArgs()

	fileSet, file, err := loadFile(args.path)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	srcDir := filepath.Dir(args.path)
	reorderImports(args.prefix, srcDir, file)

	cm := ast.NewCommentMap(fileSet, file, file.Comments)

	importStmt, parenMap := generate(file, cm)
	replaced, err := replaceImports(args.path, importStmt, parenMap)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	fmt.Fprint(os.Stdout, replaced)
}
