package serv

import (
	"errors"
	"fmt"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/sirupsen/logrus"
	"net"
	"net/http"
	"sync"
	"time"
)

type Server struct {
	once    sync.Once
	id      string
	address string
	sync.Mutex
	// session list
	users map[string]net.Conn
}

func (server *Server) readloop(user string, con net.Conn) error {
	readwait := time.Second * 30
	for {
		con.SetReadDeadline(time.Now().Add(readwait))
		frame, err := ws.ReadFrame(con)
		if err != nil {
			return err
		}
		if frame.Header.OpCode == ws.OpClose {
			return errors.New("remote side close connection")
		}
		if frame.Header.OpCode == ws.OpPing {
			err := wsutil.WriteServerMessage(con, ws.OpPong, nil)
			if err != nil {
				return err
			}
			continue
		}
		if frame.Header.Masked {
			ws.Cipher(frame.Payload, frame.Header.Mask, 0)
		}
		if frame.Header.OpCode == ws.OpText {
			go server.handle(user, string(frame.Payload))
		}
	}
}

func (server *Server) handle(user string, message string) {
	logrus.Infof("recv message %s from %s", message, user)
	server.Lock()
	defer server.Unlock()
	broadcast := fmt.Sprintf("%s -- FROM %s", message, user)
	for u, con := range server.users {
		if user == u {
			continue
		}
		logrus.Infof("send to %s : %s", u, broadcast)
		err := server.writeText(con, broadcast)
		if err != nil {
			logrus.Errorf("write to %s failed, error: %v", user, err)
		}
	}
}

func (server *Server) addUser(user string, con net.Conn) (net.Conn, bool) {
	server.Lock()
	defer server.Unlock()
	old, ok := server.users[user]
	server.users[user] = con
	return old, ok
}

func (server *Server) delUser(user string) {
	server.Lock()
	defer server.Unlock()
	delete(server.users, user)
}

func (server *Server) Shutdown() {
	server.once.Do(func() {
		server.Lock()
		defer server.Unlock()
		for _, con := range server.users {
			con.Close()
		}
	})
}

func NewServer(id, address string) *Server {
	return newServer(id, address)
}

func newServer(id, address string) *Server {
	return &Server{
		id:      id,
		address: address,
		users:   make(map[string]net.Conn, 100),
	}
}

func (server *Server) Start() error {
	mux := http.NewServeMux()
	log := logrus.WithFields(logrus.Fields{
		"module": "Server",
		"listen": server.address,
		"id":     server.id,
	})

	mux.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		con, _, _, err := ws.UpgradeHTTP(request, writer)
		if err != nil {
			con.Close()
			return
		}
		// read userId
		user := request.URL.Query().Get("user")
		if user == "" {
			con.Close()
			return
		}
		// put in session map
		old, ok := server.addUser(user, con)
		if ok {
			old.Close()
		}
		log.Infof("user %s connect", user)

		go func(user string, con net.Conn) {
			err := server.readloop(user, con)
			if err != nil {
				log.Error(err)
			}
			con.Close()
			server.delUser(user)
			log.Infof("connection of %s closed", user)
		}(user, con)
	})
	log.Infoln("started")
	return http.ListenAndServe(server.address, mux)
}

func (server *Server) writeText(con net.Conn, broadcast string) error {
	f := ws.NewTextFrame([]byte(broadcast))
	return ws.WriteFrame(con, f)
}
