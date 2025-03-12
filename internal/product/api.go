```go
package product

import (
	"encoding/json"
	"net/http"
	"strconv"
	"sync"
)

type Product struct {
	ID          int     `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	Stock       int     `json:"stock"`
}

var (
	products = make(map[int]Product)
	nextID   = 1
	mu       sync.Mutex
)

func CreateProduct(w http.ResponseWriter, r *http.Request) {
	var p Product
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil || p.Name == "" || p.Price <= 0 || p.Stock < 0 {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	mu.Lock()
	p.ID = nextID
	nextID++
	products[p.ID] = p
	mu.Unlock()

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(p)
}

func GetProduct(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	mu.Lock()
	product, exists := products[id]
	mu.Unlock()

	if !exists {
		http.Error(w, "Product not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(product)
}

func UpdateProduct(w http.ResponseWriter, r *http.Request) {
	var p Product
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil || p.ID <= 0 || p.Name == "" || p.Price <= 0 || p.Stock < 0 {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	mu.Lock()
	if _, exists := products[p.ID]; !exists {
		mu.Unlock()
		http.Error(w, "Product not found", http.StatusNotFound)
		return
	}
	products[p.ID] = p
	mu.Unlock()

	json.NewEncoder(w).Encode(p)
}

func DeleteProduct(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	mu.Lock()
	defer mu.Unlock()
	if _, exists := products[id]; !exists {
		http.Error(w, "Product not found", http.StatusNotFound)
		return
	}
	delete(products, id)
	w.WriteHeader(http.StatusNoContent)
}
```

!!internal/product/api_test.go!!
```go
package product

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateProduct(t *testing.T) {
	product := Product{Name: "Test Product", Description: "Test Description", Price: 10.99, Stock: 100}
	body, _ := json.Marshal(product)

	req, err := http.NewRequest("POST", "/products", bytes.NewBuffer(body))
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(CreateProduct)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusCreated)
	}

	var createdProduct Product
	json.NewDecoder(rr.Body).Decode(&createdProduct)
	if createdProduct.Name != product.Name {
		t.Errorf("handler returned unexpected body: got %v want %v", createdProduct.Name, product.Name)
	}
}

func TestGetProduct(t *testing.T) {
	product := Product{Name: "Test Product", Description: "Test Description", Price: 10.99, Stock: 100}
	body, _ := json.Marshal(product)

	// Create a product first
	reqCreate, _ := http.NewRequest("POST", "/products", bytes.NewBuffer(body))
	rrCreate := httptest.NewRecorder()
	handlerCreate := http.HandlerFunc(CreateProduct)
	handlerCreate.ServeHTTP(rrCreate, reqCreate)

	var createdProduct Product
	json.NewDecoder(rrCreate.Body).Decode(&createdProduct)

	// Now get the created product
	req, _ := http.NewRequest("GET", "/products?id="+strconv.Itoa(createdProduct.ID), nil)
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(GetProduct)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var fetchedProduct Product
	json.NewDecoder(rr.Body).Decode(&fetchedProduct)
	if fetchedProduct.ID != createdProduct.ID {
		t.Errorf("handler returned unexpected body: got %v want %v", fetchedProduct.ID, createdProduct.ID)
	}
}

func TestUpdateProduct(t *testing.T) {
	product := Product{Name: "Test Product", Description: "Test Description", Price: 10.99, Stock: 100}
	body, _ := json.Marshal(product)

	reqCreate, _ := http.NewRequest("POST", "/products", bytes.NewBuffer(body))
	rrCreate := httptest.NewRecorder()
	handlerCreate := http.HandlerFunc(CreateProduct)
	handlerCreate.ServeHTTP(rrCreate, reqCreate)

	var createdProduct Product
	json.NewDecoder(rrCreate.Body).Decode(&createdProduct)

	// Update the product
	createdProduct.Price = 12.99
	bodyUpdate, _ := json.Marshal(createdProduct)
	reqUpdate, _ := http.NewRequest("PUT", "/products", bytes.NewBuffer(bodyUpdate))
	rrUpdate := httptest.NewRecorder()
	handlerUpdate := http.HandlerFunc(UpdateProduct)
	handlerUpdate.ServeHTTP(rrUpdate, reqUpdate)

	if status := rrUpdate.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var updatedProduct Product
	json.NewDecoder(rrUpdate.Body).Decode(&updatedProduct)
	if updatedProduct.Price != 12.99 {
		t.Errorf("handler returned unexpected body: got %v want %v", updatedProduct.Price, 12.99)
	}
}

func TestDeleteProduct(t *testing.T) {
	product := Product{Name: "Test Product", Description: "Test Description", Price: 10.99, Stock: 100}
	body, _ := json.Marshal(product)

	reqCreate, _ := http.NewRequest("POST", "/products", bytes.NewBuffer(body))
	rrCreate := httptest.NewRecorder()
	handlerCreate := http.HandlerFunc(CreateProduct)
	handlerCreate.ServeHTTP(rrCreate, reqCreate)

	var createdProduct Product
	json.NewDecoder(rrCreate.Body).Decode(&createdProduct)

	reqDelete, _ := http.NewRequest("DELETE", "/products?id="+strconv.Itoa(createdProduct.ID), nil)
	rrDelete := httptest.NewRecorder()
	handlerDelete := http.HandlerFunc(DeleteProduct)
	handlerDelete.ServeHTTP(rrDelete, reqDelete)

	if status := rrDelete.Code; status != http.StatusNoContent {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNoContent)
	}
}
```