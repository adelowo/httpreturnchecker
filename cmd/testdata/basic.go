package handler

import (
	"encoding/json"
	"net/http"
)

// OK - return here
func handler1(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("hello"))
	return
}

// OK - last statement
func handler2(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("hello"))
}

// Bad - write with no return and not last statement
func handler3(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("hello")) // want "response write operation must be followed by return unless it's the last statement"
	someOtherOperation()
}

// OK - writes in different branches with returns
func handler4(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		w.Write([]byte("post"))
		return
	}
	w.Write([]byte("get"))
}

// Bad - write with no return and other operations
func handler5(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK) // want "response write operation must be followed by return unless it's the last statement"
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// BAD - the delete branch misses a return
func handler6(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		w.Write([]byte("post"))
		return
	}

	if r.Method == "DELETE" {
		w.Write([]byte("delete")) // want "response write operation must be followed by return unless it's the last statement"
	}

	w.Write([]byte("get"))
}

// OK - writes in branches with whitespace before returns
func handler7(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		w.Write([]byte("post"))

		// Comment

		return
	}

	w.Write([]byte("get"))

	// Trailing comment
}

// Bad - writes in branches without returns, with whitespace
func handler8(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		w.Write([]byte("post"))

		// Comment
		return
	}

	if r.Method == "DELETE" {
		w.Write([]byte("delete")) // want "response write operation must be followed by return unless it's the last statement"

		// Comment
	}

	w.Write([]byte("get"))

	// Final comment
}

func someOtherOperation() {}
