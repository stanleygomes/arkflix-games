package core_test

import (
	"testing"

	"github.com/stanleygomes/arkflix-games/app/internal/lists"
)

func TestSmartList(t *testing.T) {
	h := setup(t)
	l, err := h.svc.CreateSmartList("Nunca joguei", lists.Rule{Source: "collection:all", Filters: []string{"never"}, Sort: "yearDesc"})
	if err != nil || l.Rule == nil || l.Rule.Sort != "yearDesc" {
		t.Fatalf("criar: %+v %v", l, err)
	}
	if _, err := h.svc.SetInList(l.ID, h.game(t, "Super Mario World").ID, true); err != lists.ErrSmart {
		t.Fatalf("jogo na mão em lista automática: %v", err)
	}
	if _, err := h.svc.CreateSmartList("Sem filtro", lists.Rule{Source: "collection:all"}); err == nil {
		t.Fatal("regra sem filtro devia falhar")
	}
	b, _ := h.svc.Bootstrap()
	if len(b.Lists) != 1 || b.Lists[0].Rule == nil || b.Lists[0].Rule.Filters[0] != "never" {
		t.Fatalf("boot: %+v", b.Lists)
	}
	if _, err := h.svc.RenameList(l.ID, "Pra começar"); err != nil {
		t.Fatal(err)
	}
	if err := h.svc.DeleteList(l.ID); err != nil {
		t.Fatal(err)
	}
}
