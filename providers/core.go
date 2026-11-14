package providers

import (
	"log"

	"github.com/Caknoooo/go-gin-clean-starter/config"
	authController "github.com/Caknoooo/go-gin-clean-starter/modules/auth/controller"
	authRepo "github.com/Caknoooo/go-gin-clean-starter/modules/auth/repository"
	authService "github.com/Caknoooo/go-gin-clean-starter/modules/auth/service"
	userController "github.com/Caknoooo/go-gin-clean-starter/modules/user/controller"
	"github.com/Caknoooo/go-gin-clean-starter/modules/user/repository"
	userService "github.com/Caknoooo/go-gin-clean-starter/modules/user/service"

	akunController "github.com/Caknoooo/go-gin-clean-starter/modules/akun/controller"
	akunRepo "github.com/Caknoooo/go-gin-clean-starter/modules/akun/repository"
	akunService "github.com/Caknoooo/go-gin-clean-starter/modules/akun/service"
	cutiController "github.com/Caknoooo/go-gin-clean-starter/modules/cuti/controller"
	cutiRepo "github.com/Caknoooo/go-gin-clean-starter/modules/cuti/repository"
	cutiService "github.com/Caknoooo/go-gin-clean-starter/modules/cuti/service"
	dokumenController "github.com/Caknoooo/go-gin-clean-starter/modules/dokumen/controller"
	dokumenRepo "github.com/Caknoooo/go-gin-clean-starter/modules/dokumen/repository"
	dokumenService "github.com/Caknoooo/go-gin-clean-starter/modules/dokumen/service"
	notifikasiController "github.com/Caknoooo/go-gin-clean-starter/modules/notifikasi/controller"
	notifikasiRepo "github.com/Caknoooo/go-gin-clean-starter/modules/notifikasi/repository"
	notifikasiService "github.com/Caknoooo/go-gin-clean-starter/modules/notifikasi/service"
	pendidikController "github.com/Caknoooo/go-gin-clean-starter/modules/pendidik/controller"
	pendidikRepo "github.com/Caknoooo/go-gin-clean-starter/modules/pendidik/repository"
	pendidikService "github.com/Caknoooo/go-gin-clean-starter/modules/pendidik/service"
	sekolahController "github.com/Caknoooo/go-gin-clean-starter/modules/sekolah/controller"
	sekolahRepo "github.com/Caknoooo/go-gin-clean-starter/modules/sekolah/repository"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/constants"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/storage"
	"github.com/samber/do"
	"gorm.io/gorm"
)

func InitDatabase(injector *do.Injector) {
	do.ProvideNamed(injector, constants.DB, func(i *do.Injector) (*gorm.DB, error) {
		return config.SetUpDatabaseConnection(), nil
	})
}

func RegisterDependencies(injector *do.Injector) {
	InitDatabase(injector)

	do.ProvideNamed(injector, constants.JWTService, func(i *do.Injector) (authService.JWTService, error) {
		return authService.NewJWTService(), nil
	})

	db := do.MustInvokeNamed[*gorm.DB](injector, constants.DB)
	jwtService := do.MustInvokeNamed[authService.JWTService](injector, constants.JWTService)

	userRepository := repository.NewUserRepository(db)
	refreshTokenRepository := authRepo.NewRefreshTokenRepository(db)

	userService := userService.NewUserService(userRepository, db)
	authService := authService.NewAuthService(userRepository, refreshTokenRepository, jwtService, db)

	do.Provide(
		injector, func(i *do.Injector) (userController.UserController, error) {
			return userController.NewUserController(userService), nil
		},
	)

	do.Provide(
		injector, func(i *do.Injector) (authController.AuthController, error) {
			return authController.NewAuthController(authService), nil
		},
	)

	// Fondasi
	st, err := storage.NewFromEnv()
	if err != nil {
		log.Fatalf("konfigurasi object storage: %v", err)
	}
	dokumenSvc := dokumenService.NewDokumenService(dokumenRepo.NewDokumenRepository(db), db, st)
	notifikasiSvc := notifikasiService.NewNotifikasiService(notifikasiRepo.NewNotifikasiRepository(db))
	pendidikSvc := pendidikService.NewPendidikService(pendidikRepo.NewPendidikRepository(db), db)

	do.Provide(injector, func(i *do.Injector) (dokumenController.DokumenController, error) {
		return dokumenController.NewDokumenController(dokumenSvc), nil
	})
	do.Provide(injector, func(i *do.Injector) (notifikasiController.NotifikasiController, error) {
		return notifikasiController.NewNotifikasiController(notifikasiSvc), nil
	})
	do.Provide(injector, func(i *do.Injector) (sekolahController.SekolahController, error) {
		return sekolahController.NewSekolahController(sekolahRepo.NewSekolahRepository(db)), nil
	})
	do.Provide(injector, func(i *do.Injector) (pendidikController.PendidikController, error) {
		return pendidikController.NewPendidikController(pendidikSvc), nil
	})

	do.Provide(injector, func(i *do.Injector) (akunController.AkunController, error) {
		return akunController.NewAkunController(akunService.NewAkunService(akunRepo.NewAkunRepository(db), db)), nil
	})

	// Cuti
	cutiSvc := cutiService.NewCutiService(cutiRepo.NewCutiRepository(db), dokumenSvc, notifikasiSvc, db)
	do.Provide(injector, func(i *do.Injector) (cutiController.CutiController, error) {
		return cutiController.NewCutiController(cutiSvc), nil
	})
}
