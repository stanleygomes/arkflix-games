package profiles_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stanleygomes/arkflix-games/app/internal/profiles"
)

func TestMigrationKeepsRootAsMain(t *testing.T) {
	dir := t.TempDir()
	s := &profiles.Store{Dir: dir}
	all, err := s.All()
	if err != nil || len(all) != 1 || all[0].ID != profiles.Main {
		t.Fatalf("sem arquivo vira um perfil principal: %+v %v", all, err)
	}
	if s.DataDir(profiles.Main) != dir {
		t.Fatal("o principal usa a pasta raiz (dados de antes continuam)")
	}
	if _, err := os.Stat(filepath.Join(dir, "profiles.json")); err == nil {
		t.Fatal("só ler não grava nada")
	}
}

func TestLifecycleAndPIN(t *testing.T) {
	dir := t.TempDir()
	s := &profiles.Store{Dir: dir}
	ana, err := s.Create("  Ana   Clara ", "cat", profiles.Colors[0])
	if err != nil || ana.Name != "Ana Clara" {
		t.Fatalf("criar: %+v %v", ana, err)
	}
	if st, err := os.Stat(s.DataDir(ana.ID)); err != nil || !st.IsDir() {
		t.Fatal("perfil novo ganha pasta própria")
	}
	for _, bad := range [][3]string{{"ana clara", "cat", profiles.Colors[0]}, {"", "cat", profiles.Colors[0]}, {"X", "pikachu", profiles.Colors[0]}, {"X", "cat", "#000"}, {strings.Repeat("a", 21), "cat", profiles.Colors[0]}} {
		if _, err := s.Create(bad[0], bad[1], bad[2]); err == nil {
			t.Fatalf("devia recusar %v", bad)
		}
	}
	if _, err := s.SetPIN(ana.ID, "", "12a4"); err == nil {
		t.Fatal("PIN com letra")
	}
	if _, err := s.SetPIN(ana.ID, "", "1234"); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(filepath.Join(dir, "profiles.json"))
	if strings.Contains(string(raw), "1234") {
		t.Fatal("PIN não pode ficar em texto puro")
	}
	if s.Check(ana.ID, "0000") != profiles.ErrPIN || s.Check(ana.ID, "1234") != nil {
		t.Fatal("conferir PIN")
	}
	if _, err := s.SetPIN(ana.ID, "9999", ""); err != profiles.ErrPIN {
		t.Fatal("trocar PIN pede o atual")
	}
	if err := s.Delete(ana.ID, "0000"); err != profiles.ErrPIN {
		t.Fatal("apagar perfil com PIN pede o PIN")
	}
	if err := s.Delete(profiles.Main, ""); err == nil {
		t.Fatal("principal não se apaga")
	}
	if _, err := s.SetKids(profiles.Main, true); err == nil {
		t.Fatal("principal não vira infantil")
	}
	os.WriteFile(filepath.Join(s.DataDir(ana.ID), "state.json"), []byte("{}"), 0o600)
	if err := s.Delete(ana.ID, "1234"); err != nil {
		t.Fatal(err)
	}
	trash, _ := filepath.Glob(filepath.Join(dir, "profiles", "apagados", ana.ID+"-*", "state.json"))
	if len(trash) != 1 {
		t.Fatal("pasta do perfil apagado vai pra profiles/apagados")
	}
	again := &profiles.Store{Dir: dir}
	if all, _ := again.All(); len(all) != 1 {
		t.Fatalf("persistência: %+v", all)
	}
}

func TestLimit(t *testing.T) {
	s := &profiles.Store{Dir: t.TempDir()}
	for i := 1; i < profiles.Max; i++ {
		if _, err := s.Create(string(rune('A'+i)), "ghost", profiles.Colors[1]); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := s.Create("Z", "ghost", profiles.Colors[1]); err == nil {
		t.Fatal("limite de perfis")
	}
}
