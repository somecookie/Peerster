package gossiper

import "fmt"

type Gossiper struct {
	UIPort     string
	GossipAddr string
	Name       string
	Peers      []string
	Simple     bool
}

func (g *Gossiper) String() string {
	s := fmt.Sprintf("Name: %s\nUIPort: %s\nGossipAddr: %s\n", g.Name, g.UIPort, g.GossipAddr)

	if g.Simple {
		s += "Simple: yes\n"
	} else {
		s += "Simple: no\n"
	}

	s += "Peers:"

	if len(g.Peers) == 0 {
		s += " no peers"
	} else {
		s += "\n"
		for i, peer := range g.Peers {
			if i != len(g.Peers)-1 {
				s += "- " + peer + "\n"
			} else {
				s += "- " + peer
			}

		}
	}

	return s
}
