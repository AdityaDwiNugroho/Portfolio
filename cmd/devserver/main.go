package main

import (
	"log"
	"net/http"
	"os"

	"github.com/AdityaDwiNugroho/Portfolio/api"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	fs := http.FileServer(http.Dir("public"))
	http.Handle("/", fs)

	http.HandleFunc("/api/projects", api.Projects)
	http.HandleFunc("/api/contact", api.Contact)
	http.HandleFunc("/api/visitor", api.Visitor)

	log.Printf("Server starting on http://localhost:%s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
