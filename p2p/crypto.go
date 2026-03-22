package p2p

import (
	"crypto/rand"
	"math/big"

	"github.com/anthdm/ggpoker/deck"
)

// RFC 3526 2048-bit MODP Group 14
const primeHex = "FFFFFFFFFFFFFFFFC90FDAA22168C234C4C6628B80DC1CD129024E088A67CC74020BBEA63B139B22514A08798E3404DDEF9519B3CD3A431B302B0A6DF25F14374FE1356D6D51C245E485B576625E7EC6F44C42E9A637ED6B0BFF5CB6F406B7EDEE386BFB5A899FA5AE9F24117C4B1FE649286651ECE45B3DC2007CB8A163BF0598DA48361C55D39A69163FA8FD24CF5F83655D23DCA3AD961C62F356208552BB9ED529077096966D670C354E4ABC9804F1746C08CA18217C32905E462E36CE3BE39E772C180E86039B2783A2EC07A28FB5C55DF06F4C52C9DE2BCBF6955817183995497CEA956AE515D2261898FA051015728E5A8AACAA68FFFFFFFFFFFFFFFF"

var (
	SharedPrime *big.Int
	totient     *big.Int
	one         = big.NewInt(1)
)

func init() {
	SharedPrime, _ = new(big.Int).SetString(primeHex, 16)
	totient = new(big.Int).Sub(SharedPrime, one)
}

type KeyPair struct {
	EncryptKey *big.Int
	DecryptKey *big.Int
}

func GenerateKeys() (*KeyPair, error) {
	for {
		e, err := rand.Int(rand.Reader, totient)
		if err != nil {
			return nil, err
		}
		if e.Cmp(one) <= 0 {
			continue
		}
		gcd := new(big.Int).GCD(nil, nil, e, totient)
		if gcd.Cmp(one) == 0 {
			d := new(big.Int).ModInverse(e, totient)
			if d != nil {
				return &KeyPair{EncryptKey: e, DecryptKey: d}, nil
			}
		}
	}
}

func Encrypt(payload []byte, key *big.Int) []byte {
	m := new(big.Int).SetBytes(payload)
	c := new(big.Int).Exp(m, key, SharedPrime)
	return c.Bytes()
}

func Decrypt(payload []byte, key *big.Int) []byte {
	c := new(big.Int).SetBytes(payload)
	m := new(big.Int).Exp(c, key, SharedPrime)
	return m.Bytes()
}

func IntToCard(val int) deck.Card {
	id := val - 2
	suit := deck.Suit(id / 13)
	value := (id % 13) + 1
	return deck.NewCard(suit, value)
}

func GeneratePlaintextDeck() [][]byte {
	out := make([][]byte, 52)
	for i := 0; i < 52; i++ {
		out[i] = []byte{byte(i + 2)}
	}
	return out
}
