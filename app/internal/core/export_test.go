package core

import "time"

// SetQuitGrace encurta a espera do "fechar jogo" nos testes.
func SetQuitGrace(d time.Duration) { quitGrace = d }

// Conflict expõe a regra de conflito de saves.
var Conflict = conflict
