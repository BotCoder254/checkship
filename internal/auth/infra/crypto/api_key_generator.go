package crypto

import (
	"github.com/checkship-deploy/checkship/internal/auth/domain"
	"github.com/checkship-deploy/checkship/pkg/crypto"
)

const apiKeyLengthInBytes = 64

type keyGenerator struct{}

func NewKeyGenerator() domain.KeyGenerator {
	return &keyGenerator{}
}

func (*keyGenerator) Generate() (domain.APIKey, error) {
	return crypto.RandomKey[domain.APIKey](apiKeyLengthInBytes)
}
