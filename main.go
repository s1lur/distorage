import (
	"crypto/aes"
	"crypto/rand"
	"crypto/ecdsa"
	"crypto/ecdh"
	"fmt"
	"github.com/wealdtech/go-merkletree/keccak256"
	"crypto/curve"
	"crypto/x509"
)

func main() {
	// устанавливаем соединение

	// ownPrivKey - собственный статичный приватный ключ
	// отправляем собственный публичный ключ (для проверки принадлежности адреса)
	marshalledOwnPubKey, err := x509.MarshallPKIXPublicKey(ownPrivKey.PublicKey)
	if err != nil {
		panic(err)
	}
	connection.WriteMessage(websocket.BinaryMessage, marshalledOwnPubKey)

	// отправляем только что сгенерированный публичный ключ для диффи-хеллмана
	curve := ecdh.X25519()
	priv, err := curve.GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}
	pubKey := privKey.PublicKey()
	marshalledPubKey, err := x509.MarshallPKIXPublicKey(pubKey)
	if err != nil {
		panic(err)
	}
	err = connection.WriteMessage(websocket.BinaryMessage, marshalledPubKey)

	// получаем публичный ключ для диффи-хеллмана
	mt, message, err := connection.ReadMessage()
	if err != nil {
		panic
	}
	if mt != websocket.BinaryMessage {
		panic(fmt.Errorf("wrong message type received: %d", mt))
	}
	remotePubKey, err := x509.ParsePKIXPublicKey(message)
	if err != nil {
		panic(err)
	}
	var sharedSectet []byte
	switch v := remotePubKey.(type) {
	case *ecdh.PublicKey:
		sharedSectet = priv.ECDH(v)
	default:
		panic(fmt.Errorf("recived wrong key type: %T", v))
	}
	
	// getEncryptedFile - получение зашифрованного файла
	// отправляем смысловое сообщение
	body := getEncryptedFile()
	nonce := make([]byte, 32)
	_, err = rand.Read(nonce)
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return err
	}
	encNonce := make([]byte, aes.BlockSize)
	block.Encrypt(nonce, encNonce)
	keccak := keccak256.New()
	sig := ecdsa.SignASN1(rand.Reader, ownPrivKey, keccak.Hash(encNonce))
	messageBody := make([]byte, 32+72+len(body))
	copy(messageBody[:32], nonce)
	copy(messageBody[32:104], sig)
	copy(messageBody[104:], body)
	err = connection.WriteMessage(websocket.BinaryMessage, messageBody)
	
	// загрузка файла завершена
}