package service

func WithinQuotaLimit(count, quota int) bool {
	return count < quota
}

func WithinCreditLimit(taken, adding int) bool {
	return taken+adding <= 24
}

func CreditsAfterSwap(taken, leaving, joining int) int {
	return taken - leaving + joining
}
