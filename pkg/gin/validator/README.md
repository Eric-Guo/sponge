## validator

`validator` is based on [validator](https://github.com/go-playground/validator) library. It provides request parameter validation for gin.

<br>

## Example of use

```go
package main

import (
    "net/http"

    "github.com/Eric-Guo/sponge/pkg/gin/validator"

    "github.com/gin-gonic/gin"
    "github.com/gin-gonic/gin/binding"
)

func main() {
	r := gin.Default()
	binding.Validator = validator.Init()
	
	r.POST("/create_user", CreateUser)
	
	r.Run(":8080")
}

type createUserRequest struct {
	Name  string `json:"name" form:"name" binding:"required"`
	Password string `json:"password" form:"password" binding:"required"`
	Age   int    `json:"age" form:"age" binding:"gte=0,lte=120"`
	Email string `json:"email" form:"email" binding:"email"`
}

func CreateUser(c *gin.Context) {
	form := &createUserRequest{}
	err := c.ShouldBindJSON(form)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"msg": "ok"})
}
```
