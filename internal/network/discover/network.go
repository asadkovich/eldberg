package discover

import (
	"errors"
	"fmt"
	"io"
	"net"
	"sync"

	"go.uber.org/zap"
)

type Network struct {
	mutex   sync.Mutex
	address *net.UDPAddr
	conn    *net.UDPConn
	pending map[nodeID]messageType

	// askCh triggered when some node requests
	// our nodes list.
	// Access to it with Ask().
	askCh chan struct{}
	// nodesCh sends out nodes gotten
	// after our ask sending to another node
	// or when some node requested us.
	// Access to it with Nodes().
	nodesCh chan []*netnode

	logger *zap.SugaredLogger
}

const (
	maxPacketSize = 1024
)

func newNetwork(ip string, port int, logger *zap.SugaredLogger) (*Network, error) {
	addr := &net.UDPAddr{
		IP:   net.ParseIP(ip),
		Port: port,
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, fmt.Errorf("listen UDP: %w", err)
	}

	n := &Network{
		address: addr,
		conn:    conn,
		pending: make(map[nodeID]messageType),
		nodesCh: make(chan []*netnode),
		logger:  logger,
	}

	return n, nil
}

func (n *Network) shutdown() error {
	if err := n.conn.Close(); err != nil {
		return fmt.Errorf("close connection: %w", err)
	}

	return nil
}

// Ask returns chan which triggered on new Ask message.
func (n *Network) Ask() chan struct{} {
	return n.askCh
}

// Nodes returns chan which sends lists of new netnodes.
func (n *Network) Nodes() chan []*netnode {
	return n.nodesCh
}

// sendAsk sends message to the peer node and starts pending response
// with list of nodes.
func (n *Network) sendAsk(node *netnode) error {
	addr := node.addr()
	conn, err := net.Dial(addr.Network(), addr.String())
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}

	msg := &message{
		ID:   node.id,
		Type: Ask,
		Data: nil,
	}

	if err := n.send(&conn, msg); err != nil {
		return fmt.Errorf("send: %w", err)
	}

	// Start pending for the response with node list
	n.addPending(node.id, Nodes)

	return nil
}

// sendPing sends message to the peer node and starts pending response
// with Pong message.
func (n *Network) sendPing(node *netnode) error {
	addr := node.addr()
	conn, err := net.Dial(addr.Network(), addr.String())
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}

	msg := &message{
		ID:   node.id,
		Type: Ping,
		Data: nil,
	}

	if err := n.send(&conn, msg); err != nil {
		return fmt.Errorf("send: %w", err)
	}

	// Start pending for the response with pong message
	n.addPending(node.id, Pong)

	return nil
}

// sendPong sends Pong message to the node which pinged us.
func (n *Network) sendPong(node *netnode) error {
	addr := node.addr()
	conn, err := net.Dial(addr.Network(), addr.String())
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}

	msg := &message{
		ID:   node.id,
		Type: Pong,
		Data: nil,
	}

	if err := n.send(&conn, msg); err != nil {
		return fmt.Errorf("send: %w", err)
	}

	return nil
}

func (n *Network) send(conn *net.Conn, msg *message) error {
	data, err := serializeMessage(msg)
	if err != nil {
		return fmt.Errorf("serialize message: %w", err)
	}

	_, err = (*conn).Write(data)
	if err != nil {
		return fmt.Errorf("write message: %w", err)
	}

	return nil
}

// read waits for incoming UDP packets.
func (n *Network) read() {
	buf := make([]byte, maxPacketSize)
	for {
		_, from, err := n.conn.ReadFromUDP(buf)
		if IsTemporaryErr(err) {
			n.logger.Debug("read from UDP temp error: %s", err)
		} else if err != nil {
			if !errors.Is(err, io.EOF) {
				n.logger.Errorf("read from UDP: %s", err)
			}

			return
		}

		if err := n.handleMessage(from, buf); err != nil {
			n.logger.Debugf("handle message failed: %s", err)
		}
	}
}

// handleMessage handles all incoming messages and decides
// which way this message should be processed.
func (n *Network) handleMessage(from *net.UDPAddr, data []byte) error {
	msg, err := deserializeMessage(data)
	if err != nil {
		return fmt.Errorf("deserialize message: %w", err)
	}

	node := &netnode{
		id:   msg.ID,
		ip:   from.IP,
		port: from.Port,
	}

	if typ, ok := n.pending[msg.ID]; ok && typ == msg.Type {
		n.handleResponse(node, msg)
	} else {
		n.handleRequest(node, msg)
	}

	return nil
}

// handleResponse process message from other node as response
// because we were pending it.
func (n *Network) handleResponse(node *netnode, msg *message) {
	switch msg.Type {
	case Pong:
	case Nodes:
		nodes, ok := msg.Data.([]*netnode)
		if !ok {
			n.logger.Debugf("received %T instead of nodes list", msg.Data)

			return
		}

		// send new nodes to DHT
		n.nodesCh <- nodes
	default:
	}

	n.removePending(node.id)
}

// handleRequest process message from other node as request
// because we were not pending it.
func (n *Network) handleRequest(node *netnode, msg *message) {
	switch msg.Type {
	case Ping:
		if err := n.sendPong(node); err != nil {
			n.logger.Debugf("send pong message: %s", err)
		}
	case Ask:
		n.askCh <- struct{}{}
	}
}

func (n *Network) addPending(id nodeID, typ messageType) {
	n.mutex.Lock()
	n.pending[id] = typ
	n.mutex.Unlock()
}

func (n *Network) removePending(id nodeID) {
	n.mutex.Lock()
	delete(n.pending, id)
	n.mutex.Unlock()
}
