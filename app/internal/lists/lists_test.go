package lists_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/stanleygomes/arkflix-games/app/internal/lists"
)

func TestLifecycle(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lists.json")
	s := &lists.Store{Path: path}
	if all, err := s.All(); err != nil || len(all) != 0 {
		t.Fatalf("vazio: %v %v", all, err)
	}
	futebol, err := s.Create("  Futebol   da  galera ")
	if err != nil || futebol.Name != "Futebol da galera" {
		t.Fatalf("criar: %+v %v", futebol, err)
	}
	if _, err := s.Create("futebol DA galera"); err == nil || !strings.Contains(err.Error(), "já existe") {
		t.Fatalf("nome repetido devia falhar: %v", err)
	}
	if _, err := s.Create("   "); err == nil {
		t.Fatal("nome vazio devia falhar")
	}
	if _, err := s.Create(strings.Repeat("á", 41)); err == nil {
		t.Fatal("nome com mais de 40 letras devia falhar")
	}
	s.Set(futebol.ID, "fifa", true)
	s.Set(futebol.ID, "fifa", true) // de novo: não duplica
	l, _ := s.Set(futebol.ID, "iss", true)
	if len(l.Games) != 2 {
		t.Fatalf("jogos: %v", l.Games)
	}
	l, _ = s.Set(futebol.ID, "fifa", false)
	if len(l.Games) != 1 || l.Games[0] != "iss" {
		t.Fatalf("tirar: %v", l.Games)
	}
	if _, err := s.Rename(futebol.ID, "Bola"); err != nil {
		t.Fatal(err)
	}
	other, _ := s.Create("Dois jogadores")
	if _, err := s.Rename(other.ID, "bola"); err == nil {
		t.Fatal("renomear pra nome existente devia falhar")
	}
	again := &lists.Store{Path: path}
	all, _ := again.All()
	if len(all) != 2 || all[0].Name != "Bola" || all[0].Games[0] != "iss" {
		t.Fatalf("persistência: %+v", all)
	}
	if err := s.Delete(futebol.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Set(futebol.ID, "x", true); err != lists.ErrNotFound {
		t.Fatalf("lista apagada: %v", err)
	}
	all, _ = s.All()
	if len(all) != 1 {
		t.Fatalf("depois de apagar: %+v", all)
	}
}

func TestSmart(t *testing.T) {
	s := &lists.Store{Path: filepath.Join(t.TempDir(), "lists.json")}
	rpg, err := s.CreateSmart("RPG de SNES", lists.Rule{Source: "system:snes", Filters: []string{"genre:RPG", " never ", "genre:RPG"}})
	if err != nil {
		t.Fatal(err)
	}
	if rpg.Rule == nil || rpg.Rule.Sort != "name" || len(rpg.Rule.Filters) != 2 || rpg.Rule.Filters[1] != "never" {
		t.Fatalf("regra limpa: %+v", rpg.Rule)
	}
	if _, err := s.Set(rpg.ID, "smw", true); err != lists.ErrSmart {
		t.Fatalf("pôr jogo na mão devia falhar: %v", err)
	}
	if _, err := s.CreateSmart("rpg de snes", lists.Rule{Source: "system:snes", Filters: []string{"fav"}}); err == nil {
		t.Fatal("nome repetido devia falhar")
	}
	bad := []lists.Rule{
		{Source: "list:abc", Filters: []string{"fav"}},
		{Source: "system:../x", Filters: []string{"fav"}},
		{Source: "system:snes"},
		{Source: "system:snes", Filters: []string{"hack"}},
		{Source: "system:snes", Filters: []string{"decade:1995"}},
		{Source: "system:snes", Filters: []string{"players:9"}},
		{Source: "system:snes", Filters: []string{"fav"}, Sort: "random"},
	}
	for _, r := range bad {
		if _, err := s.CreateSmart("Ruim", r); err == nil {
			t.Errorf("regra inválida aceita: %+v", r)
		}
	}
	if _, err := s.Rename(rpg.ID, "RPGs"); err != nil {
		t.Fatal(err)
	}
	all, _ := (&lists.Store{Path: s.Path}).All()
	if len(all) != 1 || all[0].Name != "RPGs" || all[0].Rule.Source != "system:snes" || len(all[0].Games) != 0 {
		t.Fatalf("persistência: %+v", all)
	}
	if err := s.Delete(rpg.ID); err != nil {
		t.Fatal(err)
	}
}
