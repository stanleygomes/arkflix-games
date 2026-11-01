// Package httpapi expõe o core.Service por HTTP: o mesmo método que o Wails
// liga no desktop vira POST /api/call/<Método> (argumentos em array JSON).
// Eventos saem por Server-Sent Events em /api/events. É o modo web (testes de
// ponta a ponta, navegador) e a base pra rodar a plataforma num servidor.
package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"path"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/stanleygomes/arkflix-games/app/internal/core"
	"github.com/stanleygomes/arkflix-games/app/internal/gamepad"
)

// Broker distribui eventos pros clientes SSE. Implementa core.Emitter.
type Broker struct {
	mu      sync.Mutex
	clients map[chan []byte]struct{}
}

// NewBroker cria o distribuidor de eventos.
func NewBroker() *Broker { return &Broker{clients: map[chan []byte]struct{}{}} }

// Emit manda o evento pra todos os clientes conectados.
func (b *Broker) Emit(name string, data any) {
	msg, err := json.Marshal(map[string]any{"name": name, "data": data})
	if err != nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	for c := range b.clients {
		select {
		case c <- msg:
		default: // cliente lento: perde o evento em vez de travar o app
		}
	}
}

func (b *Broker) serve(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "sem streaming", http.StatusInternalServerError)
		return
	}
	c := make(chan []byte, 64)
	b.mu.Lock()
	b.clients[c] = struct{}{}
	b.mu.Unlock()
	defer func() {
		b.mu.Lock()
		delete(b.clients, c)
		b.mu.Unlock()
	}()
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	fmt.Fprint(w, ": ok\n\n")
	flusher.Flush()
	ping := time.NewTicker(15 * time.Second)
	defer ping.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case msg := <-c:
			fmt.Fprintf(w, "data: %s\n\n", msg)
			flusher.Flush()
		case <-ping.C:
			fmt.Fprint(w, ": ping\n\n")
			flusher.Flush()
		}
	}
}

// Options configura o servidor.
type Options struct {
	Assets fs.FS // interface compilada (web/dist)
	Test   bool  // libera /api/test/* (injetar botão do controle)
	Pads   *gamepad.Manager
}

// Métodos que não fazem sentido pela API.
var hidden = map[string]bool{"WaitReady": true}

var errType = reflect.TypeOf((*error)(nil)).Elem()

// call executa um método do serviço pelo nome com argumentos JSON.
func call(svc any, name string, body io.Reader) (any, error) {
	if hidden[name] {
		return nil, errNotFound
	}
	m := reflect.ValueOf(svc).MethodByName(name)
	if !m.IsValid() {
		return nil, errNotFound
	}
	var raw []json.RawMessage
	data, err := io.ReadAll(io.LimitReader(body, 4<<20))
	if err != nil {
		return nil, err
	}
	if len(strings.TrimSpace(string(data))) > 0 {
		if err := json.Unmarshal(data, &raw); err != nil {
			return nil, badRequest("argumentos devem ser um array JSON")
		}
	}
	t := m.Type()
	if len(raw) != t.NumIn() {
		return nil, badRequest(fmt.Sprintf("%s espera %d argumento(s), veio %d", name, t.NumIn(), len(raw)))
	}
	args := make([]reflect.Value, t.NumIn())
	for i := range args {
		v := reflect.New(t.In(i))
		if err := json.Unmarshal(raw[i], v.Interface()); err != nil {
			return nil, badRequest(fmt.Sprintf("argumento %d inválido: %v", i+1, err))
		}
		args[i] = v.Elem()
	}
	out := m.Call(args)
	var result any
	for _, v := range out {
		if v.Type().Implements(errType) {
			if !v.IsNil() {
				return nil, v.Interface().(error)
			}
			continue
		}
		result = v.Interface()
	}
	return result, nil
}

var errNotFound = errors.New("método não existe")

type badRequest string

func (b badRequest) Error() string { return string(b) }

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// Handler monta as rotas.
func Handler(svc *core.Service, b *Broker, opt Options) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/call/{method}", func(w http.ResponseWriter, r *http.Request) {
		// cabeçalho próprio: site de fora não consegue mandar sem CORS (que não liberamos)
		if r.Header.Get("X-Arkflix") != "1" {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "cabeçalho X-Arkflix ausente"})
			return
		}
		result, err := call(svc, r.PathValue("method"), r.Body)
		var bad badRequest
		switch {
		case errors.Is(err, errNotFound):
			writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		case errors.As(err, &bad):
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		case err != nil:
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		default:
			writeJSON(w, http.StatusOK, map[string]any{"result": result})
		}
	})
	mux.HandleFunc("GET /api/events", b.serve)
	mux.Handle("GET /media/", core.Media(svc))
	if opt.Test && opt.Pads != nil {
		mux.HandleFunc("POST /api/test/pad", func(w http.ResponseWriter, r *http.Request) {
			var e gamepad.Event
			if err := json.NewDecoder(r.Body).Decode(&e); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
				return
			}
			opt.Pads.Inject(e)
			writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
		})
	}
	if opt.Assets != nil {
		files := http.FileServerFS(opt.Assets)
		mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
			name := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
			if name == "" {
				name = "index.html"
			}
			if _, err := fs.Stat(opt.Assets, name); err != nil {
				r.URL.Path = "/" // rota da interface: devolve o index
			}
			files.ServeHTTP(w, r)
		})
	}
	return mux
}
