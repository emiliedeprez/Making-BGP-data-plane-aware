package server

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/eapache/channels"
	api "github.com/osrg/gobgp/v3/api"
	"github.com/osrg/gobgp/v3/internal/pkg/table"
	"github.com/osrg/gobgp/v3/pkg/config/oc"
	"github.com/osrg/gobgp/v3/pkg/log"
)

type rqmEventType uint8

const (
	rqmConnected rqmEventType = iota
	rqmDisconnected
	rqmSubscribe
	rqmUnsubscribe
	rqmMsg
)

type rqmEvent struct {
	EventType      rqmEventType
	Src            string
	Data           map[string]interface{}
	MeasurementIPs []string
	Prefix         string
	Neighbor       string
	Quality        uint64
	conn           *net.TCPConn
	Path           *table.Path
	Community      uint32
}

type rqmManager struct {
	eventCh   *channels.InfiniteChannel
	lock      sync.RWMutex
	clientMap map[string]*rqmClient
	logger    log.Logger
	rqmCh     *channels.InfiniteChannel
}

type RQMInfo struct {
	TimeLimit        uint64
	SamplingInterval uint64
	Community        string
	Path             *table.Path
}

func newRQMManager(logger log.Logger) *rqmManager {
	m := &rqmManager{
		eventCh:   channels.NewInfiniteChannel(),
		clientMap: make(map[string]*rqmClient),
		logger:    logger,
		rqmCh:     channels.NewInfiniteChannel(),
	}
	go m.HandleRQMEvent()
	return m
}

func (m *rqmManager) HandleRQMEvent() {
	for ev := range m.eventCh.Out() {
		if ev == nil {
			break
		}
		client, y := m.clientMap[ev.(*rqmEvent).Src]
		if !y {
			if ev.(*rqmEvent).EventType == rqmConnected {
				ev.(*rqmEvent).conn.Close()
			}
			m.logger.Error("Can't find RQM server configuration",
				log.Fields{
					"Topic": "rqm",
					"Key":   ev.(*rqmEvent).Src})
			return
		}
		switch ev.(*rqmEvent).EventType {
		case rqmDisconnected:
			m.logger.Info("RQM server is disconnected",
				log.Fields{
					"Topic": "rqm",
					"Key":   ev.(*rqmEvent).Src})
			client.conn = nil
			client.state.Downtime = time.Now().Unix()
			go client.tryConnect()
		case rqmConnected:
			m.logger.Info("RQM server is connected",
				log.Fields{
					"Topic": "rqm",
					"Key":   ev.(*rqmEvent).Src})
			client.conn = ev.(*rqmEvent).conn
			client.state.Uptime = time.Now().Unix()
			go client.established()
		case rqmSubscribe:
			m.handleRQMSubscribe(client, ev.(*rqmEvent).MeasurementIPs, ev.(*rqmEvent).Neighbor, ev.(*rqmEvent).Prefix, ev.(*rqmEvent).Community, ev.(*rqmEvent).Path)
		case rqmUnsubscribe:
			m.handleRQMUnsubscribe(client, ev.(*rqmEvent).Neighbor, ev.(*rqmEvent).Prefix)
		case rqmMsg:
			m.handleRQMMsg(client, ev.(*rqmEvent).Data)
		}
	}
}

func (m *rqmManager) handleRQMSubscribe(client *rqmClient, measurementIPs []string, Neighbor string, Prefix string, Community uint32, Path *table.Path) {
	_, y := client.subscribed[Neighbor+Prefix]
	if !y {
		rule := client.maprules[Community]
		if client.conn != nil {
			resp := map[string]interface{}{
				"type":             "subscribe",
				"measurementIPs":   measurementIPs,
				"neighbor":         Neighbor,
				"prefix":           Prefix,
				"timeLimit":        rule.TimeLimit,
				"samplingInterval": rule.SamplingInterval,
			}

			encoder := json.NewEncoder(client.conn)
			encoder.Encode(&resp)

			client.state.RQMMessages.RQMSent.Request++
			client.subscribed[Neighbor+Prefix] = RQMInfo{
				TimeLimit:        rule.TimeLimit,
				SamplingInterval: rule.SamplingInterval,
				Community:        rule.Community,
				Path:             Path,
			}
		}
	}
}

func (m *rqmManager) handleRQMUnsubscribe(client *rqmClient, Neighbor string, Prefix string) {
	if _, y := client.subscribed[Neighbor+Prefix]; y {
		delete(client.subscribed, Neighbor+Prefix)
		if client.conn != nil {
			resp := map[string]interface{}{
				"type":     "unsubscribe",
				"neighbor": Neighbor,
				"prefix":   Prefix,
			}

			encoder := json.NewEncoder(client.conn)
			encoder.Encode(&resp)

			client.state.RQMMessages.RQMSent.Request++
		}
	}
}

func (m *rqmManager) handleRQMMsg(client *rqmClient, Data map[string]interface{}) {
	switch Data["type"].(string) {
	case "expired_timer":
		{
			Neighbor := Data["neighbor"].(string)
			Prefix := Data["prefix"].(string)
			if c, y := client.subscribed[Neighbor+Prefix]; y {
				m.rqmCh.In() <- &rqmEvent{
					Neighbor: Neighbor,
					Path:     c.Path,
					Quality:  0}
			}
			delete(client.subscribed, Neighbor+Prefix)
		}
	case "quality":
		{
			Prefix := Data["prefix"].(string)
			for n, q := range Data["neighbors"].(map[string]interface{}) {
				if c, y := client.subscribed[n+Prefix]; y {
					m.rqmCh.In() <- &rqmEvent{
						Neighbor: n,
						Path:     c.Path,
						Quality:  uint64(q.(float64))}
				}
			}
		}
	default:
		m.logger.Warn("Unrecognised type or nil type",
			log.Fields{
				"Topic": "rqm",
				"Key":   client.address})
	}

}

func (m *rqmManager) AddServer(address string, port uint32, rules []*api.RQMRule) error {
	host := net.JoinHostPort(address, strconv.Itoa(int(port)))
	if _, ok := m.clientMap[host]; ok {
		return fmt.Errorf("RQM server exists %s", host)
	}
	m.clientMap[host] = newRQMClient(address, port, m.eventCh, &m.lock, m.logger, rules)
	return nil
}

func (m *rqmManager) DeleteServer(host string) error {
	client, ok := m.clientMap[host]
	if !ok {
		return fmt.Errorf("RQM server doesn't exists %s", host)
	}
	client.stop()
	delete(m.clientMap, host)
	return nil
}

func (m *rqmManager) GetServers() []*oc.RQMServer {

	l := make([]*oc.RQMServer, 0, len(m.clientMap))
	for _, client := range m.clientMap {
		state := &client.state

		if client.conn == nil {
			state.Up = false
		} else {
			state.Up = true
		}

		state.Request += client.state.Request
		state.Response += client.state.Response

		addr, port, _ := net.SplitHostPort(client.host)
		l = append(l, &oc.RQMServer{
			Config: oc.RQMConfig{
				Address: addr,
				Port:    func() uint32 { p, _ := strconv.ParseUint(port, 10, 16); return uint32(p) }(),
			},
			State: client.state,
		})
	}
	return l
}

func (m *rqmManager) GetRules() ([]*api.Rqm, error) {
	l := make([]*api.Rqm, 0)

	for _, client := range m.clientMap {
		l = append(l, &api.Rqm{
			Conf: &api.RQMConf{
				Address: client.address,
				Port:    client.port,
			},
			Rule: client.rules,
		})
	}
	return l, nil
}

func (m *rqmManager) GetClientMap() (map[uint32]*api.RQMRule){
	for _, client := range m.clientMap {
		return client.maprules
	}
	return make(map[uint32]*api.RQMRule, 0)
}

type rqmClient struct {
	host       string
	conn       *net.TCPConn
	eventCh    *channels.InfiniteChannel
	lock       *sync.RWMutex
	cancelfnc  context.CancelFunc
	ctx        context.Context
	state      oc.RQMState
	requests   map[uint16]net.IP
	logger     log.Logger
	rules      []*api.RQMRule
	address    string
	port       uint32
	maprules   map[uint32]*api.RQMRule
	subscribed map[string]RQMInfo
}

func newRQMClient(address string, port uint32, ch *channels.InfiniteChannel, lock *sync.RWMutex, logger log.Logger, rules []*api.RQMRule) *rqmClient {
	maprules := make(map[uint32]*api.RQMRule, 0)
	for _, r := range rules {
		c, _ := table.ParseCommunity(r.Community)
		maprules[c] = r
	}
	ctx, cancel := context.WithCancel(context.Background())
	c := &rqmClient{
		host:       net.JoinHostPort(address, fmt.Sprint(port)),
		address:    address,
		port:       port,
		eventCh:    ch,
		lock:       lock,
		ctx:        ctx,
		cancelfnc:  cancel,
		requests:   make(map[uint16]net.IP),
		logger:     logger,
		rules:      rules,
		maprules:   maprules,
		subscribed: make(map[string]RQMInfo),
	}
	go c.tryConnect()
	return c
}

func (c *rqmClient) reset() {
	if c.conn != nil {
		c.conn.Close()
	}
}

func (c *rqmClient) stop() {
	c.cancelfnc()
	c.reset()
}

func (c *rqmClient) tryConnect() {
	for {
		select {
		case <-c.ctx.Done():
			return
		default:
		}
		if conn, err := net.Dial("tcp", c.host); err != nil {
			time.Sleep(connectRetryInterval * time.Second)
		} else {
			c.lock.RLock()
			c.eventCh.In() <- &rqmEvent{
				EventType: rqmConnected,
				Src:       c.host,
				conn:      conn.(*net.TCPConn),
			}
			c.lock.RUnlock()
			return
		}
	}
}

func (c *rqmClient) established() (err error) {
	defer func() {
		c.conn.Close()
		c.lock.RLock()
		c.eventCh.In() <- &rqmEvent{
			EventType: rqmDisconnected,
			Src:       c.host,
		}
		c.lock.RUnlock()
	}()

	reader := bufio.NewReader(c.conn)
	decoder := json.NewDecoder(reader)

	for {

		var req map[string]interface{}
		if err := decoder.Decode(&req); err != nil {
			return err
		}

		c.eventCh.In() <- &rqmEvent{
			EventType: rqmMsg,
			Src:       c.host,
			Data:      req,
		}

	}
}
