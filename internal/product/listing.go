```go
package product

import (
	"net/http"
	"github.com/go-playground/validator/v10"
	"github.com/gin-gonic/gin"
)

// Product represents the product model
type Product struct {
	Name        string  `json:"name" binding:"required,min=3,max=100"`
	Description string  `json:"description" binding:"required,min=10,max=500"`
	Price       float64 `json:"price" binding:"required,min=0"`
	Stock       int     `json:"stock" binding:"required,min=0"`
}

// CreateProduct handles the creation of a new product
func CreateProduct(c *gin.Context) {
	var product Product
	if err := c.ShouldBindJSON(&product); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, product)
}

// SetupRouter initializes the router
func SetupRouter() *gin.Engine {
	r := gin.Default()
	r.POST("/products", CreateProduct)
	return r
}
```
!!internal/product/listing_test.go!!
```go
package product

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestCreateProduct_ValidInput(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := SetupRouter()

	product := `{"name": "Product A", "description": "This is a valid product.", "price": 10.50, "stock": 100}`
	req, _ := http.NewRequest("POST", "/products", bytes.NewBufferString(product))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestCreateProduct_InvalidInput(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := SetupRouter()

	product := `{"name": "P", "description": "Short", "price": -10, "stock": -5}`
	req, _ := http.NewRequest("POST", "/products", bytes.NewBufferString(product))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
```