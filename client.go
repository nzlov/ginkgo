package ginkgo

type Client interface {
	Start()
	Stop()
}

type ClientOption struct {
}
