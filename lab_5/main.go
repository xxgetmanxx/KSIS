package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

const baseDir = "./storage_data"

func main() {

	os.RemoveAll(baseDir)

	os.MkdirAll(baseDir, 0755)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		path := filepath.Join(baseDir, filepath.Clean(r.URL.Path))

		switch r.Method {

		case http.MethodGet:
			info, err := os.Stat(path)
			if err != nil {
				http.Error(w, "Not Found", 404)
				return
			}
			if info.IsDir() {
				entries, _ := os.ReadDir(path)
				var list []string = []string{}
				for _, e := range entries {
					list = append(list, e.Name())
				}
				json.NewEncoder(w).Encode(list)
				log.Printf("[GET]: %s", r.URL.Path) // Короткий лог
				return
			}
			http.ServeFile(w, r, path)
			log.Printf("[GET]: %s", r.URL.Path) // Короткий лог

		case http.MethodPut:
			os.MkdirAll(filepath.Dir(path), 0755)
			if srcPath := r.Header.Get("X-Copy-From"); srcPath != "" {
				src, _ := os.Open(filepath.Join(baseDir, filepath.Clean(srcPath)))
				dst, _ := os.Create(path)
				io.Copy(dst, src)
				src.Close()
				dst.Close()
				log.Printf("[COP]: %s -> %s", srcPath, r.URL.Path) // Короткий лог
			} else {
				f, _ := os.Create(path)
				io.Copy(f, r.Body)
				f.Close()
				log.Printf("[SAV]: %s", r.URL.Path) // Короткий лог
			}
			w.WriteHeader(201)

		case http.MethodHead:
			info, err := os.Stat(path)
			if err != nil || info.IsDir() {
				w.WriteHeader(404)
				return
			}
			w.Header().Set("Content-Length", string(info.Size()))
			w.WriteHeader(200)
			log.Printf("[HED]: %s", r.URL.Path) // Короткий лог

		case http.MethodDelete:
			os.RemoveAll(path)
			log.Printf("[DEL]: %s", r.URL.Path) // Короткий лог
			w.WriteHeader(204)

		default:

			w.WriteHeader(405)

		}

	})

	log.Println("server_52")

	http.ListenAndServe(":8080", nil)

}
