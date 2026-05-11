package main

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const baseDir = "./storage_data"

func getSafePath(urlPath string) (string, error) {

	absBase, err := filepath.Abs(baseDir)

	if err != nil {

		return "", err

	}

	finalPath := filepath.Join(absBase, filepath.Clean(urlPath))

	if !strings.HasPrefix(finalPath, absBase) {

		return "", errors.New("Forbidden")

	}

	return finalPath, nil

}

func main() {

	os.MkdirAll(baseDir, 0755)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		path, err := getSafePath(r.URL.Path)

		if err != nil {

			http.Error(w, "Forbidden", http.StatusForbidden) // "403"

			return

		}

		switch r.Method {

		case http.MethodGet: // скачать - просмотр

			info, err := os.Stat(path)

			if err != nil {

				if os.IsNotExist(err) {

					http.Error(w, "Not Found", http.StatusNotFound) // "404"

				} else {

					http.Error(w, "Internal Server Error", http.StatusInternalServerError) // "500"

				}

				return

			}

			if info.IsDir() {

				entries, err := os.ReadDir(path)

				if err != nil {

					http.Error(w, "Internal Server Error", http.StatusInternalServerError) // "500"

					return

				}

				list := []string{}

				for _, e := range entries {

					list = append(list, e.Name())

				}

				w.Header().Set("Content-Type", "application/json")

				json.NewEncoder(w).Encode(list)

				log.Printf("[GET]: %s", r.URL.Path)

				return

			}

			http.ServeFile(w, r, path) ////

			log.Printf("[GET]: %s", r.URL.Path)

		case http.MethodPut: // создать - обновить

			info, err := os.Stat(path)

			exists := err == nil && !info.IsDir()

			if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {

				http.Error(w, "Internal Server Error", http.StatusInternalServerError) // "500"

				return

			}

			if srcHeader := r.Header.Get("X-Copy-From"); srcHeader != "" {

				srcPath, err := getSafePath(srcHeader)

				if err != nil {

					http.Error(w, "Forbidden", http.StatusForbidden) // "403"

					return

				}

				src, err := os.Open(srcPath)

				if err != nil {

					http.Error(w, "Not Found", http.StatusNotFound) // "404"

					return

				}

				defer src.Close()

				dst, err := os.Create(path)

				if err != nil {

					http.Error(w, "Internal Server Error", http.StatusInternalServerError) // "500"

					return

				}

				defer dst.Close()

				if _, err := io.Copy(dst, src); err != nil {

					http.Error(w, "Internal Server Error", http.StatusInternalServerError) // "500"

					return
				}

				log.Printf("[COP]: %s -> %s", srcHeader, r.URL.Path)

			} else {

				f, err := os.Create(path)

				if err != nil {

					http.Error(w, "Internal Server Error", http.StatusInternalServerError) // "500"

					return

				}

				defer f.Close()

				if _, err := io.Copy(f, r.Body); err != nil {

					http.Error(w, "Internal Server Error", http.StatusInternalServerError) // "500"

					return

				}

				log.Printf("[SAV]: %s", r.URL.Path)

			}

			if exists {

				w.WriteHeader(http.StatusNoContent) // "204"

			} else {

				w.WriteHeader(http.StatusCreated) // "201"

			}

		case http.MethodHead:

			info, err := os.Stat(path)

			if err != nil {

				w.WriteHeader(http.StatusNotFound) // "404"

				return

			}

			var size int64

			if info.IsDir() {

				filepath.Walk(path, func(_ string, f os.FileInfo, err error) error {

					if err == nil && !f.IsDir() {

						size += f.Size()

					}

					return nil

				})

			} else {

				size = info.Size()

			}

			w.Header().Set("Content-Length", strconv.FormatInt(size, 10))

			w.WriteHeader(http.StatusOK) // "200"

			log.Printf("[HED]: %s (Size: %d)", r.URL.Path, size)

		case http.MethodDelete:

			if _, err := os.Stat(path); err != nil {

				if os.IsNotExist(err) {

					http.Error(w, "Not Found", http.StatusNotFound)

					return

				}

				http.Error(w, "Internal Server Error", http.StatusInternalServerError) // "204"

				return

			}

			if err := os.RemoveAll(path); err != nil {

				http.Error(w, "Internal Server Error", http.StatusInternalServerError)

				return

			}

			log.Printf("[DEL]: %s", r.URL.Path)

			w.WriteHeader(http.StatusNoContent) // 204

		default:

			w.WriteHeader(http.StatusMethodNotAllowed)

		}

	})

	if err := http.ListenAndServe(":8080", nil); err != nil {

		log.Fatal(err)

	}

}
