package arseeding

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestEcrecoverPubkey(t *testing.T) {
	everHash := "0x2013ed3bbbbae31bc175415150c3f6af9134d8669d028506b72bef4865f4bb34"
	sig := "0xd70599fdd03d27af0aff49a63ab58539083e0cf00f9ae5c060609c715b7b58a7736b7b86c75831754fd2133bc7d2b2b7257657d6b23327a6c55a916d706a7ec61b"
	pubkey, err := ecrecoverPubkey(everHash, sig)
	assert.NoError(t, err)
	assert.Equal(t, "0481fc0c730b4ef129feb6994e79a30f604bed227bdaaeb2184a1f554c728c9b5987184f6b6ddba329b2652c3418518eb56075a3e249084d3943c7af0321967c67", pubkey)
}

func TestBase64Address(t *testing.T) {
	pub := "BNGYKP0o-HlwclXUOzKSAXuSE1RPLfhLvo3_d9JuQNGrbg4z1UA_AAD8REXBAFjjN5GJt85vk2KvYdwwER812ro"
	addr, err := Base64Address(pub)
	assert.NoError(t, err)
	assert.Equal(t, "NqG1FwGUVRcOzZeVHciV6pFLRgS1l4AkbxaYXY0kxvM", addr)
}
