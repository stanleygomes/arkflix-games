package isoinfo_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stanleygomes/arkflix-games/app/internal/fixture"
	"github.com/stanleygomes/arkflix-games/app/internal/isoinfo"
)

func TestPSP(t *testing.T) {
	info, err := isoinfo.Read(bytes.NewReader(fixture.PSPISO("ULUS10041", "Lara Croft")))
	if err != nil {
		t.Fatal(err)
	}
	want := isoinfo.Info{System: "psp", Serial: "ULUS10041", Version: "1.00", Title: "Lara Croft"}
	if info != want {
		t.Fatalf("got %+v, want %+v", info, want)
	}
}

func TestPS2(t *testing.T) {
	path := filepath.Join(t.TempDir(), "fifa.iso")
	if err := os.WriteFile(path, fixture.PS2ISO("SLUS-21431"), 0o644); err != nil {
		t.Fatal(err)
	}
	info, err := isoinfo.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.System != "ps2" || info.Serial != "SLUS-21431" {
		t.Fatalf("got %+v", info)
	}
}

func TestNotISO(t *testing.T) {
	if _, err := isoinfo.Read(bytes.NewReader(make([]byte, 40000))); err == nil {
		t.Fatal("aceitou lixo como ISO")
	}
	if _, err := isoinfo.Read(bytes.NewReader([]byte("curto"))); err == nil {
		t.Fatal("aceitou arquivo curto")
	}
}

func TestISOWithoutGame(t *testing.T) {
	img := fixture.BuildISO(map[string][]byte{"README.TXT": []byte("oi")})
	if _, err := isoinfo.Read(bytes.NewReader(img)); err == nil {
		t.Fatal("ISO sem PARAM.SFO nem SYSTEM.CNF devia falhar")
	}
}

func TestPS2SerialVariants(t *testing.T) {
	cases := map[string]string{
		"BOOT2 = cdrom0:\\SLES_123.45;1":   "SLES-12345",
		"boot2=cdrom0:\\SCUS_971.13;1\r\n": "SCUS-97113",
	}
	for cnf, want := range cases {
		got, err := isoinfo.PS2Serial([]byte(cnf))
		if err != nil || got != want {
			t.Errorf("%q: got %q %v, want %q", cnf, got, err, want)
		}
	}
	if _, err := isoinfo.PS2Serial([]byte("BOOT = cdrom:\\SLPS_000.01")); err == nil {
		t.Error("BOOT (PS1) não devia virar serial de PS2")
	}
}

func TestSFOTruncated(t *testing.T) {
	sfo := fixture.SFO(map[string]string{"DISC_ID": "ULUS10041"})
	if _, err := isoinfo.ParseSFO(sfo[:len(sfo)-4]); err == nil {
		t.Fatal("SFO truncado devia falhar")
	}
}

func TestPS1RawBin(t *testing.T) {
	info, err := isoinfo.Read(bytes.NewReader(fixture.PS1Bin("SCUS-94900")))
	if err != nil || info.System != "psx" || info.Serial != "SCUS-94900" {
		t.Fatalf("PS1 .bin: %+v %v", info, err)
	}
	// PS2 em .bin modo 1 também
	iso := fixture.PS2ISO("SLUS-21431")
	info, err = isoinfo.Read(bytes.NewReader(fixture.RawBin(iso, false)))
	if err != nil || info.Serial != "SLUS-21431" {
		t.Fatalf("PS2 .bin modo 1: %+v %v", info, err)
	}
	if s, err := isoinfo.PS1Serial([]byte("BOOT=cdrom:SLES-123.45;1")); err != nil || s != "SLES-12345" {
		t.Fatalf("variação do BOOT: %q %v", s, err)
	}
}
