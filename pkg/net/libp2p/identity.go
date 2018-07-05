package libp2p

import (
	"crypto/rand"
	"fmt"
	"io"
	mrand "math/rand"

	"github.com/keep-network/keep-core/pkg/net/gen/pb"
	libp2pcrypto "github.com/libp2p/go-libp2p-crypto"
	peer "github.com/libp2p/go-libp2p-peer"
	pstore "github.com/libp2p/go-libp2p-peerstore"
)

// identity represents a group member's network level identity. It
// implements the net.TransportIdentifier interface. A valid group member will
// generate or provide a keypair, which will correspond to a network ID.
//
// Consumers of the net package require an ID to register with protocol level
// IDs, as well as a public key for authentication.
type identity struct {
	id      peer.ID
	pubKey  libp2pcrypto.PubKey
	privKey libp2pcrypto.PrivKey
}

type networkIdentity peer.ID

func (n networkIdentity) ProviderName() string {
	return "libp2p"
}

func (n networkIdentity) String() string {
	return peer.ID(n).Pretty()
}

func (i *identity) Marshal() ([]byte, error) {
	var (
		err error
	)

	pubKey := i.pubKey
	if pubKey == nil {
		pubKey, err = i.id.ExtractPublicKey()
		if err != nil {
			return nil, err
		}
	}
	if pubKey == nil {
		return nil, fmt.Errorf(
			"failed to generate public key with peerid %v",
			i.id.Pretty(),
		)
	}
	pubKeyBytes, err := pubKey.Bytes()
	if err != nil {
		return nil, err
	}
	return (&pb.Identity{PubKey: pubKeyBytes}).Marshal()
}

func (i *identity) Unmarshal(bytes []byte) error {
	var (
		err        error
		pid        peer.ID
		pbIdentity pb.Identity
	)

	if err = pbIdentity.Unmarshal(bytes); err != nil {
		return fmt.Errorf("unmarshalling failed with error %s", err)
	}
	i.pubKey, err = libp2pcrypto.UnmarshalPublicKey(pbIdentity.PubKey)
	if err != nil {
		return err
	}
	pid, err = peer.IDFromPublicKey(i.pubKey)
	if err != nil {
		return fmt.Errorf("Failed to generate valid libp2p identity with err: %s", err)
	}
	i.id = pid

	return nil
}

// AddIdentityToStore takes an identity and notifies the addressbook of the
// existance of a new client joining the network.
func addIdentityToStore(i *identity) (pstore.Peerstore, error) {
	// TODO: investigate a generic store interface that gives us a unified interface
	// to our address book (peerstore in libp2p) from secure storage (dht)
	peerstore := pstore.NewPeerstore()

	if err := peerstore.AddPrivKey(i.id, i.privKey); err != nil {
		return nil, fmt.Errorf("failed to add PrivateKey to store with error %s", err)
	}
	if err := peerstore.AddPubKey(i.id, i.pubKey); err != nil {
		return nil, fmt.Errorf("failed to add PubKey to store with error %s", err)
	}
	return peerstore, nil
}

// generateIdentity generates a public/private-key pair (using the libp2p/crypto
// wrapper for golang/crypto). A randseed value of 0, the default for ints in Go,
// will result in a cryptographically strong source of psuedorandomness,
// whereas providing a randseed value (anything but 0), will result in using
// the seed and math/rand, a deterministic, weak source of psuedorandomness.
func generateIdentity(randseed int) (*identity, error) {
	// FIXME: rather than a seed, read in pub/pk from config
	var r io.Reader
	if randseed != 0 {
		r = mrand.New(mrand.NewSource(int64(randseed)))
	} else {
		r = rand.Reader
	}

	privKey, pubKey, err := libp2pcrypto.GenerateKeyPairWithReader(libp2pcrypto.Ed25519, 2048, r)
	if err != nil {
		return nil, err
	}

	pid, err := peer.IDFromPublicKey(pubKey)
	if err != nil {
		return nil, err
	}

	return &identity{privKey: privKey, pubKey: pubKey, id: pid}, nil
}
