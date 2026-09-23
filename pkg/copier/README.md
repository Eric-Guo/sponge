## copyopt

`copier` is `github.com/jinzhu/copier`,  default option is add converters for time.Time <--> String.

### Example of use

```go.
package main

import (
    "fmt"
    "time"

    "github.com/Eric-Guo/sponge/pkg/copier"
)

type Model struct {
    ID        int64
    MyIP      string
    OrderID   uint32
    CreatedAt *time.Time
    UpdatedAt *time.Time
    DeletedAt *time.Time
}

type Reply struct {
    Id        int
    MyIp      string
    OrderId   int
    CreatedAt string
    UpdatedAt string
    DeletedAt string
}

func main() {
    now := time.Now()
    updated := now.Add(time.Hour)
    deleted := updated.Add(time.Hour)

    src := &Model{
        ID:        123,
        MyIP:      "127.0.0.1",
        OrderID:   888,
        CreatedAt: &now,
        UpdatedAt: &updated,
        DeletedAt: &deleted,
    }
    dst := &Reply{}
    err := copier.Copy(dst, src)
    if err != nil {
        panic(err)
    }

    fmt.Printf("%+v\n", dst)
}
```