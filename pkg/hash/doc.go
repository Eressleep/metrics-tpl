// Package hash предоставляет функции для работы с HMAC-SHA256.
//
// Используется для подписи и проверки целостности метрик при передаче
// между агентом и сервером.
//
// Пример использования:
//
//	key := "secret-key"
//	data := []byte("metric data")
//
//	hash := hash.ComputeHMAC(data, key)
//	fmt.Printf("Hash: %s\n", hash)
//
//	if hash.VerifyHMAC(data, hash, key) {
//	    fmt.Println("Hash verified")
//	}
package hash
