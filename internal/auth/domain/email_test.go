package domain_test

import (
	"testing"

	"github.com/checkship-deploy/checkship/internal/auth/domain"
	"github.com/checkship-deploy/checkship/pkg/assert"
)

func Test_Email_ValidatesAnEmail(t *testing.T) {
	r, err := domain.EmailFrom("")

	assert.Equal(t, "", r)
	assert.ErrorIs(t, domain.ErrInvalidEmail, err)

	r, err = domain.EmailFrom("agood@email.com")

	assert.Equal(t, "agood@email.com", r)
	assert.Nil(t, err)
}
