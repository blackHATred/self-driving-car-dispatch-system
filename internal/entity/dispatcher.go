package entity

type Dispatcher struct {
	ID           int
	GrantsType   GrantsType
	Grants       []int
	PasswordHash string
}

type GetDispatcherResponse struct {
	ID         int        `json:"id"`
	GrantsType GrantsType `json:"grants_type"`
	Grants     []int      `json:"grants"`
}

type AddDispatcherRequest struct {
	Password   string     `json:"password"    binding:"required"`
	GrantsType GrantsType `json:"grants_type" binding:"required"`
	Grants     []int      `json:"grants"      binding:"omitempty"`
}

type EditDispatcherRequest struct {
	ID         int        `json:"id"          binding:"required"`
	GrantsType GrantsType `json:"grants_type" binding:"required"`
	Grants     []int      `json:"grants"      binding:"omitempty"`
}

func IsGrantsTypeValid(grantsType GrantsType) bool {
	return grantsType == ListGrants || grantsType == AllGrants
}
