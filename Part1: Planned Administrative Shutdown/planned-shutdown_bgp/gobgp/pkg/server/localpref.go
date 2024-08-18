// Copyright (C) 2015-2021 Nippon Telegraph and Telephone Corporation.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package server

import (
	"time"

	"github.com/eapache/channels"
	"github.com/osrg/gobgp/v3/internal/pkg/table"
	"github.com/osrg/gobgp/v3/pkg/log"
)

type LocalPrefEvent struct {
	Time     uint64
	Neighbor string
	NLRI     *table.Path
}

type localPrefManager struct {
	eventCh     *channels.InfiniteChannel
	logger      log.Logger
	localPrefCh *channels.InfiniteChannel
}

func newLocalPrefManager(logger log.Logger) *localPrefManager {
	m := &localPrefManager{
		eventCh:     channels.NewInfiniteChannel(),
		logger:      logger,
		localPrefCh: channels.NewInfiniteChannel(),
	}
	go m.HandleLocalPrefEvent()
	return m
}

func (m *localPrefManager) HandleLocalPrefEvent() {
	for ev := range m.eventCh.Out() {
		if ev == nil {
			break
		}
		go waitTime(ev.(*LocalPrefEvent).Time, ev.(*LocalPrefEvent).Neighbor, ev.(*LocalPrefEvent).NLRI, m.localPrefCh)
	}
}

func waitTime(waiting_time uint64, neighbor string, NLRI *table.Path, localPrefCh *channels.InfiniteChannel) {
	time.Sleep(time.Millisecond * time.Duration(waiting_time))
	localPrefCh.In() <- &localPrefNotify{
		Neighbor: neighbor,
		NLRI: NLRI,
	}
}
