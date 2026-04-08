package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type StatusResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	DB      string `json:"db"`
}

var mongoClient *mongo.Client

func main() {
	// MongoDB Connection
	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		// Dùng URI cứng với mật khẩu cần thay thế nếu chưa có trong biến môi trường
		mongoURI = "mongodb://macchu:<db_password>@ac-pyjeukq-shard-00-00.qlupeij.mongodb.net:27017,ac-pyjeukq-shard-00-01.qlupeij.mongodb.net:27017,ac-pyjeukq-shard-00-02.qlupeij.mongodb.net:27017/?ssl=true&replicaSet=atlas-8ps0fd-shard-0&authSource=admin&appName=Cluster0"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Printf("Lỗi khởi tạo kết nối MongoDB: %v\n", err)
	} else {
		err = client.Ping(ctx, readpref.Primary())
		if err != nil {
			log.Printf("Không thể ping được tới MongoDB (có thể do sai mật khẩu): %v\n", err)
		} else {
			log.Println("✅ Đã kết nối thành công tới MongoDB Atlas!")
			mongoClient = client
		}
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Allow CORS for local dev
		w.Header().Set("Access-Control-Allow-Origin", "*")

		dbStatus := "disconnected"
		if mongoClient != nil {
			dbStatus = "connected"
		}

		json.NewEncoder(w).Encode(StatusResponse{
			Status:  "ok",
			Message: "Photobooth backend is running!",
			DB:      dbStatus,
		})
	})

	log.Println("Server đang chạy tại http://localhost:8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
