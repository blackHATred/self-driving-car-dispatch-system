package password

import (
	"encoding/base64"
	"golang.org/x/crypto/bcrypt"
)

// HashPassword хеширует пароль с использованием bcrypt.
// bcrypt автоматически включает соль в хеш, так что её не нужно передавать отдельно.
// Функция возвращает хэш, закодированный в base64.
func HashPassword(password string) (string, error) {
	// bcrypt.GenerateFromPassword использует алгоритм bcrypt для создания хеша пароля.
	// bcrypt.DefaultCost определяет сложность вычислений (по умолчанию 10).
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	// Возвращаем хеш, закодированный в base64
	return base64.StdEncoding.EncodeToString(hash), nil
}

// CheckPassword проверяет пароль на соответствие хешу с использованием bcrypt.
// Password - пароль.
// Hash - хеш пароля, полученный с помощью bcrypt, закодированный в base64.
func CheckPassword(password, hash string) bool {
	// Декодируем хеш из base64
	decodedHash, err := base64.StdEncoding.DecodeString(hash)
	if err != nil {
		return false
	}
	// Сравниваем пароль с хешем
	return bcrypt.CompareHashAndPassword(decodedHash, []byte(password)) == nil
}
