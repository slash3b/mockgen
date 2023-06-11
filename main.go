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

var fset = token.NewFileSet()

func main() {
	path, err := os.Getwd()
	if err != nil {
		log.Fatalln(err.Error())
		return
	}

	fmt.Println("\n\nDebug start: -----------------------------")
	fmt.Println("current directory is ", path)
	fmt.Println("all arguments", os.Args)
	fmt.Println("Package:", os.Getenv("GOPACKAGE"))
	fmt.Println("PWD:", os.Getenv("PWD"))
	fmt.Println("Debug end: -------------------------------")

	fileName := os.Getenv("GOFILE")

	astFile, err := parser.ParseFile(fset, fileName, nil, parser.AllErrors)
	if err != nil {
		panic(err)
	}

	var v Visitor
	v.Interfaces = make(map[string][]Method)

	ast.Walk(v, astFile)

	var g Generator
	g.toFile(fileName, v)
}

type Item struct {
	Name string
	Type string
}

type Method struct {
	Name   string
	Input  []Item
	Output []Item
}

type Visitor struct {
	Imports    []string
	Interfaces map[string][]Method
}

func (v Visitor) Visit(n ast.Node) ast.Visitor {
	if n == nil {
		return nil
	}

	switch t := n.(type) {
	case *ast.TypeSpec:
		iface, isInterface := t.Type.(*ast.InterfaceType)
		if isInterface {
			v.make(t.Name.Name, iface)
		}
	}

	return v
}

func (v *Visitor) make(iname string, iface *ast.InterfaceType) {
	for _, field := range iface.Methods.List {
		switch method := field.Type.(type) {
		case *ast.FuncType: // which is a method signature, e.g. Foo(x int) string
			// ast.Print(fset, field)

			if len(field.Names) != 1 {
				panic(fmt.Sprintf("unexpected amount of field.Names %d", len(field.Names)))
			}

			methodName := field.Names[0].Name

			input := make([]Item, 0)
			for _, param := range method.Params.List {
				paramType, ok := param.Type.(*ast.Ident)
				if !ok {
					panic(fmt.Sprintf("unexpected AST type %T, *ast.Ident expected", param.Type))
				}

				if len(param.Names) == 0 {
					input = append(input, Item{Type: paramType.Name})
					continue
				}

				for _, n := range param.Names {
					input = append(input, Item{Name: n.Name, Type: paramType.Name})
				}
			}

			output := make([]Item, 0)
			for _, param := range method.Results.List {
				paramType, ok := param.Type.(*ast.Ident)
				if !ok {
					panic(fmt.Sprintf("unexpected AST type %T, *ast.Ident expected", param.Type))
				}

				if len(param.Names) == 0 {
					output = append(output, Item{Type: paramType.Name})
					continue
				}

				for _, n := range param.Names {
					output = append(output, Item{Name: n.Name, Type: paramType.Name})
				}
			}

			// now store it!
			methods, ok := v.Interfaces[iname]
			if !ok {
				methods = make([]Method, 0)
			}
			methods = append(methods, Method{Name: methodName, Input: input, Output: output})
			v.Interfaces[iname] = methods
		default:
			/*
				fmt.Println("non-functype")
				fmt.Printf("%T %#[1]v\n", v.Type)
				selector := v.Type.(*ast.SelectorExpr)
				fmt.Printf("%#[1]v\n", selector.X)
				fmt.Printf("%#[1]v\n", selector.Sel)
			*/
		}
	}
}

type Generator struct{}

func (g *Generator) toFile(fileName string, v Visitor) {
	fileNameParts := strings.Split(fileName, ".")
	testFileName := fileNameParts[0] + "_test.go"

	var buf bytes.Buffer

	fmt.Fprintf(&buf, "//Auto-generated. Do Not Edit!")
	fmt.Fprintf(&buf, "\n")
	fmt.Fprintf(&buf, "package bar")
	fmt.Fprintf(&buf, "\n")

	fmt.Fprintf(&buf, "import (")
	fmt.Fprintf(&buf, `
		  "github.com/stretchr/testify/mock"
)`)

	fmt.Fprintf(&buf, "\n")

	// add method here
	for iface, methods := range v.Interfaces {
		// mock struct generation
		fmt.Fprintf(&buf, "\n")
		fmt.Fprintf(&buf, fmt.Sprintf(`var _ %s = (*%[1]sMock)(nil)`, iface))
		fmt.Fprintf(&buf, "\n")

		mockName := iface + "Mock"

		fmt.Fprintf(&buf, fmt.Sprintf(`type %sMock struct {`, iface))
		fmt.Fprintf(&buf, `
mock.Mock
		`)

		fmt.Fprintf(&buf, `}
		`)

		// methods

		receiverName := strings.ToLower(string(iface[0]) + "m")

		for _, m := range methods {
			fmt.Fprintf(&buf, "\n")

			incoming := []string{}
			for _, inc := range m.Input {
				incoming = append(incoming, fmt.Sprintf(`%s %s`, inc.Name, inc.Type))
			}

			outgoing := []string{}
			for _, out := range m.Output {
				outgoing = append(outgoing, fmt.Sprintf(`%s %s`, out.Name, out.Type))
			}

			// method signature
			fmt.Fprintf(&buf, fmt.Sprintf(`func(%s *%s) %s`, receiverName, mockName, m.Name))
			fmt.Fprintf(&buf, fmt.Sprintf(`(%s) (%s) {`, strings.Join(incoming, ","), strings.Join(outgoing, ",")))
			fmt.Fprintf(&buf, "\n")
			// method body ARGS
			fmt.Fprintf(&buf, fmt.Sprintf(`args := %s.Called(`, receiverName))

			{
				incoming := []string{}
				for _, inc := range m.Input {
					incoming = append(incoming, inc.Name)
				}

				fmt.Fprintf(&buf, strings.Join(incoming, `,`))
			}

			fmt.Fprintf(&buf, `)`)
			fmt.Fprintf(&buf, "\n")

			// method body RETURN

			fmt.Fprintf(&buf, "\n")

			{
				fmt.Fprintf(&buf, "return ")

				outgoing := []string{}
				for i, out := range m.Output {
					// todo all types
					// for now we'll just use Get
					if out.Type == "error" {
						outgoing = append(outgoing, fmt.Sprintf(`args.Error(%d)`, i))
					} else {
						switch out.Type {
						case "int":
							outgoing = append(outgoing, fmt.Sprintf(`args.Int(%d)`, i))
						case "string":
							outgoing = append(outgoing, fmt.Sprintf(`args.String(%d)`, i))
						default:
							outgoing = append(outgoing, fmt.Sprintf(`args.Get(%d).(*%s)`, i, out.Type))
						}
					}
				}

				fmt.Fprintf(&buf, strings.Join(outgoing, ","))
			}

			fmt.Fprintf(&buf, "\n")

			fmt.Fprintf(&buf, "}")
			fmt.Fprintf(&buf, "\n")
		}
	}

	fmt.Println(buf.String())
	formattedBytes, err := format.Source(buf.Bytes())
	if err != nil {
		panic(err)
	}

	err = os.WriteFile(testFileName, formattedBytes, 0644)
	if err != nil {
		panic(err)
	}

	fmt.Println("Done!")
}
