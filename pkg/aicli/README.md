## aicli

`aicli` is AI assistant client library for Go, support ChatGPT, DeepSeek and Gemini.

<br>

## Example of use

### ChatGPT

```go
package main

import (
    "context"
    "fmt"
    "github.com/Eric-Guo/sponge/pkg/aicli/chatgpt"
)

func main() {
    var apiKey = "sk-xxxxxx"
    client, _ := chatgpt.NewClient(apiKey) // you can use set client options, e.g. WithModel(ModelGPT4o)

    // case 1: default
    content, _ := client.Send(context.Background(), "Who are you?)
    fmt.Println(content)

    // case 2: stream
    answer := client.SendStream(context.Background(), "Which model did you use to answer the question?")
    for content := range answer.Content {
        fmt.Printf(content)
    }
    if answer.Err != nil {
        panic(answer.Err)
    }
}
```

`NewClient` function supports setting options, such as `WithModel("gpt-4o")`, `WithEnableContext()` and other parameter settings.

<br>

### DeepSeek

```go
package main

import (
    "context"
    "fmt"
    "github.com/Eric-Guo/sponge/pkg/aicli/deepseek"
)

func main() {
    var apiKey = "sk-xxxxxx"
    client, _ := deepseek.NewClient(apiKey) // you can use set client options, e.g. WithModel(ModelDeepSeekReasoner)

    // case 1: default
    content, _ := client.Send(context.Background(), "Who are you?)
    fmt.Println(content)

    // case 2: stream
    answer := client.SendStream(context.Background(), "Which model did you use to answer the question?")
    for content := range answer.Content {
        fmt.Printf(content)
    }
    if answer.Err != nil {
        panic(answer.Err)
    }
}
```

`NewClient` function supports setting options, such as `WithModel("deepseek-reasoner")`, `WithEnableContext()` and other parameter settings.

<br>

### Gemini

```go
package main

import (
    "context"
    "fmt"
    "github.com/Eric-Guo/sponge/pkg/aicli/gemini"
)

func main() {
    var apiKey = "sk-xxxxxx"
    client, _ := gemini.NewClient(apiKey) // you can use set client options, e.g. WithModel(Model20FlashThinking)

    // case 1: default
    content, _ := client.Send(context.Background(), "Who are you?")
    fmt.Println(content)

    // case 2: stream
    answer := client.SendStream(context.Background(), "Which model did you use to answer the question?")
    for content := range answer.Content {
        fmt.Printf(content)
    }
    if answer.Err != nil {
        panic(answer.Err)
    }
}
```

`NewClient` function supports setting options, such as `WithModel("gemini-2.0-flash-thinking-exp")`, `WithEnableContext()` and other parameter settings.

`Send` and `SendStream` functions support file upload, example: `Send(ctx, "prompt", file)`.

Supported file types are as follows:
```
text/plain
application/pdf
audio/mpeg
audio/mp3
audio/wav
image/png
image/jpeg
video/mov
video/mpeg
video/mp4
video/mpg
video/avi
video/wmv
video/mpegps
video/flv
```
