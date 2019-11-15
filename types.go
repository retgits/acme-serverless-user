package user

import "encoding/json"

func UnmarshalUser(data string) (User, error) {
	var r User
	err := json.Unmarshal([]byte(data), &r)
	return r, err
}

func (r *User) Marshal() (string, error) {
	s, err := json.Marshal(r)
	if err != nil {
		return "", err
	}
	return string(s), nil
}

type User struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	Firstname string `json:"firstname"`
	Lastname  string `json:"lastname"`
	Email     string `json:"email"`
}

func UnmarshalRegisterResponse(data string) (RegisterResponse, error) {
	var r RegisterResponse
	err := json.Unmarshal([]byte(data), &r)
	return r, err
}

func (r *RegisterResponse) Marshal() (string, error) {
	s, err := json.Marshal(r)
	if err != nil {
		return "", err
	}
	return string(s), nil
}

type RegisterResponse struct {
	Message    string `json:"message"`
	ResourceID string `json:"resourceId"`
	Status     int64  `json:"status"`
}
