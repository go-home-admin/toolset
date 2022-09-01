## 這是一個專門解析golang結構的包

如何使用

````
scan := "./"
for dir, fileParsers := range parser.NewGoParserForDir(scan) {
	for _, fileParser := range fileParsers {
		for _, goType := range fileParser.Types {
			for _, attr := range goType.Attrs {
				if attr.HasTag("inject") {
					// 是否擁有某個tag
				}
			}

			// 檢查struct註解
			if goType.Doc.HasAnnotation("@Bean") {
				
			}
		}
	}
}
````