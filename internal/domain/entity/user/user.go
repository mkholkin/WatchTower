package user

type User struct {
	Login        string `json:"login"`
	PasswordHash string `json:"password_hash"`
}
