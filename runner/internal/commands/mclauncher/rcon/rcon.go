package rcon

type Rcon struct {
	executor RconExecutorInterface
}

func NewRcon(executor RconExecutorInterface) *Rcon {
	return &Rcon{
		executor: executor,
	}
}
