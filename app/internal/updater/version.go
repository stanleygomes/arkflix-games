// Package updater acha, baixa, confere e instala a release nova do app
// (GitHub Releases). Rede, sistema e comandos entram por campo: os testes
// trocam tudo e nunca mexem no binário que está rodando.
package updater

import (
	"strconv"
	"strings"
)

// parse lê "v1.9.3" (e "v1.9.3-7-gabc-dirty" do git describe) como
// major.minor.patch. ok = false pra "dev" e lixo.
func parse(v string) (out [3]int, ok bool) {
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	if i := strings.IndexAny(v, "-+"); i >= 0 {
		v = v[:i]
	}
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return out, false
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return out, false
		}
		out[i] = n
	}
	return out, true
}

// Newer diz se latest passa current em major.minor.patch. Build de
// desenvolvimento (v1.9.3-7-gabc) só atualiza pra uma tag maior que a base;
// "dev" nunca atualiza.
func Newer(latest, current string) bool {
	l, okL := parse(latest)
	c, okC := parse(current)
	if !okL || !okC {
		return false
	}
	for i := range l {
		if l[i] != c[i] {
			return l[i] > c[i]
		}
	}
	return false
}
