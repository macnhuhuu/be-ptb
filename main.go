package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type Session struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Photos    []string          `bson:"photos" json:"photos"`
	CreatedAt time.Time          `bson:"created_at" json:"createdAt"`
}

type StatusResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	DB      string `json:"db"`
}

var mongoClient *mongo.Client
var sessionsCollection *mongo.Collection

func main() {
	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://macchu:huuhuu123@ac-pyjeukq-shard-00-00.qlupeij.mongodb.net:27017,ac-pyjeukq-shard-00-01.qlupeij.mongodb.net:27017,ac-pyjeukq-shard-00-02.qlupeij.mongodb.net:27017/?ssl=true&replicaSet=atlas-8ps0fd-shard-0&authSource=admin&appName=Cluster0"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Printf("MongoDB Init Error: %v\n", err)
	} else {
		err = client.Ping(ctx, readpref.Primary())
		if err != nil {
			log.Printf("MongoDB Ping Error: %v\n", err)
		} else {
			log.Println("✅ Connected to MongoDB Atlas!")
			mongoClient = client
			sessionsCollection = client.Database("photobooth").Collection("sessions")
		}
	}

	mux := http.NewServeMux()

	// Middleware for CORS
	withCORS := func(h http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			if r.Method == "OPTIONS" {
				return
			}
			h(w, r)
		}
	}

	mux.HandleFunc("/", withCORS(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Welcome to ProBooth Backend API",
			"status":  "active",
		})
	}))

	mux.HandleFunc("/api/status", withCORS(func(w http.ResponseWriter, r *http.Request) {
		dbStatus := "disconnected"
		if mongoClient != nil {
			dbStatus = "connected"
		}
		json.NewEncoder(w).Encode(StatusResponse{
			Status:  "ok",
			Message: "Backend running",
			DB:      dbStatus,
		})
	}))

	mux.HandleFunc("/api/sessions", withCORS(func(w http.ResponseWriter, r *http.Request) {
		// --- TRƯỜNG HỢP POST (LƯU ẢNH) ---
		if r.Method == http.MethodPost {
			var sess Session
			if err := json.NewDecoder(r.Body).Decode(&sess); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			sess.CreatedAt = time.Now()
			
			if sessionsCollection != nil {
				res, err := sessionsCollection.InsertOne(context.TODO(), sess)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				sess.ID = res.InsertedID.(primitive.ObjectID)
			} else {
				sess.ID = primitive.NewObjectID()
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(sess)
			return
		}

		// --- TRƯỜNG HỢP GET (LẤY ẢNH QUA ID) ---
		// URL có dạng /api/sessions/643...
		if r.Method == http.MethodGet {
			path := strings.TrimPrefix(r.URL.Path, "/api/sessions")
			idStr := strings.TrimPrefix(path, "/")
			
			if idStr == "" {
				http.Error(w, "Thiếu ID phiên chụp", http.StatusBadRequest)
				return
			}

			objID, err := primitive.ObjectIDFromHex(idStr)
			if err != nil {
				http.Error(w, "Định dạng ID không hợp lệ: "+idStr, http.StatusBadRequest)
				return
			}

			var sess Session
			err = sessionsCollection.FindOne(context.TODO(), bson.M{"_id": objID}).Decode(&sess)
			if err != nil {
				http.Error(w, "Không tìm thấy phiên chụp trong MongoDB", http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(sess)
			return
		}
	}))

	log.Println("Server running at http://localhost:8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
