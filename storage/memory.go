package storage

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

type memoryItem struct {
	Data   []byte
	Expire int64
}
type memoryItemMap map[string]memoryItem

type Memory struct {
	data memoryItemMap
	sync sync.Mutex
}

func (m *Memory) Post(data interface{}, expire int64) (string, error) {
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return "", DataError{err}
	}
	m.sync.Lock()
	defer m.sync.Unlock()

	for t := 0; t < 10; t++ {
		id, err := GenerateRandomId()
		if err != nil {
			return "", err
		}
		if _, present := m.data[id]; present {
			continue
		}
		m.data[id] = memoryItem{
			Data:   dataBytes,
			Expire: expire,
		}
		return id, nil
	}
	return "", IdGenerationError{fmt.Errorf("Could not find unique id")}
}

func (m *Memory) Get(id string, data interface{}) error {
	m.sync.Lock()
	defer m.sync.Unlock()
	if d, present := m.data[id]; present && d.Expire > time.Now().Unix() {
		return json.Unmarshal(d.Data, data)
	} else {
		return NotFound{fmt.Errorf("Id %s not found", id)}
	}
}

func (m *Memory) Delete(id string) error {
	m.sync.Lock()
	defer m.sync.Unlock()
	delete(m.data, id)
	return nil
}

func (m *Memory) gc() {
	saveItems := make(memoryItemMap)
	now := time.Now().Unix()
	m.sync.Lock()
	defer m.sync.Unlock()
	for key, val := range m.data {
		if val.Expire > now {
			saveItems[key] = val
		}
	}
	m.data = saveItems
}

func OpenMemoryStorage() Storage {
	mem := Memory{
		data: make(memoryItemMap),
	}
	go func() {
		for t := time.Tick(24 * time.Hour); ; <-t {
			mem.gc()
		}
	}()
	return &mem
}
