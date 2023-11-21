package ws

import (
	"fmt"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/s1lur/distorage/daemon/internal/usecase"
	"log"
	"net/http"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Routes struct {
	clients   map[*websocket.Conn]bool
	cryptoUC  usecase.Crypto
	storageUC usecase.Storage
}

type sessionInfo struct {
	requestMessage []byte
	sharedKey      []byte
	remotePubKey   []byte
}

func RegisterRoutes(c usecase.Crypto, s usecase.Storage) *mux.Router {
	routes := Routes{
		clients:   make(map[*websocket.Conn]bool),
		cryptoUC:  c,
		storageUC: s,
	}
	r := mux.NewRouter()
	r.HandleFunc("/store/{fileId}", routes.Store).Methods("GET", "POST")
	r.HandleFunc("/get/{fileId}", routes.Get).Methods("GET", "POST")
	r.HandleFunc("/delete/{fileId}", routes.Delete).Methods("GET", "POST")
	return r
}

func (routes *Routes) executePreamble(connection *websocket.Conn) (*sessionInfo, error) {
	privKey, err := routes.cryptoUC.GenerateECDHKey()
	if err != nil {
		return nil, err
	}
	pubKey := privKey.PublicKey()

	err = connection.WriteMessage(websocket.BinaryMessage, pubKey.Bytes())
	if err != nil {
		return nil, err
	}
	mt, message, err := connection.ReadMessage()
	if mt != websocket.BinaryMessage {
		return nil, fmt.Errorf("wrong message type received: %d", mt)
	}
	if err != nil {
		return nil, err
	}
	remotePubKey := message

	mt, message, err = connection.ReadMessage()
	if mt != websocket.BinaryMessage {
		return nil, fmt.Errorf("wrong message type received: %d", mt)
	}
	if err != nil {
		return nil, err
	}
	sharedKey, err := routes.cryptoUC.ExecuteECDH(privKey, message)
	if err != nil {
		return nil, err
	}
	mt, message, err = connection.ReadMessage()
	if err != nil {
		return nil, err
	}
	return &sessionInfo{
		sharedKey:      sharedKey,
		remotePubKey:   remotePubKey,
		requestMessage: message,
	}, nil

}

func (routes *Routes) Store(w http.ResponseWriter, r *http.Request) {
	connection, _ := upgrader.Upgrade(w, r, nil)
	defer connection.Close()

	routes.clients[connection] = true
	defer delete(routes.clients, connection)

	vars := mux.Vars(r)
	session, err := routes.executePreamble(connection)
	nonce := session.requestMessage[:32]
	sig := session.requestMessage[32:104]
	body := session.requestMessage[104:]
	err = routes.cryptoUC.VerifyAddress(
		session.sharedKey,
		nonce,
		sig,
		session.remotePubKey,
	)
	if err != nil {
		log.Fatalf("ws - store - %w", err)
		return
	}
	fileId := vars["fileId"]
	remoteAddr := routes.cryptoUC.GetAddress(session.remotePubKey)
	err = routes.storageUC.StoreFile(
		fileId,
		remoteAddr,
		body,
	)
	if err != nil {
		log.Fatalf("ws - store - %w", err)
		return
	}
	err = connection.WriteMessage(
		websocket.BinaryMessage,
		[]byte{0xc8})
	if err != nil {
		log.Fatalf("ws - store - %w", err)
		return
	}
}

func (routes *Routes) Get(w http.ResponseWriter, r *http.Request) {
	connection, _ := upgrader.Upgrade(w, r, nil)
	defer connection.Close()

	routes.clients[connection] = true
	defer delete(routes.clients, connection)

	vars := mux.Vars(r)
	session, err := routes.executePreamble(connection)
	if err != nil {
		log.Fatalf("ws - get - %w", err)
		return
	}
	nonce := session.requestMessage[:32]
	sig := session.requestMessage[32:104]
	err = routes.cryptoUC.VerifyAddress(
		session.sharedKey,
		nonce,
		sig,
		session.remotePubKey,
	)
	if err != nil {
		log.Fatalf("ws - get - %w", err)
		return
	}
	fileId := vars["fileId"]
	remoteAddr := routes.cryptoUC.GetAddress(session.remotePubKey)
	contents, err := routes.storageUC.ReadFile(fileId, remoteAddr)
	if err != nil {
		log.Fatalf("ws - get - %w", err)
		return
	}
	err = connection.WriteMessage(
		websocket.BinaryMessage,
		routes.storageUC.GetFileContents(contents))
	if err != nil {
		log.Fatalf("ws - get - %w", err)
		return
	}
}

func (routes *Routes) Delete(w http.ResponseWriter, r *http.Request) {
	connection, _ := upgrader.Upgrade(w, r, nil)
	defer connection.Close()

	routes.clients[connection] = true
	defer delete(routes.clients, connection)

	vars := mux.Vars(r)
	session, err := routes.executePreamble(connection)
	if err != nil {
		log.Fatalf("ws - delete - %w", err)
		return
	}
	nonce := session.requestMessage[:32]
	sig := session.requestMessage[32:104]
	err = routes.cryptoUC.VerifyAddress(
		session.sharedKey,
		nonce,
		sig,
		session.remotePubKey,
	)
	if err != nil {
		log.Fatalf("ws - delete - %w", err)
		return
	}
	fileId := vars["fileId"]
	remoteAddr := routes.cryptoUC.GetAddress(session.remotePubKey)
	err = routes.storageUC.DeleteFile(fileId, remoteAddr)
	if err != nil {
		log.Fatalf("ws - delete - %w", err)
		return
	}
	err = connection.WriteMessage(
		websocket.BinaryMessage,
		[]byte{0xcc},
	)
	if err != nil {
		log.Fatalf("ws - delete - %w", err)
		return
	}
}
