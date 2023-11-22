package app

import (
	"encoding/hex"
	"github.com/gorilla/websocket"
	"github.com/s1lur/distorage/daemon/config"
	"github.com/s1lur/distorage/daemon/internal/controller/ws"
	"github.com/s1lur/distorage/daemon/internal/usecase"
	"github.com/s1lur/distorage/daemon/pkg/wsserver"
	"github.com/sevlyar/go-daemon"
	"log"
	"net/url"
	"os"
	"os/signal"
	"path"
	"syscall"
	"time"
)

// мне в падлу здесь писать доку, можно на этот файл не обращать внимания

func Run(cfg *config.Config) {
	cntxt := &daemon.Context{
		PidFileName: "distorage_daemon.pid",
		PidFilePerm: 0644,
		LogFileName: "distorage_daemon.log",
		LogFilePerm: 0640,
		WorkDir:     "./",
		Umask:       027,
		Args:        []string{"[go-daemon app]", os.Args[1]},
	}

	d, err := cntxt.Reborn()
	if err != nil {
		log.Fatal("Unable to run: ", err)
		return
	}
	if d != nil {
		return
	}
	defer cntxt.Release()

	byteAddr, err := hex.DecodeString(cfg.Addr)
	if err != nil {
		log.Fatal("Error decoding address: ", err)
	}

	cryptoUseCase := usecase.NewCryptoUC()
	storageUseCase := usecase.NewStorageUC(path.Join(cfg.BasePath, "store"))

	router := ws.RegisterRoutes(cryptoUseCase, storageUseCase)

	wsServer := wsserver.New(router, wsserver.Port(cfg.Port))

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	go connectToServer(
		cfg.ServerIpAddr,
		byteAddr,
		interrupt)

	select {
	case s := <-interrupt:
		log.Printf("app - Run - signal: %s", s.String())
	case err = <-wsServer.Notify():
		log.Fatalf("app - Run - httpServer.Notify: %s", err)
	}

	err = wsServer.Shutdown()
	if err != nil {
		log.Fatalf("app - Run - httpServer.Shutdown: %s", err)
	}

	err = <-wsServer.Notify()
	log.Fatalf("app - run - wsServer.Notify: %s", err)
}

func connectToServer(serverIpAddr string, addr []byte, interrupt chan os.Signal) {
	u := url.URL{Scheme: "ws", Host: serverIpAddr, Path: "/"}
	log.Printf("connecting to %s", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()

	done := make(chan struct{})

	err = c.WriteMessage(websocket.TextMessage, addr)
	if err != nil {
		log.Fatal("connectToServer - error sending message: ", err)
	}

	go func() {
		defer close(done)
		for {
			mt, _, err := c.ReadMessage()
			if err != nil {
				log.Println("connectToServer - read: ", err)
				return
			}
			if mt == websocket.PingMessage {
				c.WriteMessage(websocket.PongMessage, []byte{})
				if err != nil {
					log.Println("connectToServer - write:", err)
					return
				}
			}
		}
	}()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case <-interrupt:
			log.Println("interrupt")
			err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("write close:", err)
				return
			}
			select {
			case <-done:
			case <-time.After(time.Second):
			}
			return
		}
	}
}
