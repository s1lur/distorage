package ws

import (
	"crypto/aes"
	"crypto/x509"
	"encoding/hex"
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

// Routes хранит в себе текущий список подключений и необходимые юзкейсы
type Routes struct {
	clients   map[*websocket.Conn]bool
	cryptoUC  usecase.Crypto
	storageUC usecase.Storage
}

// sessionInfo - служебная структура, используемая как возвращаемое знаение функции преамбулы
type sessionInfo struct {
	requestMessage []byte
	sharedKey      []byte
	remotePubKey   []byte
}

// RegisterRoutes инициализирует все ручки апи демона
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

// executePreamble осуществляет обмен ключами диффи-хеллмана с подключившимся клиентом
// и возвращает необходимую информацию о нем
func (routes *Routes) executePreamble(connection *websocket.Conn) (*sessionInfo, error) {
	// генерация приватного ключа для диффи-хеллмана
	privKey, err := routes.cryptoUC.GenerateECDHKey()
	if err != nil {
		return nil, err
	}
	// отправка публичного ключа из сгенерированного приватного
	pubKey := privKey.PublicKey()
	marshalledPubKey, err := x509.MarshalPKIXPublicKey(pubKey)
	if err != nil {
		return nil, err
	}
	err = connection.WriteMessage(websocket.BinaryMessage, marshalledPubKey)
	if err != nil {
		return nil, err
	}

	// получение публичного ключа электронной подписи клиента и ключа для обмена диффи-хеллмана
	mt, message, err := connection.ReadMessage()
	if mt != websocket.BinaryMessage {
		return nil, fmt.Errorf("wrong message type received: %d", mt)
	}
	if err != nil {
		return nil, err
	}
	pubKeyEndPos := len(message) - len(marshalledPubKey)
	remotePubKey := message[:pubKeyEndPos]

	// вычисление общего секрета
	sharedKey, err := routes.cryptoUC.ExecuteECDH(privKey, message[pubKeyEndPos:])
	if err != nil {
		return nil, err
	}

	// получение сообщения со смысловой нагрузкой (конец преамбулы)
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

// Store ручка, сохраняющая файл
func (routes *Routes) Store(w http.ResponseWriter, r *http.Request) {
	// апгрейд соединения и сохранение информации о соединении
	connection, _ := upgrader.Upgrade(w, r, nil)
	defer connection.Close()
	routes.clients[connection] = true
	defer delete(routes.clients, connection)

	// получение названия файла и проверка длины
	vars := mux.Vars(r)
	fileId := vars["fileId"]
	if !checkFileId(fileId) {
		_ = connection.WriteMessage(websocket.BinaryMessage, []byte{0x01, 0x90})
		return
	}

	// сохранение данных, полученных из преамбулы, в соответствующие переменные
	session, err := routes.executePreamble(connection)
	sigSize := session.requestMessage[0]
	nonce := session.requestMessage[1 : 1+aes.BlockSize]
	sig := session.requestMessage[1+aes.BlockSize : 1+aes.BlockSize+sigSize]
	body := session.requestMessage[1+aes.BlockSize+sigSize:]

	// проверка адреса (см. VerifyAddress)
	err = routes.cryptoUC.VerifyAddress(
		session.sharedKey,
		nonce,
		sig,
		session.remotePubKey,
	)
	if err != nil {
		log.Printf("ws - store - %w\n", err)
		return
	}

	// получаем адрес из публичного ключа ЭП
	remoteAddr := routes.cryptoUC.GetAddress(session.remotePubKey)

	// проверка на то, что файл может быть сохранён
	if !routes.storageUC.CanBeStored(fileId, remoteAddr) {
		_ = connection.WriteMessage(websocket.BinaryMessage, []byte{0x01, 0x93})
		return
	}

	// сохранение файла
	err = routes.storageUC.StoreFile(
		fileId,
		remoteAddr,
		body,
	)
	if err != nil {
		log.Printf("ws - store - %w\n", err)
		return
	}

	// отправка сообщения об успехе
	err = connection.WriteMessage(
		websocket.BinaryMessage,
		[]byte{0xc8})
	if err != nil {
		log.Printf("ws - store - %w\n", err)
		return
	}
}

// Get возвращает файл по указанному имени
func (routes *Routes) Get(w http.ResponseWriter, r *http.Request) {
	// апгрейд соединения и сохранение информации о соединении
	connection, _ := upgrader.Upgrade(w, r, nil)
	defer connection.Close()
	routes.clients[connection] = true
	defer delete(routes.clients, connection)

	// получение названия файла и проверка длины
	vars := mux.Vars(r)
	fileId := vars["fileId"]
	if !checkFileId(fileId) {
		_ = connection.WriteMessage(websocket.BinaryMessage, []byte{0x01, 0x90})
		return
	}

	// сохранение данных, полученных из преамбулы, в соответствующие переменные
	session, err := routes.executePreamble(connection)
	if err != nil {
		log.Printf("ws - get - %w\n", err)
		return
	}
	sigSize := session.requestMessage[0]
	nonce := session.requestMessage[1 : 1+aes.BlockSize]
	sig := session.requestMessage[1+aes.BlockSize : 1+aes.BlockSize+sigSize]

	// проверка адреса (см. VerifyAddress)
	err = routes.cryptoUC.VerifyAddress(
		session.sharedKey,
		nonce,
		sig,
		session.remotePubKey,
	)
	if err != nil {
		log.Printf("ws - get - %w\n", err)
		return
	}

	// получаем адрес из публичного ключа ЭП
	remoteAddr := routes.cryptoUC.GetAddress(session.remotePubKey)

	// проверка на то, что файл существует
	if !routes.storageUC.CheckExistence(fileId) {
		_ = connection.WriteMessage(websocket.BinaryMessage, []byte{0x01, 0x94})
		return
	}

	// чтение файл из файловой системы устройства (внутри метода идет проверка адреса)
	contents, err := routes.storageUC.ReadFile(fileId, remoteAddr)
	if err != nil {
		log.Printf("ws - get - %w\n", err)
		return
	}

	// отправляем прочитанный файл
	err = connection.WriteMessage(
		websocket.BinaryMessage,
		routes.storageUC.GetFileContents(contents))
	if err != nil {
		log.Printf("ws - get - %w\n", err)
		return
	}
}

// Delete удаляет переданный файл
func (routes *Routes) Delete(w http.ResponseWriter, r *http.Request) {
	// апгрейд соединения и сохранение информации о соединении
	connection, _ := upgrader.Upgrade(w, r, nil)
	defer connection.Close()
	routes.clients[connection] = true
	defer delete(routes.clients, connection)

	// получение названия файла и проверка длины
	vars := mux.Vars(r)
	fileId := vars["fileId"]
	if !checkFileId(fileId) {
		_ = connection.WriteMessage(websocket.BinaryMessage, []byte{0x01, 0x90})
		return
	}

	// сохранение данных, полученных из преамбулы, в соответствующие переменные
	session, err := routes.executePreamble(connection)
	if err != nil {
		log.Printf("ws - delete - %w\n", err)
		return
	}
	sigSize := session.requestMessage[0]
	nonce := session.requestMessage[1 : 1+aes.BlockSize]
	sig := session.requestMessage[1+aes.BlockSize : 1+aes.BlockSize+sigSize]

	// проверка адреса (см. VerifyAddress)
	err = routes.cryptoUC.VerifyAddress(
		session.sharedKey,
		nonce,
		sig,
		session.remotePubKey,
	)
	if err != nil {
		log.Printf("ws - delete - %w\n", err)
		return
	}

	// получаем адрес из публичного ключа ЭП
	remoteAddr := routes.cryptoUC.GetAddress(session.remotePubKey)

	// проверка на то, что файл существует
	if !routes.storageUC.CheckExistence(fileId) {
		_ = connection.WriteMessage(websocket.BinaryMessage, []byte{0x01, 0x94})
		return
	}

	// удаление файла из файловой системы устройства (внутри метода идет проверка адреса)
	err = routes.storageUC.DeleteFile(fileId, remoteAddr)
	if err != nil {
		log.Printf("ws - delete - %w\n", err)
		return
	}

	// отправка сообщения об успехе
	err = connection.WriteMessage(
		websocket.BinaryMessage,
		[]byte{0xcc},
	)
	if err != nil {
		log.Printf("ws - delete - %w\n", err)
		return
	}
}

func checkFileId(fileId string) bool {
	if len(fileId) != 64 {
		return false
	}
	dst := make([]byte, 32)

	_, err := hex.Decode(dst, []byte(fileId))
	return err == nil
}
