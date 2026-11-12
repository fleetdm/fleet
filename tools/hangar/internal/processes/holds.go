package processes

import (
	"fmt"
	"net/url"
	"slices"
	"strconv"
	"strings"
)

// HoldSpec claims a slot for one agent job or assign.
type HoldSpec struct {
	Slot       string   // "s2"
	Owner      string   // job id or "assign:<n>"
	Project    string   // compose project of the slot
	ServerPort uint16   // slot's fleet serve port, for perf-* matching
	Replace    []string // running <slot>:* ids this holder may stop or replace
	Allow      []string // running <slot>:* ids that may keep running
}

// Hold claims h.Slot for h.Owner. It fails if the slot is held, if any
// <slot>:* process is running or stopping that isn't in Replace or Allow, if
// a Docker op is in flight on Project, or if a running perf-* process has
// --server_url on ServerPort. Checked atomically under stateMu.
//
// While a slot is held, StartInvocation, StopAs, and RestartAs refuse its ids
// unless the owner is the holder, StartInvocation refuses perf-* runs aimed at
// ServerPort, and DockerComposeDownAs/RestartAs refuse Project unless the
// owner is the holder. ShutdownNow bypasses holds.
func (m *Manager) Hold(h HoldSpec) error {
	m.stateMu.Lock()
	err := m.admitLocked(h)
	if err == nil {
		m.holds[h.Slot] = h
	}
	m.stateMu.Unlock()
	if err == nil {
		m.emitSlots()
	}
	return err
}

func (m *Manager) admitLocked(h HoldSpec) error {
	if cur, held := m.holds[h.Slot]; held {
		return fmt.Errorf("%s is held by %s", h.Slot, cur.Owner)
	}
	prefix := h.Slot + ":"
	mayRun := func(id string) bool {
		return !strings.HasPrefix(id, prefix) || slices.Contains(h.Replace, id) || slices.Contains(h.Allow, id)
	}
	for id, inv := range m.live {
		if !mayRun(id) {
			return fmt.Errorf("%s is running", id)
		}
		if h.ServerPort != 0 && strings.HasPrefix(id, "perf-") && serverURLPort(inv.args.Args) == h.ServerPort {
			return fmt.Errorf("%s targets %s's server port %d", id, h.Slot, h.ServerPort)
		}
	}
	for id, info := range m.procs {
		if (info.State == "running" || info.State == "stopping") && !mayRun(id) {
			return fmt.Errorf("%s is %s", id, info.State)
		}
	}
	if h.Project != "" && m.dockerOps[h.Project] > 0 {
		return fmt.Errorf("a docker compose operation is in flight on %s", h.Project)
	}
	return nil
}

// Release drops slot's hold if owner holds it.
func (m *Manager) Release(slot, owner string) {
	m.stateMu.Lock()
	h, held := m.holds[slot]
	released := held && h.Owner == owner
	if released {
		delete(m.holds, slot)
	}
	m.stateMu.Unlock()
	if released {
		m.emitSlots()
	}
}

// HeldBy returns slot's holder.
func (m *Manager) HeldBy(slot string) (owner string, held bool) {
	m.stateMu.Lock()
	defer m.stateMu.Unlock()
	h, held := m.holds[slot]
	return h.Owner, held
}

// Holds returns slot -> owner for every held slot.
func (m *Manager) Holds() map[string]string {
	m.stateMu.Lock()
	defer m.stateMu.Unlock()
	out := make(map[string]string, len(m.holds))
	for slot, h := range m.holds {
		out[slot] = h.Owner
	}
	return out
}

func (m *Manager) emitSlots() {
	if m.emit != nil {
		m.emit.Emit("agent:slots", nil)
	}
}

// heldErrLocked refuses a <slot>:* id whose slot is held by someone other
// than owner. Caller holds stateMu.
func (m *Manager) heldErrLocked(id, owner string) error {
	slot, _, ok := strings.Cut(id, ":")
	if !ok {
		return nil
	}
	if h, held := m.holds[slot]; held && h.Owner != owner {
		return fmt.Errorf("%s is held by %s", slot, h.Owner)
	}
	return nil
}

// refuseStartLocked applies the hold rules to a spawn. Caller holds stateMu.
func (m *Manager) refuseStartLocked(id string, a StartArgs) error {
	if err := m.heldErrLocked(id, a.Owner); err != nil {
		return err
	}
	if !strings.HasPrefix(id, "perf-") {
		return nil
	}
	port := serverURLPort(a.Args)
	for slot, h := range m.holds {
		if port != 0 && port == h.ServerPort {
			return fmt.Errorf("%s targets %s's server port %d, held by %s", id, slot, port, h.Owner)
		}
	}
	return nil
}

// beginDockerOp registers a compose operation on project so Hold sees it in
// flight, refusing a project held by someone other than owner. Call end when
// the operation finishes.
func (m *Manager) beginDockerOp(project, owner string) (end func(), err error) {
	m.stateMu.Lock()
	defer m.stateMu.Unlock()
	for slot, h := range m.holds {
		if h.Project == project && h.Owner != owner {
			return nil, fmt.Errorf("%s (project %s) is held by %s", slot, project, h.Owner)
		}
	}
	return m.trackDockerOpLocked(project), nil
}

// trackDockerOpLocked counts a compose op on project as in flight until end
// runs. Caller holds stateMu.
func (m *Manager) trackDockerOpLocked(project string) (end func()) {
	m.dockerOps[project]++
	return func() {
		m.stateMu.Lock()
		defer m.stateMu.Unlock()
		m.dockerOps[project]--
		if m.dockerOps[project] <= 0 {
			delete(m.dockerOps, project)
		}
	}
}

// serverURLPort returns the port of a --server_url (or -server_url) flag in
// args, or 0 when there's none or it has no explicit port.
func serverURLPort(args []string) uint16 {
	for i, a := range args {
		name, val, hasVal := strings.Cut(strings.TrimLeft(a, "-"), "=")
		if name != "server_url" || a == name {
			continue
		}
		if !hasVal {
			if i+1 >= len(args) {
				return 0
			}
			val = args[i+1]
		}
		u, err := url.Parse(val)
		if err != nil {
			return 0
		}
		p, _ := strconv.ParseUint(u.Port(), 10, 16)
		return uint16(p)
	}
	return 0
}
