package btypes

type Loginer interface {
	Login() (account, password string)
}
