package authtest

import (
	"fmt"

	"github.com/wrgl/core/pkg/auth"
)

type AuthnStore struct {
	m map[string]string
}

func NewAuthnStore() *AuthnStore {
	return &AuthnStore{
		m: map[string]string{},
	}
}

func (s *AuthnStore) SetPassword(email, password string) error {
	s.m[email] = password
	return nil
}

func (s *AuthnStore) Authenticate(email, password string) (token string, err error) {
	if v, ok := s.m[email]; ok && v == password {
		return email, nil
	}
	return "", fmt.Errorf("email/password invalid")
}

func (s *AuthnStore) CheckToken(token string) (claims *auth.Claims, err error) {
	if _, ok := s.m[token]; ok {
		return &auth.Claims{
			Email: token,
		}, nil
	}
	return nil, fmt.Errorf("invalid token")
}

func (s *AuthnStore) RemoveUser(email string) error {
	delete(s.m, email)
	return nil
}

func (s *AuthnStore) Flush() error {
	return nil
}
