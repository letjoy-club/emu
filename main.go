package main

import (
	"embed"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"gopkg.in/yaml.v3"
)

//go:embed static
var static embed.FS

type Resp struct {
	Data interface{} `json:"data"`
	Err  string      `json:"error"`
}

func NewError(err error) Resp {
	return Resp{Err: err.Error()}
}

func NewData(data interface{}) Resp {
	return Resp{Data: data}
}

func main() {

	configPtr := flag.String("config", "config.yaml", "config file path")
	flag.Parse()

	if _, err := os.Stat(*configPtr); os.IsNotExist(err) {
		// Create default config file
		data, err := yaml.Marshal(GenerateDefault())
		if err != nil {
			panic(err)
		}
		fmt.Println("no config file found, creating default config file")
		err = os.WriteFile(*configPtr, data, 0644)
		if err != nil {
			panic(err)
		}
		return
	}
	config, err := readConfigFromFile(*configPtr)
	if err != nil {
		panic(err)
	}

	go hub.Start()
	engine := Engine{}
	engine.Init(config.Mode, config.Services)

	r := chi.NewRouter()
	// r.Use(middleware.DefaultLogger)
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		data, err := static.ReadFile("static/index.html")
		if err != nil {
			w.Write([]byte(err.Error()))
			return
		}
		w.Write(data)
	})

	var staticFS = fs.FS(static)
	htmlContent, err := fs.Sub(staticFS, "static")
	if err != nil {
		log.Fatal(err)
	}
	fs := http.FileServer(http.FS(htmlContent))

	// fs := http.FileServer(http.Dir("./static/"))
	r.Handle("/static/*", http.StripPrefix("/", fs))

	r.Route("/api", func(r chi.Router) {
		r.Use(middleware.BasicAuth("letjoy", config.AccountMap()))
		r.Route("/service", func(r chi.Router) {
			r.Get("/", func(w http.ResponseWriter, r *http.Request) {
				render.JSON(w, r, NewData(config.Services))
			})

			r.Route("/{service}", func(r chi.Router) {
				r.Get("/clear", func(w http.ResponseWriter, r *http.Request) {
					err := clearHistory()
					render.JSON(w, r, NewData(err))
				})
				r.Get("/config", func(w http.ResponseWriter, r *http.Request) {
					name := chi.URLParam(r, "service")
					service := engine.GetService(name)
					if service != nil {
						render.JSON(w, r, NewData(service))
					} else {
						render.JSON(w, r, NewError(ErrServiceNotFound))
					}
				})
				r.Post("/restart", func(w http.ResponseWriter, r *http.Request) {
					name := chi.URLParam(r, "service")
					err := engine.Restart(name)
					if err == nil {
						render.JSON(w, r, NewData(nil))
					} else {
						render.JSON(w, r, NewError(err))
					}
				})
				r.Post("/upload", func(w http.ResponseWriter, r *http.Request) {
					name := chi.URLParam(r, "service")
					service := engine.GetService(name)
					if service != nil {
						defer r.Body.Close()
						if err := r.ParseMultipartForm(1 << 20); err != nil {
							render.JSON(w, r, NewError(err))
							return
						}
						fileResume, fileHeader, err := r.FormFile("file")
						if err != nil {
							render.JSON(w, r, NewError(err))
							return
						}
						defer fileResume.Close()
						var filename string
						var uploadErr error
						if service.Packed() {
							filename, uploadErr = UploadSmallFiles(service.Exec+".tar.gz", "binary", fileResume)
						} else {
							filename, uploadErr = UploadSmallFiles(service.Exec, "binary", fileResume)
						}
						if uploadErr != nil {
							fmt.Println("file size:", fileHeader.Size)
							render.JSON(w, r, NewError(uploadErr))
							return
						}
						if service.Packed() {
							serviceFolder := service.ServiceFolder()
							if _, err := os.Stat(serviceFolder); err == nil {
								os.RemoveAll(serviceFolder + ".bak")
								os.Rename(serviceFolder, serviceFolder+".bak")
							}

						} else {
							// 如果是二进制文件
							if _, err := CopyFile(service.ExecPath(), service.ExecPath()+".bak"); err != nil {
								render.JSON(w, r, err)
								return
							}
						}

						engine.StopService(name)

						if service.Packed() {
							// 如果是压缩包，直接解压
							serviceFolder := service.ServiceFolder()
							if err := ExtractTarGz(filename, serviceFolder); err != nil {
								render.JSON(w, r, NewError(err))
								return
							}
						} else {
							// 如果是二进制文件，需要手动覆盖
							if err := os.Rename(filename, service.ExecPath()); err != nil {
								render.JSON(w, r, NewError(err))
								return
							}
						}
						if err := engine.StartService(name); err != nil {
							render.JSON(w, r, NewError(err))
							return
						}
						render.JSON(w, r, NewData(nil))
					} else {
						render.JSON(w, r, NewError(ErrServiceNotFound))
					}
				})
				r.Post("/start", func(w http.ResponseWriter, r *http.Request) {
					name := chi.URLParam(r, "service")
					err := engine.StartService(name)
					if err == nil {
						render.JSON(w, r, NewData(nil))
					} else {
						render.JSON(w, r, NewError(err))
					}
				})
				r.Post("/stop", func(w http.ResponseWriter, r *http.Request) {
					name := chi.URLParam(r, "service")
					err := engine.StopService(name)
					if err == nil {
						render.JSON(w, r, NewData(nil))
					} else {
						render.JSON(w, r, NewError(err))
					}
				})
				r.Get("/output", func(w http.ResponseWriter, r *http.Request) {
					name := chi.URLParam(r, "service")
					if engine.GetService(name) == nil {
						render.JSON(w, r, NewError(ErrServiceNotFound))
						return
					}
					conn, err := upgrader.Upgrade(w, r, nil)
					if err != nil {
						render.JSON(w, r, NewError(err))
						return
					}
					hub.Join(name, conn)
				})
				r.Get("/log", func(w http.ResponseWriter, r *http.Request) {
					name := chi.URLParam(r, "service")
					service := engine.GetService(name)
					if service != nil {
						render.JSON(w, r, NewData(service.runner.LogFiles()))
					} else {
						render.JSON(w, r, NewError(ErrServiceNotFound))
					}
				})
				r.Get("/log/{file}", func(w http.ResponseWriter, r *http.Request) {
					file := chi.URLParam(r, "file")
					reader, err := os.Open("log/" + file)
					if err != nil {
						fmt.Println(err)
						render.JSON(w, r, NewError(err))
						return
					}
					defer reader.Close()
					w.Header().Set("Content-Disposition", "attachment; filename="+file)
					fs, _ := reader.Stat()
					size := fs.Size()
					w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
					io.Copy(w, reader)
				})
			})
		})
	})

	if config.Port == 0 {
		config.Port = 8080
	}

	fmt.Printf("listening on port: %d\n", config.Port)
	if err := http.ListenAndServe(fmt.Sprintf(":%d", config.Port), r); err != nil {
		panic(err)
	}
}
