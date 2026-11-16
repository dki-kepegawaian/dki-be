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
	anjabController "github.com/Caknoooo/go-gin-clean-starter/modules/anjab/controller"
	anjabRepo "github.com/Caknoooo/go-gin-clean-starter/modules/anjab/repository"
	anjabService "github.com/Caknoooo/go-gin-clean-starter/modules/anjab/service"
	arsipController "github.com/Caknoooo/go-gin-clean-starter/modules/arsip/controller"
	arsipRepo "github.com/Caknoooo/go-gin-clean-starter/modules/arsip/repository"
	arsipService "github.com/Caknoooo/go-gin-clean-starter/modules/arsip/service"
	cutiController "github.com/Caknoooo/go-gin-clean-starter/modules/cuti/controller"
	cutiRepo "github.com/Caknoooo/go-gin-clean-starter/modules/cuti/repository"
	cutiService "github.com/Caknoooo/go-gin-clean-starter/modules/cuti/service"
	dapodikController "github.com/Caknoooo/go-gin-clean-starter/modules/dapodik/controller"
	dapodikRepo "github.com/Caknoooo/go-gin-clean-starter/modules/dapodik/repository"
	dapodikService "github.com/Caknoooo/go-gin-clean-starter/modules/dapodik/service"
	dasborController "github.com/Caknoooo/go-gin-clean-starter/modules/dasbor/controller"
	dasborRepo "github.com/Caknoooo/go-gin-clean-starter/modules/dasbor/repository"
	dasborService "github.com/Caknoooo/go-gin-clean-starter/modules/dasbor/service"
	disiplinController "github.com/Caknoooo/go-gin-clean-starter/modules/disiplin/controller"
	disiplinRepo "github.com/Caknoooo/go-gin-clean-starter/modules/disiplin/repository"
	disiplinService "github.com/Caknoooo/go-gin-clean-starter/modules/disiplin/service"
	dokumenController "github.com/Caknoooo/go-gin-clean-starter/modules/dokumen/controller"
	dokumenRepo "github.com/Caknoooo/go-gin-clean-starter/modules/dokumen/repository"
	dokumenService "github.com/Caknoooo/go-gin-clean-starter/modules/dokumen/service"
	kpController "github.com/Caknoooo/go-gin-clean-starter/modules/kenaikanpangkat/controller"
	kpRepo "github.com/Caknoooo/go-gin-clean-starter/modules/kenaikanpangkat/repository"
	kpService "github.com/Caknoooo/go-gin-clean-starter/modules/kenaikanpangkat/service"
	kinerjaController "github.com/Caknoooo/go-gin-clean-starter/modules/kinerja/controller"
	kinerjaRepo "github.com/Caknoooo/go-gin-clean-starter/modules/kinerja/repository"
	kinerjaService "github.com/Caknoooo/go-gin-clean-starter/modules/kinerja/service"
	notifikasiController "github.com/Caknoooo/go-gin-clean-starter/modules/notifikasi/controller"
	notifikasiRepo "github.com/Caknoooo/go-gin-clean-starter/modules/notifikasi/repository"
	notifikasiService "github.com/Caknoooo/go-gin-clean-starter/modules/notifikasi/service"
	pendidikController "github.com/Caknoooo/go-gin-clean-starter/modules/pendidik/controller"
	pendidikRepo "github.com/Caknoooo/go-gin-clean-starter/modules/pendidik/repository"
	pendidikService "github.com/Caknoooo/go-gin-clean-starter/modules/pendidik/service"
	pensiunController "github.com/Caknoooo/go-gin-clean-starter/modules/pensiun/controller"
	pensiunRepo "github.com/Caknoooo/go-gin-clean-starter/modules/pensiun/repository"
	pensiunService "github.com/Caknoooo/go-gin-clean-starter/modules/pensiun/service"
	sekolahController "github.com/Caknoooo/go-gin-clean-starter/modules/sekolah/controller"
	sekolahRepo "github.com/Caknoooo/go-gin-clean-starter/modules/sekolah/repository"
	suratController "github.com/Caknoooo/go-gin-clean-starter/modules/surat/controller"
	suratRepo "github.com/Caknoooo/go-gin-clean-starter/modules/surat/repository"
	suratService "github.com/Caknoooo/go-gin-clean-starter/modules/surat/service"
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

	// Layanan Dapodik
	dapodikSvc := dapodikService.NewDapodikService(dapodikRepo.NewDapodikRepository(db), dokumenSvc, notifikasiSvc, db)
	do.Provide(injector, func(i *do.Injector) (dapodikController.DapodikController, error) {
		return dapodikController.NewDapodikController(dapodikSvc), nil
	})

	// Kinerja P3K Paruh Waktu
	kinerjaSvc := kinerjaService.NewKinerjaService(kinerjaRepo.NewKinerjaRepository(db), dokumenSvc, notifikasiSvc, db)
	do.Provide(injector, func(i *do.Injector) (kinerjaController.KinerjaController, error) {
		return kinerjaController.NewKinerjaController(kinerjaSvc), nil
	})

	// Analisis Jabatan
	anjabSvc := anjabService.NewAnjabService(anjabRepo.NewAnjabRepository(db), notifikasiSvc, db)
	do.Provide(injector, func(i *do.Injector) (anjabController.AnjabController, error) {
		return anjabController.NewAnjabController(anjabSvc), nil
	})

	// Pensiun & Kenaikan Pangkat
	pensiunSvc := pensiunService.NewPensiunService(pensiunRepo.NewPensiunRepository(db), dokumenSvc, notifikasiSvc, db)
	do.Provide(injector, func(i *do.Injector) (pensiunController.PensiunController, error) {
		return pensiunController.NewPensiunController(pensiunSvc), nil
	})
	kpSvc := kpService.NewKenaikanPangkatService(kpRepo.NewKenaikanPangkatRepository(db), dokumenSvc, notifikasiSvc, db)
	do.Provide(injector, func(i *do.Injector) (kpController.KenaikanPangkatController, error) {
		return kpController.NewKenaikanPangkatController(kpSvc), nil
	})

	// Sikap Disiplin
	disiplinSvc := disiplinService.NewDisiplinService(disiplinRepo.NewDisiplinRepository(db), dokumenSvc, notifikasiSvc, st, db)
	do.Provide(injector, func(i *do.Injector) (disiplinController.DisiplinController, error) {
		return disiplinController.NewDisiplinController(disiplinSvc), nil
	})

	// Generator Surat
	suratSvc := suratService.NewSuratService(suratRepo.NewSuratRepository(db), dokumenSvc, notifikasiSvc, db)
	do.Provide(injector, func(i *do.Injector) (suratController.SuratController, error) {
		return suratController.NewSuratController(suratSvc), nil
	})

	// Arsip kepegawaian
	arsipSvc := arsipService.NewArsipService(arsipRepo.NewArsipRepository(db), dokumenSvc, db)
	do.Provide(injector, func(i *do.Injector) (arsipController.ArsipController, error) {
		return arsipController.NewArsipController(arsipSvc), nil
	})

	// Dasbor lintas modul
	do.Provide(injector, func(i *do.Injector) (dasborService.DasborService, error) {
		return dasborService.NewDasborService(dasborRepo.NewDasborRepository(db)), nil
	})
	do.Provide(injector, func(i *do.Injector) (dasborController.DasborController, error) {
		return dasborController.NewDasborController(do.MustInvoke[dasborService.DasborService](i)), nil
	})
}
