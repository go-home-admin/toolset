package parser

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"log"
	"strings"
)

func NewAst(path string) map[string][]GoFileParser {
	got := make(map[string][]GoFileParser)
	for _, dir := range GetChildrenDir(path) {
		arr := make([]GoFileParser, 0)
		for _, file := range dir.GetFiles(".go") {
			if strings.Index(file.Name(), "_gen.go") != -1 {
				continue
			}
			gof := getAstGoFileParser(file.Path)
			arr = append(arr, gof)
		}
		got[dir.Path] = arr
	}

	return got
}

func getAstGoFileParser(fileName string) GoFileParser {
	got := GoFileParser{
		PackageName: "",
		PackageDoc:  "",
		Imports:     make(map[string]string),
		Types:       make(map[string]GoType),
		Funds:       make(map[string]GoFunc),
	}
	// 1. 读取文件内容
	fileContent, err := ioutil.ReadFile(fileName)
	if err != nil {
		log.Fatalf("Error reading file: %v", err)
	}
	// 2. 创建一个token.FileSet（用于保存位置信息）
	fset := token.NewFileSet()

	// 3. 解析Go源代码
	file, err := parser.ParseFile(fset, fileName, fileContent, parser.ParseComments)
	if err != nil {
		log.Fatal(err)
	}

	got.PackageName = file.Name.Name

	// 4. 遍历AST并处理函数和变量
	for _, decl := range file.Decls {
		switch decl := decl.(type) {
		case *ast.FuncDecl: // 处理函数声明
		case *ast.GenDecl:
			// 处理通用声明（变量声明）
			switch decl.Tok {
			case token.VAR:
			case token.IMPORT: // import
				for _, spec := range decl.Specs {
					Spec := spec.(*ast.ImportSpec)
					if Spec.Name == nil {
						arr := strings.Split(Spec.Path.Value, "/")
						got.Imports[strings.Trim(arr[len(arr)-1], "\"")] = strings.Trim(Spec.Path.Value, "\"")
					} else {
						got.Imports[Spec.Name.Name] = strings.Trim(Spec.Path.Value, "\"")
					}
				}
			case token.CONST: // const 变量声明
			case token.TYPE: // type 变量声明、结构
				var doc string
				if decl.Doc != nil {
					for _, d := range decl.Doc.List {
						doc = doc + d.Text + "\n"
					}
				} else {
					// 没有注释, 跳过
					continue
				}
				for _, spec := range decl.Specs {
					Spec := spec.(*ast.TypeSpec)
					TypeSpec, ok := Spec.Type.(*ast.StructType)
					if ok {
						t := GoType{
							Doc:          GoDoc(doc),
							Name:         Spec.Name.Name,
							Attrs:        map[string]GoTypeAttr{},
							AttrsSort:    make([]string, 0),
							GoFileParser: &got,
						}

						for _, field := range TypeSpec.Fields.List {
							attr := GoTypeAttr{GoFileParser: &got}
							for _, name := range field.Names {
								attr.Name = name.Name
							}

							if expr, ok := field.Type.(*ast.SelectorExpr); ok {
								if attr.Name == "" {
									attr.Name = strings.Trim(expr.Sel.Name, ".")
								}
								attr.TypeAlias = expr.X.(*ast.Ident).Name
								attr.TypeName = attr.TypeAlias + "." + expr.Sel.Name
								attr.TypeImport = got.Imports[attr.TypeAlias]
							} else if expr, ok := field.Type.(*ast.Ident); ok {
								if attr.Name == "" {
									attr.Name = expr.Name
								}
								attr.TypeName = expr.Name
								attr.InPackage = true
							} else if expr, ok := field.Type.(*ast.MapType); ok {
								if _, ok := expr.Key.(*ast.SelectorExpr); ok {
									attr.TypeName = "map todo"
									attr.InPackage = true
								} else {
									attr.TypeName = expr.Key.(*ast.Ident).Name
									attr.InPackage = true
								}
							} else if _, ok := field.Type.(*ast.ArrayType); ok {
								attr.TypeName = "[]todo"
							} else if _, ok := field.Type.(*ast.InterfaceType); ok {
								attr.TypeName = "todointerface"
							} else if expr, ok := field.Type.(*ast.StarExpr); ok {
								if expr2, ok := expr.X.(*ast.SelectorExpr); ok {
									if attr.Name == "" {
										attr.Name = strings.Trim(expr2.Sel.Name, ".")
									}
									attr.TypeAlias = expr2.X.(*ast.Ident).Name
									attr.TypeName = "*" + attr.TypeAlias + "." + expr2.Sel.Name
									attr.TypeImport = got.Imports[attr.TypeAlias]
								} else if expr2, ok := expr.X.(*ast.Ident); ok {
									if attr.Name == "" {
										attr.Name = expr2.Name
									}
									attr.TypeAlias = expr2.Name
									attr.TypeName = "*" + expr2.Name
									attr.InPackage = true
								} else {

								}
							} else {

							}

							// 解析 go tag
							attr.Tag = make(map[string]TagDoc)
							if field.Tag != nil {
								for _, tagStrArr := range getArrGoTag(field.Tag.Value) {
									td := tagStrArr[1]
									if len(td) > 1 {
										attr.Tag[tagStrArr[0]] = TagDoc(td[1 : len(td)-1])
									}
								}
							}
							if attr.TypeName == "" {
								continue
							}
							t.Attrs[attr.Name] = attr
							t.AttrsSort = append(t.AttrsSort, attr.Name)
						}

						got.Types[Spec.Name.Name] = t
					}
				}
			default:

			}
		default:

		}
	}
	return got
}
