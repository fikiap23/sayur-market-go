package utils

import (
	"sync"

	"github.com/gorilla/websocket"
)

// Global variable
var (
	wsClients      = make(map[int]*websocket.Conn) // menyimpan koneksi websocket berdasarkan userID.
	wsClientsMutex = sync.RWMutex{}                // melindungi akses ke wsclients, agar tidak race condition saat ada banyak goroutine
)

// menambahkan koneksi websocket baru
func AddWebSocketConn(userID int, conn *websocket.Conn) {
	wsClientsMutex.Lock()
	defer wsClientsMutex.Unlock()
	wsClients[userID] = conn
}

// mengambil data websocket milik userID
func GetWebSocketConn(userID int) *websocket.Conn {
	wsClientsMutex.RLock()
	defer wsClientsMutex.RUnlock()
	return wsClients[userID]
}

// menghapus konek websocket userID
func RemoveWebSocketConn(userID int) {
	wsClientsMutex.Lock()
	defer wsClientsMutex.Unlock()
	delete(wsClients, userID)
}
