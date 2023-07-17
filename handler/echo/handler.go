package echo_handler

import (
	"fmt"
	"net/http"
)

// Handler prints request.
func Handler(greet string) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte(greet + r.URL.Query().Encode()))
		if err != nil {
			fmt.Printf("err: %v", err)
		}
	}
	return http.HandlerFunc(fn)
}
