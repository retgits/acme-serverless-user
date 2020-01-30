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

func (r *AllUsers) Marshal() (string, error) {
	s, err := json.Marshal(r)
	if err != nil {
		return "", err
	}
	return string(s), nil
}

type AllUsers struct {
	Data []User `json:"data"`
}

func UnmarshalLoginRequest(data string) (LoginRequest, error) {
	var r LoginRequest
	err := json.Unmarshal([]byte(data), &r)
	return r, err
}

func (r *LoginRequest) Marshal() (string, error) {
	s, err := json.Marshal(r)
	if err != nil {
		return "", err
	}
	return string(s), nil
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func UnmarshalLoginResponse(data string) (LoginResponse, error) {
	var r LoginResponse
	err := json.Unmarshal([]byte(data), &r)
	return r, err
}

func (r *LoginResponse) Marshal() (string, error) {
	s, err := json.Marshal(r)
	if err != nil {
		return "", err
	}
	return string(s), nil
}

type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	Status       int64  `json:"status"`
}

func UnmarshalVerifyTokenResponse(data string) (VerifyTokenResponse, error) {
	var r VerifyTokenResponse
	err := json.Unmarshal([]byte(data), &r)
	return r, err
}

func (r *VerifyTokenResponse) Marshal() (string, error) {
	s, err := json.Marshal(r)
	if err != nil {
		return "", err
	}
	return string(s), nil
}

type VerifyTokenResponse struct {
	Message string `json:"message"`
	Status  int    `json:"status"`
}

type UserDetailsResponse struct {
	User   User `json:"data"`
	Status int  `json:"status"`
}

func (r *UserDetailsResponse) Marshal() (string, error) {
	s, err := json.Marshal(r)
	if err != nil {
		return "", err
	}
	return string(s), nil
}
