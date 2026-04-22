package crypto

type KDFParams struct {
	Salt    []byte `json:"salt"`
	Time    uint32 `json:"time"`
	Memory  uint32 `json:"memory"`
	Threads uint8  `json:"threads"`
	KeyLen  uint32 `json:"key_len"`
}

type EncryptedBlob struct {
	Version    int       `json:"version"`
	KDF        KDFParams `json:"kdf"`
	Nonce      []byte    `json:"nonce"`
	Ciphertext []byte    `json:"ciphertext"`
}

func DefaultKDFParams() KDFParams {
	return KDFParams{
		Time:    3,
		Memory:  64 * 1024,
		Threads: 4,
		KeyLen:  32,
	}
}

func FastKDFParams() KDFParams {
	return KDFParams{
		Time:    1,
		Memory:  1024,
		Threads: 1,
		KeyLen:  32,
	}
}
