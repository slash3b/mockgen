package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"log"
	"os"
	"strings"
)

func main() {
	path, err := os.Getwd()
	if err != nil {
		log.Fatalln(err.Error())
		return
	}

	fmt.Println("DEBUG:")
	fmt.Println("current directory is ", path)
	fmt.Println("all arguments", os.Args)
	fmt.Println("Package:", os.Getenv("GOPACKAGE"))
	fmt.Println("PWD:", os.Getenv("PWD"))
	fmt.Println("--------------------------")

	fileName := os.Getenv("GOFILE")

	fset := token.NewFileSet()

	astFile, err := parser.ParseFile(fset, fileName, nil, parser.AllErrors)
	if err != nil {
		panic(err)
	}

	var v Visitor

	ast.Walk(v, astFile)
}

type Item struct {
	Name string
	Type string
}

type Signature struct {
	Input  []Item
	Output []Item
}

type Visitor struct {
	Imports   []string
	Interface map[string][]Signature
}

func (v *Visitor) Visit(n ast.Node) ast.Visitor {
	if n == nil {
		return nil
	}

	switch t := n.(type) {
	case *ast.TypeSpec:
		iface, isInterface := t.Type.(*ast.InterfaceType)
		if isInterface {
			fmt.Println("found an interface!")
			for _, v := range iface.Methods.List {

				var buf bytes.Buffer

				fmt.Print("NAME:")
				for _, i := range v.Names {
					fmt.Print(i.Name, " ")
				}
				fmt.Println()

				// fixme: v.Names[0].Name is the name of the interface, not a name of the package
				fmt.Fprintf(&buf, fmt.Sprintf("package %s_test\n", v.Names[0].Name))
				fmt.Fprintf(&buf, "\n")

				fmt.Fprintf(&buf, "import (")
				fmt.Fprintf(&buf, `
  "testing"
  "github.com/stretchr/testify/mock"
				`)

				fmt.Fprintf(&buf, ")")
				fmt.Fprintf(&buf, "\n")

				formattedBytes, err := format.Source(buf.Bytes())
				if err != nil {
					panic(err)
				}

				// todo: fixFileName
				err = os.WriteFile(fmt.Sprintf("%s_test.go", "bar"), formattedBytes, 0644)
				if err != nil {
					panic(err)
				}
				fmt.Println("00000000000000000000000000000000000")

				switch o := v.Type.(type) {
				case *ast.FuncType:

					fmt.Println("Params (incoming):")
					for _, fld := range o.Params.List {
						fmt.Printf("Names: %s\n", strings.Join(names(fld.Names), ","))
						fmt.Printf("Type: %#v\n", fld.Type)
					}
					fmt.Println()

					fmt.Println("Return (outgoing):")
					for _, fld := range o.Results.List {
						fmt.Printf("Names: %s\n", strings.Join(names(fld.Names), ","))
						fmt.Printf("Type: %#v\n", fld.Type)
					}
					fmt.Println()

				default:
					fmt.Println("non-functype")
					fmt.Printf("%T %#[1]v\n", v.Type)
					selector := v.Type.(*ast.SelectorExpr)
					fmt.Printf("%#[1]v\n", selector.X)
					fmt.Printf("%#[1]v\n", selector.Sel)
				}
			}
			fmt.Println("-----------------------------")
		}
	}

	// sp := strings.Repeat(" ", int(v))
	// fmt.Printf("%s %T %#v \n", sp, n, n)

	return v
}

func make(iface *ast.InterfaceType) {
}

func names(ids []*ast.Ident) []string {
	res := []string{}
	for _, v := range ids {
		res = append(res, v.Name)
	}
	return res
}
