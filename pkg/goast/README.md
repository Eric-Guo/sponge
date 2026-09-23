## goast

`goast` is a library for parsing Go code and extracting information, it supports merging two Go files into one.

## Example of use

### Parse Go code and extract information

```go
package main

import (
	"fmt"
	"github.com/Eric-Guo/sponge/pkg/goast"
)

func main() {
	src := []byte(`package main

import (
    "fmt"
)

func main() {
    fmt.Println("Hello, world!")
}
`)

	// Case 1: Parse Go code and extract information
	{
		astInfos, err := goast.ParseGoCode("main.go", src)
	}

	// Case 2: Parse file and extract information
	{
		astInfos, err := goast.ParseFile("main.go")
	}
}
```

### Merge two Go files into one

```go
package main

import (
	"fmt"
	"github.com/Eric-Guo/sponge/pkg/goast"
)

func main() {
	const (
		srcFile = "data/src.go.code"
		genFile = "data/gen.go.code"
	)

	// Case 1: without covering the same function
	{
		codeAst, err := goast.MergeGoFile(srcFile, genFile)
		fmt.Println(codeAst.Code)
	}

	// Case 2: with covering the same function
	{
		codeAst, err := goast.MergeGoFile(srcFile, genFile, goast.WithCoverSameFunc())
		fmt.Println(codeAst.Code)
	}
}
```
