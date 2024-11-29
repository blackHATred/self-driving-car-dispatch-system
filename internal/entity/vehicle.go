package entity

type Vehicle struct {
	ID           int
	PasswordHash string
}

type GetVehicleResponse struct {
	ID int `json:"id"`
}

type AddVehicleRequest struct {
	Password string `json:"password"    binding:"required"`
}
