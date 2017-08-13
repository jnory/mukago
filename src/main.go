package main

import (
	"go/token"
	"flag"
	"go/parser"
	"log"
	"go/build"
	"os"
	"go/ast"
	"strings"
	"go/printer"
	"path/filepath"
	"sort"
	"reflect"
)

type Args struct {
	path string
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

func getArgs() Args{
	f := flag.String("file", "", "path too a file.")
	prefix := flag.String("prefix", "", "prefix of the local packages.")
	flag.Parse()

	path := *f
	if path == "" {
		flag.Usage()
		os.Exit(1)
	}

	return Args{
		path: path,
		prefix: *prefix,
	}
}

func loadFile(path string) (*token.FileSet, *ast.File, error) {
	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, path, nil, parser.ParseComments)
	if err != nil {
		return nil, nil, err
	}

	return fileSet, file, nil
}

func reorderImports(prefix string, srcDir string, file *ast.File) {
	for _, d := range file.Decls {
		genDecl, ok := d.(*ast.GenDecl)
		if !ok{
			decl := d.(*ast.FuncDecl)
			log.Println(decl.Body.List)
			log.Println(decl.Doc)
			for _, item := range decl.Body.List {
				log.Println(reflect.TypeOf(item).String())
				log.Println(item)
			}
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
			spec.Path.ValuePos=0

			if isStdLib(name, srcDir) {
				stdLibAst = append(stdLibAst, spec)
			} else if strings.HasPrefix(name, prefix) {
				localLibAst = append(localLibAst, spec)
			} else {
				otherLibAst = append(otherLibAst, spec)
			}
		}

		sort.Slice(stdLibAst, func(i, j int) bool{
			return stdLibAst[i].(*ast.ImportSpec).Path.Value < stdLibAst[i].(*ast.ImportSpec).Path.Value
		})
		stdLibAst = append(stdLibAst, &ast.ImportSpec{Path:&ast.BasicLit{}})

		sort.Slice(otherLibAst, func(i, j int) bool{
			return otherLibAst[i].(*ast.ImportSpec).Path.Value < otherLibAst[i].(*ast.ImportSpec).Path.Value
		})
		otherLibAst = append(otherLibAst, &ast.ImportSpec{Path:&ast.BasicLit{}})

		sort.Slice(localLibAst, func(i, j int) bool{
			return localLibAst[i].(*ast.ImportSpec).Path.Value < localLibAst[i].(*ast.ImportSpec).Path.Value
		})

		genDecl.Specs = append(append(stdLibAst, otherLibAst...), localLibAst...)
	}

}

func main() {
	args := getArgs()

	fileSet, file, err := loadFile(args.path)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	srcDir := filepath.Dir(args.path)
	cm := ast.NewCommentMap(fileSet, file, file.Comments)
	reorderImports(args.prefix, srcDir, file)
	set := token.NewFileSet()
	cmUp := ast.NewCommentMap(set, file, file.Comments)
	var rev = make(map[string]ast.Node)
	for keyNode, cgs := range cmUp {
		text := ""
		for _, cg := range cgs {
			text += cg.Text()
		}

		rev[text] = keyNode
	}
	for keyNode, cgs := range cm {
		text := ""
		for _, cg := range cgs {
			text += cg.Text()
		}
		cmUp.Update(rev[text], keyNode)
	}
	log.Println(cmUp)
	for _, cg := range file.Comments {
		for _, c := range cg.List {
			log.Println(c)
		}
	}
	file.Comments = cmUp.Comments()
	for _, cg := range file.Comments {
		for _, c := range cg.List {
			log.Println(c)
		}
	}

	printer.Fprint(os.Stdout, set, file)
}
