package core_test

import (
	"testing"
)

func TestStorage(t *testing.T) {
	h := setup(t)
	st := h.svc.Storage()
	if st.Total <= 0 || st.Free <= 0 || st.Free > st.Total {
		t.Fatalf("disco: total %d livre %d", st.Total, st.Free)
	}
	boot, _ := h.svc.Bootstrap()
	var games int64
	for _, g := range boot.Games {
		games += g.Size
	}
	if st.Games != games || st.GameCount != len(boot.Games) {
		t.Fatalf("jogos: %d bytes/%d, esperado %d/%d", st.Games, st.GameCount, games, len(boot.Games))
	}
	var sum int64
	for i, c := range st.ByConsole {
		sum += c.Bytes
		if i > 0 && c.Bytes > st.ByConsole[i-1].Bytes {
			t.Fatal("consoles do maior pro menor")
		}
		if c.Name == "" || c.Count == 0 {
			t.Fatalf("console: %+v", c)
		}
	}
	if sum != st.Games {
		t.Fatal("soma por console = total de jogos")
	}
	if st.Saves <= 0 || len(st.SaveDirs) < 5 {
		t.Fatalf("saves do ambiente falso (states e .srm): %+v", st.SaveDirs)
	}
	for _, d := range st.SaveDirs {
		if d.Label == "Perfis extras" {
			t.Fatal("sem perfis extras, a linha não aparece")
		}
	}
	if st.App <= 0 {
		t.Fatal("configurações e estado do app ocupam algo")
	}
}
