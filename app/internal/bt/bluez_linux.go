package bt

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"sync"

	"github.com/godbus/dbus/v5"
)

const (
	bluezBus    = "org.bluez"
	ifAdapter   = "org.bluez.Adapter1"
	ifDevice    = "org.bluez.Device1"
	ifBattery   = "org.bluez.Battery1"
	ifAgentMgr  = "org.bluez.AgentManager1"
	ifAgent     = "org.bluez.Agent1"
	ifProps     = "org.freedesktop.DBus.Properties"
	agentPath   = dbus.ObjectPath("/arkflix/agent")
	agentIOCaps = "NoInputNoOutput"
)

// managed é a resposta do GetManagedObjects: caminho -> interface -> propriedades.
type managed = map[dbus.ObjectPath]map[string]map[string]dbus.Variant

// bluez fala com o BlueZ pelo barramento do sistema. A conexão abre na
// primeira chamada (abrir o app sem Bluetooth não custa nada).
type bluez struct {
	mu    sync.Mutex
	conn  *dbus.Conn
	agent bool // agente NoInputNoOutput já registrado nesta conexão
}

// New: no Linux, o BlueZ (se o serviço não estiver rodando, Adapter diz o motivo).
func New() Backend { return &bluez{} }

func (b *bluez) bus() (*dbus.Conn, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.conn != nil && b.conn.Connected() {
		return b.conn, nil
	}
	conn, err := dbus.ConnectSystemBus()
	if err != nil {
		return nil, errors.New("não deu pra falar com o Bluetooth do sistema (D-Bus)")
	}
	b.conn, b.agent = conn, false
	return conn, nil
}

func (b *bluez) objects() (*dbus.Conn, managed, error) {
	conn, err := b.bus()
	if err != nil {
		return nil, nil, err
	}
	var objs managed
	err = conn.Object(bluezBus, "/").Call("org.freedesktop.DBus.ObjectManager.GetManagedObjects", 0).Store(&objs)
	if err != nil {
		return nil, nil, errors.New("o serviço de Bluetooth (bluetoothd) não está rodando")
	}
	return conn, objs, nil
}

// adapterPath: o primeiro adaptador, preferindo um ligado (hci0 na maioria).
func adapterPath(objs managed) (dbus.ObjectPath, bool) {
	var first dbus.ObjectPath
	for path, ifs := range objs {
		props, ok := ifs[ifAdapter]
		if !ok {
			continue
		}
		if boolProp(props, "Powered") {
			return path, true
		}
		if first == "" || path < first {
			first = path
		}
	}
	return first, first != ""
}

func (b *bluez) Adapter() Adapter {
	_, objs, err := b.objects()
	if err != nil {
		return Adapter{Reason: err.Error()}
	}
	path, ok := adapterPath(objs)
	if !ok {
		return Adapter{Reason: "nenhum adaptador Bluetooth encontrado neste computador"}
	}
	props := objs[path][ifAdapter]
	return Adapter{Supported: true, Powered: boolProp(props, "Powered"), Discovering: boolProp(props, "Discovering")}
}

func (b *bluez) Devices() ([]Device, error) {
	_, objs, err := b.objects()
	if err != nil {
		return nil, err
	}
	return devicesFrom(objs), nil
}

// devicesFrom monta a lista do adaptador em uso. Sem nome e não pareado
// (beacons BLE da vizinhança) fica de fora.
func devicesFrom(objs managed) []Device {
	out := []Device{}
	ad, ok := adapterPath(objs)
	if !ok {
		return out
	}
	for path, ifs := range objs {
		props, ok := ifs[ifDevice]
		if !ok || !strings.HasPrefix(string(path), string(ad)+"/") {
			continue
		}
		d, ok := deviceFrom(props, ifs[ifBattery])
		if ok {
			out = append(out, d)
		}
	}
	Sort(out)
	return out
}

func deviceFrom(props, battery map[string]dbus.Variant) (Device, bool) {
	addr, err := ParseAddress(strProp(props, "Address"))
	if err != nil {
		return Device{}, false
	}
	d := Device{
		Address:   addr,
		Name:      strProp(props, "Name"),
		Paired:    boolProp(props, "Paired") || boolProp(props, "Bonded"),
		Connected: boolProp(props, "Connected"),
		Trusted:   boolProp(props, "Trusted"),
		Battery:   -1,
	}
	if d.Name == "" && !d.Paired {
		return Device{}, false
	}
	if d.Name == "" {
		d.Name = strProp(props, "Alias")
	}
	d.Gamepad = IsGamepad(d.Name, strProp(props, "Icon"))
	if v, ok := battery["Percentage"]; ok {
		if p, ok := v.Value().(byte); ok {
			d.Battery = int(p)
		}
	}
	return d, true
}

func boolProp(props map[string]dbus.Variant, name string) bool {
	v, _ := props[name].Value().(bool)
	return v
}

func strProp(props map[string]dbus.Variant, name string) string {
	v, _ := props[name].Value().(string)
	return v
}

// adapterObj: o adaptador em uso, pronto pra chamar.
func (b *bluez) adapterObj() (dbus.BusObject, error) {
	conn, objs, err := b.objects()
	if err != nil {
		return nil, err
	}
	path, ok := adapterPath(objs)
	if !ok {
		return nil, errors.New("nenhum adaptador Bluetooth encontrado neste computador")
	}
	return conn.Object(bluezBus, path), nil
}

// deviceObj acha o aparelho pelo endereço (procura em vez de montar o caminho:
// o adaptador pode não ser o hci0).
func (b *bluez) deviceObj(addr string) (dbus.BusObject, error) {
	conn, objs, err := b.objects()
	if err != nil {
		return nil, err
	}
	ad, _ := adapterPath(objs)
	for path, ifs := range objs {
		props, ok := ifs[ifDevice]
		if ok && strings.HasPrefix(string(path), string(ad)+"/") && strings.EqualFold(strProp(props, "Address"), addr) {
			return conn.Object(bluezBus, path), nil
		}
	}
	return nil, errNotFound
}

func setProp(obj dbus.BusObject, iface, name string, v any) error {
	return translate(obj.Call(ifProps+".Set", 0, iface, name, dbus.MakeVariant(v)).Err)
}

func (b *bluez) SetPowered(on bool) error {
	ad, err := b.adapterObj()
	if err != nil {
		return err
	}
	return setProp(ad, ifAdapter, "Powered", on)
}

func (b *bluez) StartDiscovery() error {
	ad, err := b.adapterObj()
	if err != nil {
		return err
	}
	err = translate(ad.Call(ifAdapter+".StartDiscovery", 0).Err)
	if errors.Is(err, errInProgress) {
		return nil // outro app já está buscando: serve igual
	}
	return err
}

func (b *bluez) StopDiscovery() error {
	ad, err := b.adapterObj()
	if err != nil {
		return err
	}
	ad.Call(ifAdapter+".StopDiscovery", 0) // não estava buscando: tudo bem
	return nil
}

// Pair pareia com o agente NoInputNoOutput (controle é "Just Works": sem PIN).
// Liga o Pairable antes: com ele desligado o BlueZ não guarda a chave.
func (b *bluez) Pair(ctx context.Context, addr string) error {
	if err := b.registerAgent(); err != nil {
		return err
	}
	if ad, err := b.adapterObj(); err == nil {
		setProp(ad, ifAdapter, "Pairable", true)
	}
	dev, err := b.deviceObj(addr)
	if err != nil {
		return err
	}
	err = translate(dev.CallWithContext(ctx, ifDevice+".Pair", 0).Err)
	if ctx.Err() != nil {
		dev.Call(ifDevice+".CancelPairing", 0)
		return errors.New("o controle não respondeu: deixe-o em modo de pareamento e tente de novo")
	}
	if errors.Is(err, errExists) {
		return nil
	}
	return err
}

func (b *bluez) SetTrusted(addr string, on bool) error {
	dev, err := b.deviceObj(addr)
	if err != nil {
		return err
	}
	return setProp(dev, ifDevice, "Trusted", on)
}

func (b *bluez) Connect(ctx context.Context, addr string) error {
	dev, err := b.deviceObj(addr)
	if err != nil {
		return err
	}
	err = translate(dev.CallWithContext(ctx, ifDevice+".Connect", 0).Err)
	if errors.Is(err, errConnected) {
		return nil
	}
	return err
}

func (b *bluez) Disconnect(addr string) error {
	dev, err := b.deviceObj(addr)
	if err != nil {
		return err
	}
	return translate(dev.Call(ifDevice+".Disconnect", 0).Err)
}

func (b *bluez) Remove(addr string) error {
	dev, err := b.deviceObj(addr)
	if err != nil {
		return err
	}
	ad, err := b.adapterObj()
	if err != nil {
		return err
	}
	return translate(ad.Call(ifAdapter+".RemoveDevice", 0, dev.Path()).Err)
}

// registerAgent registra o agente do app (não como padrão: o do desktop segue
// cuidando do resto). O BlueZ só manda pra ele os pareamentos que o app pediu.
func (b *bluez) registerAgent() error {
	conn, err := b.bus()
	if err != nil {
		return err
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.agent {
		return nil
	}
	if err := conn.Export(agent{}, agentPath, ifAgent); err != nil {
		return errors.New("não deu pra preparar o pareamento: " + err.Error())
	}
	err = translate(conn.Object(bluezBus, "/org/bluez").Call(ifAgentMgr+".RegisterAgent", 0, agentPath, agentIOCaps).Err)
	if err != nil && !errors.Is(err, errExists) {
		return err
	}
	b.agent = true
	return nil
}

// agent aceita tudo: só recebe pedidos dos pareamentos que o próprio app
// iniciou, e controle não tem tela nem teclado pra conferir código.
type agent struct{}

func (agent) Release() *dbus.Error                                       { return nil }
func (agent) RequestPinCode(dbus.ObjectPath) (string, *dbus.Error)       { return "0000", nil }
func (agent) DisplayPinCode(dbus.ObjectPath, string) *dbus.Error         { return nil }
func (agent) RequestPasskey(dbus.ObjectPath) (uint32, *dbus.Error)       { return 0, nil }
func (agent) DisplayPasskey(dbus.ObjectPath, uint32, uint16) *dbus.Error { return nil }
func (agent) RequestConfirmation(dbus.ObjectPath, uint32) *dbus.Error    { return nil }
func (agent) RequestAuthorization(dbus.ObjectPath) *dbus.Error           { return nil }
func (agent) AuthorizeService(dbus.ObjectPath, string) *dbus.Error       { return nil }
func (agent) Cancel() *dbus.Error                                        { return nil }

// OpenSettings: o painel de Bluetooth do desktop que estiver instalado.
func (b *bluez) OpenSettings() error { return openSettings() }

// apps de Bluetooth conhecidos, na ordem de preferência
var settingsApps = [][]string{
	{"gnome-control-center", "bluetooth"},
	{"blueman-manager"},
	{"systemsettings", "kcm_bluetooth"},
	{"kcmshell6", "kcm_bluetooth"},
	{"kcmshell5", "bluedevilglobal"},
	{"blueberry"},
}

func openSettings() error {
	for _, app := range settingsApps {
		if _, err := exec.LookPath(app[0]); err == nil {
			return exec.Command(app[0], app[1:]...).Start()
		}
	}
	return errors.New("nenhum app de Bluetooth encontrado (instale o blueman ou use o do seu desktop)")
}
