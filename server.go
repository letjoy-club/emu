package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

type engineKey struct{}

func GetEngine(r *http.Request) *Engine {
	return r.Context().Value(engineKey{}).(*Engine)
}

func WithEngine(engine *Engine) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			ctx = context.WithValue(ctx, engineKey{}, engine)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

type serviceKey struct{}

func RequireServiceMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		exec := chi.URLParam(r, "service")
		engine := GetEngine(r)
		service := engine.GetService(exec)
		if service == nil {
			render.JSON(w, r, NewError(ErrServiceNotFound))
			return
		}
		ctx := r.Context()
		ctx = context.WithValue(ctx, serviceKey{}, service)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func GetService(r *http.Request) *Service {
	return r.Context().Value(serviceKey{}).(*Service)
}

func UploadHandler(w http.ResponseWriter, r *http.Request) {
	engine := GetEngine(r)
	service := GetService(r)
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

	engine.StopService(service.Exec)

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
	if err := engine.StartService(service.Exec); err != nil {
		render.JSON(w, r, NewError(err))
		return
	}
	render.JSON(w, r, NewData(nil))
}

func StartHandler(w http.ResponseWriter, r *http.Request) {
	engine := GetEngine(r)
	service := GetService(r)
	if err := engine.StartService(service.Exec); err != nil {
		render.JSON(w, r, NewError(err))
		return
	}
	render.JSON(w, r, NewData(nil))
}

func StopHandler(w http.ResponseWriter, r *http.Request) {
	engine := GetEngine(r)
	service := GetService(r)
	if err := engine.StopService(service.Exec); err != nil {
		render.JSON(w, r, NewError(err))
		return
	}
	render.JSON(w, r, NewData(nil))
}

func RestartHandler(w http.ResponseWriter, r *http.Request) {
	engine := GetEngine(r)
	service := GetService(r)
	if err := engine.Restart(service.Exec); err != nil {
		render.JSON(w, r, NewError(err))
		return
	}
	render.JSON(w, r, NewData(nil))
}

func GetConfigHandler(w http.ResponseWriter, r *http.Request) {
	service := GetService(r)
	render.JSON(w, r, NewData(service))
}

func GetOutputHandler(w http.ResponseWriter, r *http.Request) {
	service := GetService(r)
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		render.JSON(w, r, NewError(err))
		return
	}
	hub.Join(service.Exec, conn)
}
