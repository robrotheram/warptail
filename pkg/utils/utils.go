package utils

import (
	"encoding/json"
	"log"
	"net/http"
)

func WriteErrorResponse(w http.ResponseWriter, err *RouterError) {
	WriteResponse(w, err.StatusCode, err)
}

func WriteData(w http.ResponseWriter, data any) {
	WriteResponse(w, http.StatusOK, data)
}

func WriteStatus(w http.ResponseWriter, statusCode int) {
	WriteResponse(w, statusCode, nil)
}

func WriteResponse(w http.ResponseWriter, statusCode int, data any) {
	// Set content type to JSON
	w.Header().Set("Content-Type", "application/json")

	// Set custom status code
	w.WriteHeader(statusCode)

	// Marshal the response data into JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Printf("Error marshalling JSON: %v", err)
		http.Error(w, "Error processing the response", http.StatusInternalServerError)
		return
	}

	// Write JSON to the ResponseWriter
	if _, err := w.Write(jsonData); err != nil {
		log.Printf("Error writing response: %v", err)
		http.Error(w, "Error writing the response", http.StatusInternalServerError)
	}
}
