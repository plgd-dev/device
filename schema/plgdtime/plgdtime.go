// ************************************************************************
// Copyright (C) 2023 plgd.dev, s.r.o.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// 	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// ************************************************************************

package plgdtime

import (
	"errors"
	"time"
)

const (
	ResourceType = "x.plgd.dev.time"
	ResourceURI  = "/x.plgd.dev/time"
)

type Status string

const (
	StatusSyncing           Status = "syncing"
	StatusInSync            Status = "in-sync"
	StatusInSyncFromStorage Status = "in-sync-from-storage"
)

type PlgdTime struct {
	Interfaces     []string `json:"if,omitempty"`
	ResourceTypes  []string `json:"rt,omitempty"`
	Time           string   `json:"time"`                     // time in RFC3339Nano format
	LastSyncedTime string   `json:"lastSyncedTime,omitempty"` // time in RFC3339Nano format
	Status         Status   `json:"status,omitempty"`         // status of the time synchronization
}

type PlgdTimeUpdate struct {
	Time string `json:"time"` // time in RFC3339Nano format
}

func (t PlgdTime) GetTime() (time.Time, error) {
	if t.Time == "" {
		return time.Time{}, errors.New("time is empty")
	}
	return time.Parse(time.RFC3339Nano, t.Time)
}

func (t PlgdTime) GetLastSyncedTime() (time.Time, error) {
	if t.LastSyncedTime == "" {
		// lastSyncedTime is optional it can be empty
		return time.Time{}, nil
	}
	return time.Parse(time.RFC3339Nano, t.LastSyncedTime)
}
