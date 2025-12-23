package routing

const (
	ArmyMovesPrefix = "army_moves"

	WarRecognitionsPrefix = "war"

	PauseKey = "pause"

	GameLogSlug = "game_logs"
)

const (
	ExchangePerilDirect = "peril_direct"
	ExchangePerilTopic  = "peril_topic"
)

type SimpleQueueType int

const (
	Transient SimpleQueueType = iota
	Durable
)

type AckType int

const (
	Ack AckType = iota
	NackRequeue
	NackDiscard
)
