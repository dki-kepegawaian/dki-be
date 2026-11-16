package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/Caknoooo/go-gin-clean-starter/database/entities"
	"github.com/Caknoooo/go-gin-clean-starter/modules/anjab/dto"
	"github.com/Caknoooo/go-gin-clean-starter/modules/anjab/hitung"
	"github.com/xuri/excelize/v2"
)

// Monitoring (A6) memakai hasil yang sudah ditetapkan (versi kondisi riil); sekolah yang
// belum ditetapkan hanya dihitung jumlahnya.
func (s *anjabService) Monitoring(ctx context.Context, actor entities.User, id, mapel string) (dto.MonitoringResponse, error) {
	p, err := s.repo.GetPeriode(ctx, nil, actor, id, false)
	if err != nil {
		return dto.MonitoringResponse{}, err
	}
	semua, err := s.repo.ListByPeriode(ctx, nil, p.ID, nil)
	if err != nil {
		return dto.MonitoringResponse{}, err
	}
	sort.Slice(semua, func(i, j int) bool { return semua[i].Sekolah.Nama < semua[j].Sekolah.Nama })

	res := dto.MonitoringResponse{
		PeriodeID: p.ID.String(), TahunAjaran: p.TahunAjaran, TotalSekolah: len(semua),
		Sekolah: []dto.MonitoringSekolah{},
	}
	var selisih []hitung.SekolahSelisih
	for _, a := range semua {
		row := dto.MonitoringSekolah{AnjabSekolahID: a.ID.String(), Sekolah: dto.SekolahFromEntity(a.Sekolah), Status: a.Status}
		if a.Status == entities.AnjabStatusDitetapkan && a.Hasil != "" {
			var h hitung.Hasil
			if err := json.Unmarshal([]byte(a.Hasil), &h); err != nil {
				return res, err
			}
			res.SekolahDitetapkan++
			row.Kurang, row.Lebih = h.Riil.Kurang, h.Riil.Lebih
			res.KekuranganGuru += h.Riil.Kurang
			res.KelebihanGuru += h.Riil.Lebih
			selisih = append(selisih, hitung.SekolahSelisih{
				SekolahID: a.SekolahID.String(), Nama: a.Sekolah.Nama, Kecamatan: a.Sekolah.Kecamatan, Mapel: h.Riil.Mapel,
			})
		}
		res.Sekolah = append(res.Sekolah, row)
	}
	res.Rekomendasi = []hitung.Rekomendasi{}
	for _, r := range hitung.Rekomendasikan(selisih) {
		if mapel == "" || hitung.KunciMapel(r.Mapel) == hitung.KunciMapel(mapel) {
			res.Rekomendasi = append(res.Rekomendasi, r)
		}
	}
	return res, nil
}

func (s *anjabService) MonitoringExcel(ctx context.Context, actor entities.User, id string) ([]byte, string, error) {
	m, err := s.Monitoring(ctx, actor, id, "")
	if err != nil {
		return nil, "", err
	}
	f := excelize.NewFile()
	defer f.Close()
	const rekap, saran = "Rekap Sekolah", "Rekomendasi"
	if err := f.SetSheetName("Sheet1", rekap); err != nil {
		return nil, "", err
	}
	if _, err := f.NewSheet(saran); err != nil {
		return nil, "", err
	}
	tulis := func(sheet string, baris int, nilai []any) error {
		return f.SetSheetRow(sheet, fmt.Sprintf("A%d", baris), &nilai)
	}
	if err := tulis(rekap, 1, []any{"NPSN", "Sekolah", "Jenjang", "Kecamatan", "Status", "Kekurangan (riil)", "Kelebihan (riil)"}); err != nil {
		return nil, "", err
	}
	for i, r := range m.Sekolah {
		if err := tulis(rekap, i+2, []any{r.Sekolah.NPSN, r.Sekolah.Nama, r.Sekolah.Jenjang, r.Sekolah.Kecamatan, r.Status, r.Kurang, r.Lebih}); err != nil {
			return nil, "", err
		}
	}
	if err := tulis(saran, 1, []any{"Mapel", "Guru lebih di", "Lebih", "Kecamatan asal", "Kurang di", "Kurang", "Kecamatan tujuan", "Jumlah guru", "Kecamatan sama"}); err != nil {
		return nil, "", err
	}
	for i, r := range m.Rekomendasi {
		sama := "Tidak"
		if r.KecamatanSama {
			sama = "Ya"
		}
		if err := tulis(saran, i+2, []any{r.Mapel, r.Dari.Nama, r.Dari.Jumlah, r.Dari.Kecamatan, r.Ke.Nama, r.Ke.Jumlah, r.Ke.Kecamatan, r.Jumlah, sama}); err != nil {
			return nil, "", err
		}
	}
	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, "", err
	}
	return buf.Bytes(), "rekomendasi_anjab_" + strings.ReplaceAll(m.TahunAjaran, "/", "-") + ".xlsx", nil
}
