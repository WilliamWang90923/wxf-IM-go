package client

import (
	"context"
	"errors"
	"fmt"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"net"
	"net/url"
	"time"
)

type handler struct {
	conn      net.Conn
	close     chan struct{}
	recv      chan []byte
	heartbeat time.Duration
}

type StartOptions struct {
	address string
	user    string
}

func (h *handler) readloop(dial net.Conn) error {
	logrus.Info("read loop started")
	//
	err := h.conn.SetReadDeadline(time.Now().Add(h.heartbeat * 3))
	if err != nil {
		return err
	}
	for {
		frame, err := ws.ReadFrame(dial)
		if err != nil {
			return err
		}
		if frame.Header.OpCode == ws.OpPong {
			h.conn.SetReadDeadline(time.Now().Add(h.heartbeat * 3))
		}
		if frame.Header.OpCode == ws.OpClose {
			return errors.New("remote side close the channel")
		}
		if frame.Header.OpCode == ws.OpText {
			h.recv <- frame.Payload
		}
	}
}

func (h *handler) sendText(s string) error {
	logrus.Info("send message: ", s)
	err := h.conn.SetWriteDeadline(time.Now().Add(time.Second * 8))
	if err != nil {
		return err.(net.Error)
	}
	return wsutil.WriteClientText(h.conn, []byte(s))
}

func (h *handler) heartbeatloop() error {
	logrus.Info("heartbeat loop started.")
	tick := time.NewTicker(h.heartbeat)

	for _ = range tick.C {
		logrus.Info("ping...")
		err := wsutil.WriteClientMessage(h.conn, ws.OpPing, nil)
		if err != nil {
			return err
		}
	}
	return nil
}

func connect(address string) (*handler, error) {
	_, err := url.Parse(address)
	if err != nil {
		return nil, err
	}

	dial, _, _, err := ws.Dial(context.Background(), address)
	if err != nil {
		return nil, err
	}
	h := handler{
		conn:      dial,
		close:     make(chan struct{}, 1),
		recv:      make(chan []byte, 10),
		heartbeat: time.Second * 9,
	}

	go func() {
		err := h.readloop(dial)
		if err != nil {
			logrus.Warn(err)
		}
		h.close <- struct{}{}
	}()

	go func() {
		err := h.heartbeatloop()
		if err != nil {
			logrus.Info("heartbeat loop - ", err)
		}
	}()

	return &h, nil
}

func run(ctx context.Context, startOpts *StartOptions) error {
	url := fmt.Sprintf("%s?user=%s", startOpts.address, startOpts.user)
	h, err := connect(url)
	if err != nil {
		return err
	}
	go func() {
		for msg := range h.recv {
			logrus.Info("Receive message: ", string(msg))
		}
	}()
	tk := time.NewTicker(time.Second * 6)
	for {
		select {
		case <-tk.C:
			err := h.sendText("Durandal love me.")
			if err != nil {
				logrus.Error("sendTest -", err)
			}
		case <-h.close:
			logrus.Info("connection closed")
			return nil
		}
	}
}

func NewCmd(ctx context.Context) *cobra.Command {
	opts := &StartOptions{}
	command := &cobra.Command{
		Use:   "client",
		Short: "Start client",
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(ctx, opts)
		},
	}
	command.PersistentFlags().StringVarP(&opts.address, "address", "a", "ws://localhost:8001", "server address")
	command.PersistentFlags().StringVarP(&opts.user, "user", "u", "", "user")
	return command
}
