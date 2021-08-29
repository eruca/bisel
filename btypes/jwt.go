package btypes

type JwtSession interface {
	New() JwtSession
	UserID() uint
}
