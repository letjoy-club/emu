package main

import (
	"embed"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"

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

	runners := map[string]*Runner{}
	for _, s := range config.Services {
		runner := NewRunner(s, config.Mode)
		runner.Start()
		runners[s.Name] = runner
	}

	r := chi.NewRouter()
	r.Use(middleware.DefaultLogger)
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
				r.Get("/config", func(w http.ResponseWriter, r *http.Request) {
					service := getService(r, config.Services)
					render.JSON(w, r, NewData(service))
				})
				r.Post("/restart", func(w http.ResponseWriter, r *http.Request) {
					service := getService(r, config.Services)
					if service != nil {
						runner := runners[service.Name]
						runner.Stop()
						runner = NewRunner(service, config.Mode)
						runners[service.Name] = runner
						if err := runner.Start(); err != nil {
							render.JSON(w, r, NewData(err))
							return
						}
						render.JSON(w, r, NewData(nil))
					} else {
						render.JSON(w, r, NewError(errors.New("service not found")))
					}
				})
				r.Post("/upload", func(w http.ResponseWriter, r *http.Request) {
					service := getService(r, config.Services)
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
						filename, uploadErr := UploadSmallFiles(service.Exec, "binary", fileResume)
						if uploadErr != nil {
							fmt.Println("file size:", fileHeader.Size)
							render.JSON(w, r, NewError(uploadErr))
							return
						}
						os.Chmod(filename, 0777)
						if _, err := CopyFile(service.ExecPath(), service.ExecPath()+".bak"); err != nil {
							render.JSON(w, r, err)
							return
						}

						runner := runners[service.Name]
						runner.Stop()

						runner = NewRunner(service, config.Mode)
						runners[service.Name] = runner

						if err := os.Rename(filename, service.ExecPath()); err != nil {
							render.JSON(w, r, NewError(err))
							return
						}
						if err := runner.Start(); err != nil {
							render.JSON(w, r, NewError(err))
							return
						}
						render.JSON(w, r, NewData(nil))
					} else {
						render.JSON(w, r, NewError(errors.New("service not found")))
					}
				})
				r.Post("/start", func(w http.ResponseWriter, r *http.Request) {
					service := getService(r, config.Services)
					if service != nil {
						runner := runners[service.Name]
						runner.Stop()
						runner = NewRunner(service, config.Mode)
						runners[service.Name] = runner
						err := runner.Start()
						if err != nil {
							render.JSON(w, r, err)
							return
						}
						render.JSON(w, r, NewData(nil))
					} else {
						render.JSON(w, r, NewError(errors.New("service not found")))
					}
				})
				r.Post("/stop", func(w http.ResponseWriter, r *http.Request) {
					service := getService(r, config.Services)
					if service != nil {
						runner := runners[service.Name]
						runner.Stop()
						render.JSON(w, r, NewData(nil))
					} else {
						render.JSON(w, r, NewError(errors.New("service not found")))
					}
				})
				r.Get("/log", func(w http.ResponseWriter, r *http.Request) {
				})
				r.Get("/log/{file}", func(w http.ResponseWriter, r *http.Request) {
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

func getService(r *http.Request, services []*Service) *Service {
	service := chi.URLParam(r, "service")
	for _, s := range services {
		if s.Exec == service {
			fmt.Println("service found", s.Exec)
			return s
		}
	}
	return nil
}
