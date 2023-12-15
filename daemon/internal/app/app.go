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
		cfg.ServerURL,
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

func connectToServer(serverURL string, addr []byte, interrupt chan os.Signal) {
	u := url.URL{Scheme: "ws", Host: serverURL}
	serverPath, err := url.PathUnescape(u.String())
	if err != nil {
		log.Fatalf("error parsing url: %e", err)
	}
	log.Printf("connecting to %s", serverPath)

	c, _, err := websocket.DefaultDialer.Dial(serverPath, nil)
	if err != nil {
		log.Fatal("dial:", err)
		return
	}
	c.SetPongHandler(func(string) error { log.Println("received pong from server"); return nil })
	defer c.Close()

	done := make(chan struct{})

	log.Printf("sending address")
	err = c.WriteMessage(websocket.BinaryMessage, addr)
	if err != nil {
		log.Fatal("connectToServer - error sending message: ", err)
		return
	}
	go func() {
		defer c.Close()
		defer close(done)
		go func() {
			for {
				if _, _, err := c.NextReader(); err != nil {
					break
				}
			}
		}()
		for {
			log.Println("pinging server")
			err = c.WriteMessage(websocket.PingMessage, []byte{})
			if err != nil {
				log.Println("write to server:", err)
				return
			}
			time.Sleep(20 * time.Second)
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
