// orders-api: Go service.
// Saves orders in Postgres, pushes a job on the Redis queue and records stats.
//
//	GET  /health       health check (Postgres + Redis)
//	GET  /info         version and the pod that answered
//	GET  /orders       latest orders with items, newest first
//	GET  /orders/{id}  one order with items
//	POST /orders       create an order
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
	"github.com/lib/pq"
)

const (
	queueKey     = "orders:queue"
	statsTSKey   = "stats:orders_ts"
	statsDishKey = "stats:top_dishes"
)

var (
	db      *sql.DB
	rdb     *redis.Client
	ctx     = context.Background()
	version = "dev"
)

// ---- Types ----

type OrderItem struct {
	MenuItemID int     `json:"menu_item_id"`
	Name       string  `json:"name"`
	Price      float64 `json:"price"`
	Quantity   int     `json:"quantity"`
}

type Order struct {
	ID           int         `json:"id"`
	CustomerName string      `json:"customer_name"`
	Status       string      `json:"status"`
	Items        []OrderItem `json:"items"`
	CreatedAt    time.Time   `json:"created_at"`
	UpdatedAt    time.Time   `json:"updated_at"`
}

type CreateOrderRequest struct {
	CustomerName string      `json:"customer_name"`
	Items        []OrderItem `json:"items"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

// ---- Helpers ----

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, ErrorResponse{Error: msg})
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// loadItems returns the items for the given order IDs, keyed by order ID.
func loadItems(ids []int64) (map[int][]OrderItem, error) {
	out := make(map[int][]OrderItem)
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := db.QueryContext(ctx,
		`SELECT order_id, menu_item_id, name, price, quantity
		 FROM order_items WHERE order_id = ANY($1) ORDER BY id`, pq.Array(ids))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var orderID int
		var it OrderItem
		if err := rows.Scan(&orderID, &it.MenuItemID, &it.Name, &it.Price, &it.Quantity); err != nil {
			return nil, err
		}
		out[orderID] = append(out[orderID], it)
	}
	return out, rows.Err()
}

// ---- Handlers ----

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	if err := db.PingContext(ctx); err != nil {
		writeError(w, http.StatusServiceUnavailable, "postgres unavailable: "+err.Error())
		return
	}
	if err := rdb.Ping(ctx).Err(); err != nil {
		writeError(w, http.StatusServiceUnavailable, "redis unavailable: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "version": version})
}

func infoHandler(w http.ResponseWriter, _ *http.Request) {
	pod := os.Getenv("POD_NAME")
	if pod == "" {
		pod, _ = os.Hostname()
	}
	writeJSON(w, http.StatusOK, map[string]string{"version": version, "pod": pod})
}

func listOrdersHandler(w http.ResponseWriter, _ *http.Request) {
	rows, err := db.QueryContext(ctx,
		`SELECT id, customer_name, status, created_at, updated_at
		 FROM orders ORDER BY created_at DESC LIMIT 50`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list orders")
		log.Printf("list orders: %v", err)
		return
	}
	defer rows.Close()

	orders := []Order{}
	ids := []int64{}
	for rows.Next() {
		var o Order
		if err := rows.Scan(&o.ID, &o.CustomerName, &o.Status, &o.CreatedAt, &o.UpdatedAt); err != nil {
			writeError(w, http.StatusInternalServerError, "could not read orders")
			log.Printf("scan order: %v", err)
			return
		}
		orders = append(orders, o)
		ids = append(ids, int64(o.ID))
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "could not read orders")
		return
	}

	items, err := loadItems(ids)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load order items")
		log.Printf("load items: %v", err)
		return
	}
	for i := range orders {
		orders[i].Items = items[orders[i].ID]
		if orders[i].Items == nil {
			orders[i].Items = []OrderItem{}
		}
	}
	writeJSON(w, http.StatusOK, orders)
}

func getOrderHandler(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid order id")
		return
	}
	var o Order
	err = db.QueryRowContext(ctx,
		`SELECT id, customer_name, status, created_at, updated_at FROM orders WHERE id = $1`, id).
		Scan(&o.ID, &o.CustomerName, &o.Status, &o.CreatedAt, &o.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "order not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not read order")
		log.Printf("get order: %v", err)
		return
	}
	items, err := loadItems([]int64{int64(o.ID)})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load order items")
		return
	}
	o.Items = items[o.ID]
	if o.Items == nil {
		o.Items = []OrderItem{}
	}
	writeJSON(w, http.StatusOK, o)
}

func createOrderHandler(w http.ResponseWriter, r *http.Request) {
	var req CreateOrderRequest
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	req.CustomerName = strings.TrimSpace(req.CustomerName)
	if req.CustomerName == "" {
		writeError(w, http.StatusBadRequest, "customer_name is required")
		return
	}
	if len(req.Items) == 0 {
		writeError(w, http.StatusBadRequest, "at least one item is required")
		return
	}
	for _, it := range req.Items {
		if it.MenuItemID <= 0 || it.Quantity <= 0 || it.Quantity > 50 {
			writeError(w, http.StatusBadRequest, "each item needs a menu_item_id and a quantity from 1 to 50")
			return
		}
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not start transaction")
		log.Printf("begin tx: %v", err)
		return
	}
	defer tx.Rollback() //nolint:errcheck

	order := Order{CustomerName: req.CustomerName, Items: []OrderItem{}}
	err = tx.QueryRowContext(ctx,
		`INSERT INTO orders (customer_name) VALUES ($1)
		 RETURNING id, status, created_at, updated_at`, req.CustomerName).
		Scan(&order.ID, &order.Status, &order.CreatedAt, &order.UpdatedAt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not save order")
		log.Printf("insert order: %v", err)
		return
	}

	for _, it := range req.Items {
		// Name and price always come from the menu table, never from the client.
		var name string
		var price float64
		err := tx.QueryRowContext(ctx,
			`SELECT name, price FROM menu_items WHERE id = $1 AND available = true`, it.MenuItemID).
			Scan(&name, &price)
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("menu item %d is not available", it.MenuItemID))
			return
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not read menu")
			log.Printf("read menu item: %v", err)
			return
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO order_items (order_id, menu_item_id, name, price, quantity)
			 VALUES ($1, $2, $3, $4, $5)`, order.ID, it.MenuItemID, name, price, it.Quantity); err != nil {
			writeError(w, http.StatusInternalServerError, "could not save order items")
			log.Printf("insert item: %v", err)
			return
		}
		order.Items = append(order.Items, OrderItem{
			MenuItemID: it.MenuItemID, Name: name, Price: price, Quantity: it.Quantity,
		})
	}

	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "could not commit order")
		log.Printf("commit: %v", err)
		return
	}

	publishOrder(order)
	writeJSON(w, http.StatusCreated, order)
}

// publishOrder queues the job for the kitchen worker and updates live stats.
// The order is already saved at this point, so Redis errors are logged, not returned.
func publishOrder(order Order) {
	job, _ := json.Marshal(map[string]int{"order_id": order.ID})
	if err := rdb.LPush(ctx, queueKey, job).Err(); err != nil {
		log.Printf("ERROR order %d saved but not queued: %v", order.ID, err)
	}

	now := time.Now()
	pipe := rdb.Pipeline()
	pipe.ZAdd(ctx, statsTSKey, &redis.Z{Score: float64(now.UnixMilli()), Member: strconv.Itoa(order.ID)})
	pipe.ZRemRangeByScore(ctx, statsTSKey, "-inf", strconv.FormatInt(now.Add(-time.Hour).UnixMilli(), 10))
	for _, it := range order.Items {
		pipe.ZIncrBy(ctx, statsDishKey, float64(it.Quantity), it.Name)
	}
	if _, err := pipe.Exec(ctx); err != nil {
		log.Printf("WARN stats update for order %d failed: %v", order.ID, err)
	}
}

// ---- Startup ----

// selfCheck is used by the Docker HEALTHCHECK, because the final image has no shell or curl.
func selfCheck() int {
	client := http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("http://localhost:" + getenv("PORT", "8080") + "/health")
	if err != nil || resp.StatusCode != http.StatusOK {
		return 1
	}
	return 0
}

func connect() {
	var err error
	db, err = sql.Open("postgres", getenv("DATABASE_URL", "postgres://dhaba:dhaba@postgres:5432/dhaba?sslmode=disable"))
	if err != nil {
		log.Fatalf("postgres config: %v", err)
	}
	db.SetMaxOpenConns(10)

	opt, err := redis.ParseURL(getenv("REDIS_URL", "redis://redis:6379"))
	if err != nil {
		log.Fatalf("redis config: %v", err)
	}
	rdb = redis.NewClient(opt)

	// Wait for dependencies, so the pod does not crash-loop while they start.
	for i := 1; i <= 30; i++ {
		if db.PingContext(ctx) == nil && rdb.Ping(ctx).Err() == nil {
			return
		}
		log.Printf("waiting for postgres and redis (%d/30)", i)
		time.Sleep(2 * time.Second)
	}
	log.Fatal("postgres or redis not reachable")
}

func main() {
	if v := os.Getenv("APP_VERSION"); v != "" {
		version = v
	}
	if len(os.Args) > 1 && os.Args[1] == "--health-check" {
		os.Exit(selfCheck())
	}

	connect()

	r := mux.NewRouter()
	r.HandleFunc("/health", healthHandler).Methods(http.MethodGet)
	r.HandleFunc("/info", infoHandler).Methods(http.MethodGet)
	r.HandleFunc("/orders", listOrdersHandler).Methods(http.MethodGet)
	r.HandleFunc("/orders", createOrderHandler).Methods(http.MethodPost)
	r.HandleFunc("/orders/{id}", getOrderHandler).Methods(http.MethodGet)

	addr := ":" + getenv("PORT", "8080")
	log.Printf("orders-api %s listening on %s", version, addr)
	srv := &http.Server{
		Addr:              addr,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Fatal(srv.ListenAndServe())
}
