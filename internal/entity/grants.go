package entity

type GrantsType string

const (
	// ListGrants это тип прав для диспетчера, который позволяет работать с ТС из определенного списка
	ListGrants = GrantsType("list")
	// AllGrants это тип прав для диспетчера, который позволяет работать со всеми ТС
	AllGrants = GrantsType("all")
)
