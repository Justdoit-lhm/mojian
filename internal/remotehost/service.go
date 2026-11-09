// Package remotehost owns the app's SSH hosts, remembered remote projects
// and the connection pool. It is independent of the window; the app asks
// it for connections and marks which ones are still in use.
package remotehost

import "mojian/internal/remote"

// Host is one SSH server the user added.
type Host struct {
	Name    string `json:"name"`
	Address string `json:"address"`
}

// Project is a remote folder remembered for the project list.
type Project struct {
	Host string `json:"host"`
	Path string `json:"path"`
}

// Service is the host list, remembered projects and live connections.
type Service struct {
	hosts    []Host
	projects []Project
	conns    map[string]*remote.Conn
}

// New returns an empty service.
func New() *Service {
	return &Service{conns: map[string]*remote.Conn{}}
}

// Hosts is the host list; the returned slice is the service's own.
func (s *Service) Hosts() []Host { return s.hosts }

// SetHosts replaces the host list.
func (s *Service) SetHosts(hosts []Host) { s.hosts = hosts }

// AddHost appends a host unless its address is already present.
func (s *Service) AddHost(name, address string) bool {
	if address == "" {
		return false
	}
	for _, h := range s.hosts {
		if h.Address == address {
			return false
		}
	}
	s.hosts = append(s.hosts, Host{Name: name, Address: address})
	return true
}

// RemoveHost drops the host with address and its remembered projects.
func (s *Service) RemoveHost(address string) {
	kept := s.hosts[:0]
	for _, h := range s.hosts {
		if h.Address != address {
			kept = append(kept, h)
		}
	}
	s.hosts = kept
	s.RemoveProjects(address)
}

// Projects is the remembered remote project list; the returned slice is
// the service's own.
func (s *Service) Projects() []Project { return s.projects }

// SetProjects replaces the remembered projects.
func (s *Service) SetProjects(p []Project) { s.projects = p }

// RememberProject puts p at the front, replacing the same host and path.
func (s *Service) RememberProject(p Project) {
	out := []Project{p}
	for _, old := range s.projects {
		if old.Host != p.Host || old.Path != p.Path {
			out = append(out, old)
		}
	}
	s.projects = out
}

// RemoveProjects forgets every project on host.
func (s *Service) RemoveProjects(host string) {
	kept := s.projects[:0]
	for _, p := range s.projects {
		if p.Host != host {
			kept = append(kept, p)
		}
	}
	s.projects = kept
}

// ForgetProject removes one remembered project from the list.
func (s *Service) ForgetProject(host, path string) {
	kept := s.projects[:0]
	for _, p := range s.projects {
		if p.Host != host || p.Path != path {
			kept = append(kept, p)
		}
	}
	s.projects = kept
}

// Conn returns the live connection for host, if any.
func (s *Service) Conn(host string) *remote.Conn { return s.conns[host] }

// SetConn stores c for its host and reports whether it replaced the same
// connection.
func (s *Service) SetConn(c *remote.Conn) (same bool) {
	if c == nil {
		return false
	}
	if s.conns == nil {
		s.conns = map[string]*remote.Conn{}
	}
	same = s.conns[c.Host] == c
	s.conns[c.Host] = c
	return same
}

// DeleteConn drops and returns the connection for host, if any.
func (s *Service) DeleteConn(host string) *remote.Conn {
	c := s.conns[host]
	delete(s.conns, host)
	return c
}

// CloseAll closes every pooled connection.
func (s *Service) CloseAll() {
	for host, c := range s.conns {
		remote.Close(c)
		delete(s.conns, host)
	}
}

// Conns is the live connection map; the returned map is the service's own.
func (s *Service) Conns() map[string]*remote.Conn {
	if s.conns == nil {
		s.conns = map[string]*remote.Conn{}
	}
	return s.conns
}
