## frontend

`frontend` is a library for serving static files in a Gin web application. It supports local static files and embedding static files in binary.

<br>

### Example of use

### Local static files

```go
package main

import (
	"github.com/gin-gonic/gin"
	"github.com/Eric-Guo/sponge/pkg/gin/frontend"
)

func main() {
	r := gin.Default()
	f := frontend.New("dist",
		// frontend.With404ToHome(),
	)
	err := f.SetRouter(r)
	if err != nil {
		panic(err)
	}
	err = r.Run(":8080")
	panic(err)
}
```

#### Embedding static files in binary

```go
package main

import (
	"embed"
	"github.com/gin-gonic/gin"
	"github.com/Eric-Guo/sponge/pkg/gin/frontend"
)

//go:embed dist
var staticFS embed.FS

func main() {
	r := gin.Default()
	f := frontend.New("dist",
		frontend.WithEmbedFS(staticFS),
		//frontend.WithHandleContent(func(content []byte) []byte {
		//	return bytes.ReplaceAll(content, []byte("http://localhost:24631/api/v1"), []byte("http://192.168.3.37:24631/api/v1"))
		//}, "appConfig.js"),
		// frontend.With404ToHome(),
	)
	err := f.SetRouter(r)
	if err != nil {
		panic(err)
	}
	err = r.Run(":8080")
	panic(err)
}
```
