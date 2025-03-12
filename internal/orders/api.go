```go
package orders

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/gorilla/mux"
)

// Order represents an order in the system
type Order struct {
	ID        string `json:"id"`
	UserID    string `json:"user_id"`
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
	Status    string `json:"status"`
}

// OrderStore holds the orders in memory
type OrderStore struct {
	sync.Mutex
	orders map[string]Order
}

// NewOrderStore creates a new OrderStore
func NewOrderStore() *OrderStore {
	return &OrderStore{orders: make(map[string]Order)}
}

// CreateOrderHandler handles the creation of a new order
func (store *OrderStore) CreateOrderHandler(w http.ResponseWriter, r *http.Request) {
	var order Order
	if err := json.NewDecoder(r.Body).Decode(&order); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	order.ID = generateOrderID() // Assume this function generates a unique ID
	store.Lock()
	store.orders[order.ID] = order
	store.Unlock()

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(order)
}

// GetOrderHandler handles retrieving an order by ID
func (store *OrderStore) GetOrderHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orderID := vars["id"]

	store.Lock()
	order, exists := store.orders[orderID]
	store.Unlock()

	if !exists {
		http.Error(w, "Order not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(order)
}

// SetupRoutes initializes the API routes
func SetupRoutes(store *OrderStore) *mux.Router {
	r := mux.NewRouter()
	r.HandleFunc("/orders", store.CreateOrderHandler).Methods("POST")
	r.HandleFunc("/orders/{id}", store.GetOrderHandler).Methods("GET")
	return r
}

// assume this function is implemented
func generateOrderID() string {
	// Unique ID generation logic here
	return "some-unique-id"
}
```

!!internal/orders/api_test.go!!
```go
package orders

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

func TestCreateOrder(t *testing.T) {
	store := NewOrderStore()
	router := SetupRoutes(store)

	order := Order{
		UserID:    "user123",
		ProductID: "product456",
		Quantity:  2,
		Status:    "pending",
	}

	orderJSON, _ := json.Marshal(order)
	req, _ := http.NewRequest("POST", "/orders", bytes.NewBuffer(orderJSON))
	response := httptest.NewRecorder()

	router.ServeHTTP(response, req)

	assert.Equal(t, http.StatusCreated, response.Code)

	var createdOrder Order
	json.Unmarshal(response.Body.Bytes(), &createdOrder)
	assert.NotEmpty(t, createdOrder.ID)
	assert.Equal(t, order.UserID, createdOrder.UserID)
	assert.Equal(t, order.ProductID, createdOrder.ProductID)
	assert.Equal(t, order.Quantity, createdOrder.Quantity)
	assert.Equal(t, order.Status, createdOrder.Status)
}

func TestGetOrder(t *testing.T) {
	store := NewOrderStore()
	router := SetupRoutes(store)

	order := Order{
		UserID:    "user123",
		ProductID: "product456",
		Quantity:  2,
		Status:    "pending",
	}

	store.CreateOrderHandler(httptest.NewRecorder(), httptest.NewRequest("POST", "/orders", bytes.NewBuffer(mustMarshal(t, order))))

	req, _ := http.NewRequest("GET", "/orders/"+order.ID, nil)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, req)

	assert.Equal(t, http.StatusOK, response.Code)

	var fetchedOrder Order
	json.Unmarshal(response.Body.Bytes(), &fetchedOrder)
	assert.Equal(t, order.UserID, fetchedOrder.UserID)
	assert.Equal(t, order.ProductID, fetchedOrder.ProductID)
	assert.Equal(t, order.Quantity, fetchedOrder.Quantity)
	assert.Equal(t, order.Status, fetchedOrder.Status)
}

func mustMarshal(t *testing.T, v interface{}) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
```