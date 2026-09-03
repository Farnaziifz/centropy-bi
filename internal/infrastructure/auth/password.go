package auth

import "golang.org/x/crypto/bcrypt"

// HashPassword and VerifyPassword wrap bcrypt — the admin API's login is a
// plain email+password form (unlike AlefGym's customer-facing OTP flow),
// so this service needs its own credential hashing.
func HashPassword(plain string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func VerifyPassword(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}
