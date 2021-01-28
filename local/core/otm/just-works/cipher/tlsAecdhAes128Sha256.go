package cipher

import (
	"crypto/sha256"
	"fmt"
	"hash"
	"sync/atomic"

	"github.com/pion/dtls/v2"
	"github.com/pion/dtls/v2/pkg/crypto/ciphersuite"
	"github.com/pion/dtls/v2/pkg/crypto/clientcertificate"
	"github.com/pion/dtls/v2/pkg/crypto/prf"
	"github.com/pion/dtls/v2/pkg/protocol/recordlayer"
)

// TLSAecdhAes128Sha256 implements the TLS_ADH_AES128_SHA256 CipherSuite
type TLSAecdhAes128Sha256 struct {
	cbc atomic.Value // *cryptoCBC
	id  dtls.CipherSuiteID
}

// NewTLSAecdhAes128Sha256 create cipher. By RFC id must be `0xC018` but just works uses 0xff00
func NewTLSAecdhAes128Sha256(id dtls.CipherSuiteID) *TLSAecdhAes128Sha256 {
	return &TLSAecdhAes128Sha256{
		id: id,
	}
}

// CertificateType returns what type of certificate this CipherSuite exchanges
func (c *TLSAecdhAes128Sha256) CertificateType() clientcertificate.Type {
	return clientcertificate.Type(0)
}

// ID returns the ID of the CipherSuite
func (c *TLSAecdhAes128Sha256) ID() dtls.CipherSuiteID {
	return c.id
}

func (c *TLSAecdhAes128Sha256) String() string {
	return "AECDH-AES128-SHA256"
}

// HashFunc returns the hashing func for this CipherSuite
func (c *TLSAecdhAes128Sha256) HashFunc() func() hash.Hash {
	return sha256.New
}

// AuthenticationType controls what authentication method is using during the handshake
func (c *TLSAecdhAes128Sha256) AuthenticationType() dtls.CipherSuiteAuthenticationType {
	return dtls.CipherSuiteAuthenticationTypeAnonymous
}

// IsInitialized returns if the CipherSuite has keying material and can
// encrypt/decrypt packets
func (c *TLSAecdhAes128Sha256) IsInitialized() bool {
	return c.cbc.Load() != nil
}

// Init initializes the internal Cipher with keying material
func (c *TLSAecdhAes128Sha256) Init(masterSecret, clientRandom, serverRandom []byte, isClient bool) error {
	const (
		prfMacLen = 32
		prfKeyLen = 16
		prfIvLen  = 16
	)

	keys, err := prf.GenerateEncryptionKeys(masterSecret, clientRandom, serverRandom, prfMacLen, prfKeyLen, prfIvLen, c.HashFunc())
	if err != nil {
		return err
	}

	var cbc *ciphersuite.CBC
	if isClient {
		cbc, err = ciphersuite.NewCBC(
			keys.ClientWriteKey, keys.ClientWriteIV, keys.ClientMACKey,
			keys.ServerWriteKey, keys.ServerWriteIV, keys.ServerMACKey,
			sha256.New,
		)
	} else {
		cbc, err = ciphersuite.NewCBC(
			keys.ServerWriteKey, keys.ServerWriteIV, keys.ServerMACKey,
			keys.ClientWriteKey, keys.ClientWriteIV, keys.ClientMACKey,
			sha256.New,
		)
	}
	c.cbc.Store(cbc)

	return err
}

// Encrypt encrypts a single TLS RecordLayer
func (c *TLSAecdhAes128Sha256) Encrypt(pkt *recordlayer.RecordLayer, raw []byte) ([]byte, error) {
	cbc := c.cbc.Load()
	if cbc == nil {
		return nil, fmt.Errorf("CipherSuite is not ready, unable to encrypt")
	}

	return cbc.(*ciphersuite.CBC).Encrypt(pkt, raw)
}

// Decrypt decrypts a single TLS RecordLayer
func (c *TLSAecdhAes128Sha256) Decrypt(raw []byte) ([]byte, error) {
	cbc := c.cbc.Load()
	if cbc == nil { // !c.isInitialized()
		return nil, fmt.Errorf("CipherSuite is not ready, unable to decrypt")
	}

	return cbc.(*ciphersuite.CBC).Decrypt(raw)
}
