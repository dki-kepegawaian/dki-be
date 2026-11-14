package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/Caknoooo/go-gin-clean-starter/docs"
	"github.com/Caknoooo/go-gin-clean-starter/middlewares"
	"github.com/Caknoooo/go-gin-clean-starter/modules/akun"
	"github.com/Caknoooo/go-gin-clean-starter/modules/auth"
	"github.com/Caknoooo/go-gin-clean-starter/modules/cuti"
	"github.com/Caknoooo/go-gin-clean-starter/modules/dokumen"
	"github.com/Caknoooo/go-gin-clean-starter/modules/notifikasi"
	"github.com/Caknoooo/go-gin-clean-starter/modules/pendidik"
	"github.com/Caknoooo/go-gin-clean-starter/modules/sekolah"
	"github.com/Caknoooo/go-gin-clean-starter/modules/user"
	"github.com/Caknoooo/go-gin-clean-starter/providers"
	"github.com/Caknoooo/go-gin-clean-starter/script"
	"github.com/samber/do"

	"github.com/common-nighthawk/go-figure"
	"github.com/gin-gonic/gin"
)

func args(injector *do.Injector) bool {
	if len(os.Args) > 1 {
		flag := script.Commands(injector)
		return flag
	}

	return true
}

func run(server *gin.Engine) {
	server.Static("/assets", "./assets")

	port := os.Getenv("GOLANG_PORT")
	if port == "" {
		port = "8888"
	}

	var serve string
	if os.Getenv("APP_ENV") == "localhost" {
		serve = "0.0.0.0:" + port
	} else {
		serve = ":" + port
	}

	myFigure := figure.NewColorFigure("Caknoo", "", "green", true)
	myFigure.Print()

	srv := &http.Server{
		Addr:              serve,
		Handler:           server,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      120 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("error running server: %v", err)
	}
}

func main() {
	var (
		injector = do.New()
	)

	providers.RegisterDependencies(injector)

	if !args(injector) {
		return
	}

	server := gin.Default()
	server.MaxMultipartMemory = 8 << 20
	// Batas body di aplikasi, tidak bergantung pada nginx: JSON 1 MiB, multipart 11 MiB (berkas maks 10 MB).
	server.Use(middlewares.BatasiBody(1<<20, 11<<20))
	server.Use(middlewares.CORSMiddleware())

	// Register module routes
	user.RegisterRoutes(server, injector)
	auth.RegisterRoutes(server, injector)
	akun.RegisterRoutes(server, injector)
	sekolah.RegisterRoutes(server, injector)
	pendidik.RegisterRoutes(server, injector)
	dokumen.RegisterRoutes(server, injector)
	notifikasi.RegisterRoutes(server, injector)
	cuti.RegisterRoutes(server, injector)

	// Dokumentasi API (Scalar) di /docs
	docs.RegisterRoutes(server)

	run(server)
}
