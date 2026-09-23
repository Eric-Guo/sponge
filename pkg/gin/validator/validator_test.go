package validator

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/stretchr/testify/assert"

	"github.com/Eric-Guo/sponge/pkg/utils"
)

func runValidatorHTTPServer() string {
	serverAddr, requestAddr := utils.GetLocalHTTPAddrPairs()

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	binding.Validator = Init()

	r.POST("/hello", createHello)
	r.DELETE("/hello", deleteHello)
	r.PUT("/hello", updateHello)
	r.GET("/hello", getHello)
	r.GET("/hello/:id", getHello)
	r.GET("/hellos", getHellos)

	go func() {
		err := r.Run(serverAddr)
		if err != nil {
			panic(err)
		}
	}()

	time.Sleep(time.Millisecond * 200)
	return requestAddr
}

var (
	helloStr = "hello world"
	paramErr = "params is invalid"

	wantHello    = fmt.Sprintf(`"%s"`, helloStr)
	wantParamErr = fmt.Sprintf(`"%s"`, paramErr)
)

type postForm struct {
	Name  string `json:"name" form:"name" binding:"required"`
	Age   int    `json:"age" form:"age" binding:"gte=0,lte=150"`
	Email string `json:"email" form:"email" binding:"email,required"`
}

func createHello(c *gin.Context) {
	form := &postForm{}
	err := c.ShouldBindJSON(form)
	if err != nil {
		fmt.Println(err)
		c.JSON(http.StatusBadRequest, paramErr)
		return
	}
	fmt.Printf("%+v\n", form)
	c.JSON(http.StatusOK, helloStr)
}

type deleteForm struct {
	IDS []uint64 `form:"ids" binding:"required,min=1"`
}

func deleteHello(c *gin.Context) {
	form := &deleteForm{}
	err := c.ShouldBindJSON(form)
	if err != nil {
		fmt.Println(err)
		c.JSON(http.StatusBadRequest, paramErr)
		return
	}
	fmt.Printf("%+v\n", form)
	c.JSON(http.StatusOK, helloStr)
}

type updateForm struct {
	ID    uint64 `json:"id" form:"id" binding:"required,gt=0"`
	Age   int    `json:"age" form:"age" binding:"gte=0,lte=150"`
	Email string `json:"email" form:"email" binding:"email,required"`
}

func updateHello(c *gin.Context) {
	form := &updateForm{}
	err := c.ShouldBindJSON(form)
	if err != nil {
		fmt.Println(err)
		c.JSON(http.StatusBadRequest, paramErr)
		return
	}
	fmt.Printf("%+v\n", form)
	c.JSON(http.StatusOK, helloStr)
}

type getForm struct {
	ID uint64 `form:"id" binding:"gt=0"`
}

func getHello(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 0)
	form := &getForm{ID: id}
	err := c.ShouldBindQuery(form)
	if err != nil {
		fmt.Println(err)
		c.JSON(http.StatusBadRequest, paramErr)
		return
	}
	fmt.Printf("%+v\n", form)
	c.JSON(http.StatusOK, helloStr)
}

type getsForm struct {
	Page  int    `form:"page" binding:"gte=0"`
	Limit int    `form:"limit" binding:"gte=1"`
	Sort  string `form:"sort" binding:"required,min=2"`
}

func getHellos(c *gin.Context) {
	form := &getsForm{}
	err := c.ShouldBindQuery(form)
	if err != nil {
		fmt.Println(err)
		c.JSON(http.StatusBadRequest, paramErr)
		return
	}
	fmt.Printf("%+v\n", form)
	c.JSON(http.StatusOK, helloStr)
}

// ------------------------------------------------------------------------------------------
// http client
// ------------------------------------------------------------------------------------------

func TestPostValidate(t *testing.T) {
	requestAddr := runValidatorHTTPServer()

	t.Run("success", func(t *testing.T) {
		got, err := do(http.MethodPost, requestAddr+"/hello", &postForm{
			Name:  "foo",
			Age:   10,
			Email: "bar@gmail.com",
		})
		if err != nil {
			t.Error(err)
			return
		}
		if string(got) != wantHello {
			t.Errorf("got: %s, want: %s", got, wantHello)
		}
	})

	t.Run("missing field error", func(t *testing.T) {
		got, err := do(http.MethodPost, requestAddr+"/hello", &postForm{
			Age:   10,
			Email: "bar@gmail.com",
		})
		if err != nil {
			t.Error(err)
			return
		}
		if string(got) != wantParamErr {
			t.Errorf("got: %s, want: %s", got, wantParamErr)
		}
	})

	t.Run("field range  error", func(t *testing.T) {
		got, err := do(http.MethodPost, requestAddr+"/hello", &postForm{
			Name:  "foo",
			Age:   -1,
			Email: "bar@gmail.com",
		})
		if err != nil {
			t.Error(err)
			return
		}
		if string(got) != wantParamErr {
			t.Errorf("got: %s, want: %s", got, wantParamErr)
		}
	})

	t.Run("email error", func(t *testing.T) {
		got, err := do(http.MethodPost, requestAddr+"/hello", &postForm{
			Name:  "foo",
			Age:   10,
			Email: "bar",
		})
		if err != nil {
			t.Error(err)
			return
		}
		if string(got) != wantParamErr {
			t.Errorf("got: %s, want: %s", got, wantParamErr)
		}
	})
}

// ------------------------------------------------------------------------------------------

func TestDeleteValidate(t *testing.T) {
	requestAddr := runValidatorHTTPServer()

	t.Run("success", func(t *testing.T) {
		got, err := do(http.MethodDelete, requestAddr+"/hello", &deleteForm{
			IDS: []uint64{1, 2, 3},
		})
		if err != nil {
			t.Error(err)
			return
		}
		if string(got) != wantHello {
			t.Errorf("got: %s, want: %s", got, wantHello)
		}
	})

	t.Run("missing field error", func(t *testing.T) {
		got, err := do(http.MethodDelete, requestAddr+"/hello", nil)
		if err != nil {
			t.Error(err)
			return
		}
		if string(got) != wantParamErr {
			t.Errorf("got: %s, want: %s", got, wantParamErr)
		}
	})

	t.Run("ids  error", func(t *testing.T) {
		got, err := do(http.MethodDelete, requestAddr+"/hello", &deleteForm{IDS: []uint64{}})
		if err != nil {
			t.Error(err)
			return
		}
		if string(got) != wantParamErr {
			t.Errorf("got: %s, want: %s", got, wantParamErr)
		}
	})
}

// -------------------------------------------------------------------------------------------

func TestPutValidate(t *testing.T) {
	requestAddr := runValidatorHTTPServer()

	t.Run("success", func(t *testing.T) {
		got, err := do(http.MethodPut, requestAddr+"/hello", &updateForm{
			ID:    100,
			Age:   10,
			Email: "bar@gmail.com",
		})
		if err != nil {
			t.Error(err)
			return
		}
		if string(got) != wantHello {
			t.Errorf("got: %s, want: %s", got, wantHello)
		}
	})

	t.Run("missing field error", func(t *testing.T) {
		got, err := do(http.MethodPut, requestAddr+"/hello", &updateForm{
			Age:   10,
			Email: "bar@gmail.com",
		})
		if err != nil {
			t.Error(err)
			return
		}
		if string(got) != wantParamErr {
			t.Errorf("got: %s, want: %s", got, wantParamErr)
		}
	})

	t.Run("email error", func(t *testing.T) {
		got, err := do(http.MethodPut, requestAddr+"/hello", &updateForm{
			ID:    101,
			Age:   10,
			Email: "bar",
		})
		if err != nil {
			t.Error(err)
			return
		}
		if string(got) != wantParamErr {
			t.Errorf("got: %s, want: %s", got, wantParamErr)
		}
	})
}

// -------------------------------------------------------------------------------------------

func TestGetValidate(t *testing.T) {
	requestAddr := runValidatorHTTPServer()

	t.Run("success", func(t *testing.T) {
		got, err := do(http.MethodGet, requestAddr+"/hello?id=100", nil)
		if err != nil {
			t.Error(err)
			return
		}
		if string(got) != wantHello {
			t.Errorf("got: %s, want: %s", got, wantHello)
		}
	})

	t.Run("success2", func(t *testing.T) {
		got, err := do(http.MethodGet, requestAddr+"/hello/101", nil)
		if err != nil {
			t.Error(err)
			return
		}
		if string(got) != wantHello {
			t.Errorf("got: %s, want: %s", got, wantHello)
		}
	})

	t.Run("miss id error", func(t *testing.T) {
		got, err := do(http.MethodGet, requestAddr+"/hello", nil)
		if err != nil {
			t.Error(err)
			return
		}
		if string(got) != wantParamErr {
			t.Errorf("got: %s, want: %s", got, wantParamErr)
		}
	})
}

// -------------------------------------------------------------------------------------------

func TestGetsValidate(t *testing.T) {
	requestAddr := runValidatorHTTPServer()

	t.Run("success", func(t *testing.T) {
		got, err := do(http.MethodGet, requestAddr+"/hellos?page=0&limit=10&sort=-id", nil)
		if err != nil {
			t.Error(err)
			return
		}
		if string(got) != wantHello {
			t.Errorf("got: %s, want: %s", got, wantHello)
		}
	})

	t.Run("missing field error", func(t *testing.T) {
		got, err := do(http.MethodGet, requestAddr+"/hellos?page=0&limit=10", nil)
		if err != nil {
			t.Error(err)
			return
		}
		if string(got) != wantParamErr {
			t.Errorf("got: %s, want: %s", got, wantParamErr)
		}
	})

	t.Run("size error", func(t *testing.T) {
		got, err := do(http.MethodGet, requestAddr+"/hellos?page=0&limit=0&sort=-id", nil)
		if err != nil {
			t.Error(err)
			return
		}
		if string(got) != wantParamErr {
			t.Errorf("got: %s, want: %s", got, wantParamErr)
		}
	})
}

// ------------------------------------------------------------------------------------------

func do(method string, url string, body interface{}) ([]byte, error) {
	var reader io.Reader
	if body == nil {
		reader = nil
	} else {
		v, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(v)
	}

	method = strings.ToUpper(method)
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		req, err := http.NewRequest(method, url, reader)
		if err != nil {
			return nil, err
		}
		req.Header.Add("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		return io.ReadAll(resp.Body)

	case http.MethodGet:
		resp, err := http.Get(url)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		return io.ReadAll(resp.Body)

	default:
		return nil, errors.New("unknown method")
	}
}

// ------------------------------------------------------------------------------------------

func Test_CustomValidator_ValidateStruct(t *testing.T) {
	type User struct {
		Name string `binding:"required"`
		Age  int    `binding:"gte=18"`
	}

	type UserList1 struct {
		Users []User `binding:"required,dive"`
	}

	type UserList2 struct {
		Users []*User `binding:"required,dive"`
	}

	validator := Init()

	user := &User{Name: "John", Age: 10}
	if err := validator.ValidateStruct(user); err != nil {
		assert.NotNil(t, err)
		t.Log(err)
	}

	var u = &User{Name: "John", Age: 11}
	if err := validator.ValidateStruct(&u); err != nil {
		assert.NotNil(t, err)
		t.Log(err)
	}

	users := []User{{Name: "Alice", Age: 25}, {Name: "Bob", Age: 17}}
	if err := validator.ValidateStruct(users); err != nil {
		assert.NotNil(t, err)
		t.Log(err)
	}

	userList := UserList1{}
	if err := validator.ValidateStruct(&userList); err != nil {
		assert.NotNil(t, err)
		t.Log(err)
	}

	userList1 := UserList1{
		Users: []User{{Name: "Charlie", Age: 10}, {Name: "", Age: 30}},
	}
	if err := validator.ValidateStruct(&userList1); err != nil {
		assert.NotNil(t, err)
		t.Log(err)
	}

	userList2 := UserList2{
		Users: []*User{{Name: "Charlie", Age: 30}, {Name: "", Age: 40}},
	}
	if err := validator.ValidateStruct(&userList2); err != nil {
		assert.NotNil(t, err)
		t.Log(err)
	}
}

func Benchmark_CustomValidator_ValidateStruct(b *testing.B) {
	type User struct {
		Name string `binding:"required"`
		Age  int    `binding:"gte=18"`
	}

	type UserList1 struct {
		Users []User `binding:"required,dive"` // 验证指针切片
	}

	type UserList2 struct {
		Users []*User `binding:"required,dive"` // 验证指针切片
	}

	validator := Init()

	b.Run("User struct", func(b *testing.B) {
		user := User{Name: "John", Age: 10}
		for i := 0; i < b.N; i++ {
			_ = validator.ValidateStruct(user)
		}
	})

	b.Run("User struct pointer", func(b *testing.B) {
		user := &User{Name: "John", Age: 10}
		for i := 0; i < b.N; i++ {
			_ = validator.ValidateStruct(user)
		}
	})

	b.Run("User struct pointer pointer", func(b *testing.B) {
		var u = &User{Name: "John", Age: 11}
		for i := 0; i < b.N; i++ {
			_ = validator.ValidateStruct(&u)
		}
	})

	b.Run("User slice", func(b *testing.B) {
		users := []User{{Name: "Alice", Age: 25}, {Name: "Bob", Age: 17}}
		for i := 0; i < b.N; i++ {
			_ = validator.ValidateStruct(users)
		}
	})

	b.Run("UserList slice struct", func(b *testing.B) {
		userList1 := UserList1{
			Users: []User{{Name: "Charlie", Age: 10}, {Name: "", Age: 30}},
		}
		for i := 0; i < b.N; i++ {
			_ = validator.ValidateStruct(&userList1)
		}
	})

	b.Run("UserList slice struct pointer", func(b *testing.B) {
		userList2 := UserList2{
			Users: []*User{{Name: "Charlie", Age: 30}, {Name: "", Age: 40}},
		}
		for i := 0; i < b.N; i++ {
			_ = validator.ValidateStruct(&userList2)
		}
	})
}
