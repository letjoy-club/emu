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
	"path"
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
	engine.Init(config.Mode, config.Services, config.MetaVars)

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
		r.Get("/config", func(w http.ResponseWriter, r *http.Request) {
			render.JSON(w, r, NewData(config))
		})
		r.Route("/service", func(r chi.Router) {
			r.Use(middleware.DefaultLogger)
			r.Get("/", func(w http.ResponseWriter, r *http.Request) {
				render.JSON(w, r, NewData(config.Services))
			})

			r.Route("/{service}", func(r chi.Router) {
				r.Use(WithEngine(&engine))
				r.Use(RequireServiceMiddleware)
				r.Get("/clear", func(w http.ResponseWriter, r *http.Request) {
					err := clearHistory()
					render.JSON(w, r, NewData(err))
				})
				r.Get("/config", GetConfigHandler)
				r.Get("/config-file", func(w http.ResponseWriter, r *http.Request) {
					service := GetService(r)
					if service.ConfigFile == "" {
						render.JSON(w, r, NewError(ErrServiceConfigNotFound))
						return
					}
					data, err := os.ReadFile(path.Join("service", service.ConfigFile))
					if err != nil {
						render.JSON(w, r, NewError(err))
						return
					}
					render.PlainText(w, r, string(data))
				})
				r.Post("/config-file", func(w http.ResponseWriter, r *http.Request) {
					service := GetService(r)
					if service.ConfigFile == "" {
						render.JSON(w, r, NewError(ErrServiceConfigNotFound))
						return
					}
					data, err := io.ReadAll(r.Body)
					if err != nil {
						render.JSON(w, r, NewError(err))
						return
					}
					defer r.Body.Close()
					err = os.WriteFile(path.Join("service", service.ConfigFile), data, 0644)
					if err != nil {
						render.JSON(w, r, NewError(err))
						return
					}
					render.JSON(w, r, NewData(nil))
				})
				r.Post("/restart", RestartHandler)
				r.Post("/upload", UploadHandler)
				r.Post("/start", StartHandler)
				r.Post("/stop", StopHandler)
				r.Get("/output", GetOutputHandler)
				r.Get("/log", func(w http.ResponseWriter, r *http.Request) {
					service := GetService(r)
					render.JSON(w, r, NewData(service.runner.LogFiles()))
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
