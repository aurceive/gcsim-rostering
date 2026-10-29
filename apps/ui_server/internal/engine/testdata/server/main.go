package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
)

func main() {
	host := flag.String("host", "127.0.0.1", "")
	port := flag.String("port", "", "")
	flag.Int("workers", 0, "")
	flag.Int("timeout", 0, "")
	flag.Parse()
	exe, err := os.Executable()
	if err != nil {
		panic(err)
	}
	name := filepath.Base(filepath.Dir(exe))
	var pending atomic.Int32
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/pending") && pending.Add(1) == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = fmt.Fprint(w, "unexpected result is blank")
			return
		}
		if strings.HasSuffix(r.URL.Path, "/failure") {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = fmt.Fprint(w, "actual failure")
			return
		}
		_, _ = fmt.Fprintf(w, "%s:%s:%s", name, r.Method, r.URL.Path)
	})
	if err := http.ListenAndServe(*host+":"+*port, nil); err != nil {
		panic(err)
	}
}
