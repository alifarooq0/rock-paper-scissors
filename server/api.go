package main

// Move a player makes.
type Move int

// Result of a game.
type Result int

const (
	Rock Move = iota + 1
	Paper
	Scissor
)

const (
	Win Result = iota + 1
	Draw
	Lost
)

type Player struct {
	UserID string
	Move   *Move
}

type Game struct {
	ID           string
	ChannelID    string
	Player1      Player
	Player2      Player
	WinnerUserID string
}

func ToStringForButton(move Move) string {
	switch move {
	case Rock:
		return "Rock ✊"
	case Paper:
		return "Paper ✋"
	case Scissor:
		return "Scissor ✌️"
	default:
		return "Unknown 👽"
	}
}

func ToString(move Move) string {
	switch move {
	case Rock:
		return "rock"
	case Paper:
		return "paper"
	case Scissor:
		return "scissor"
	default:
		return "unknown"
	}
}

func ToEmoji(move Move) string {
	switch move {
	case Rock:
		return "✊"
	case Paper:
		return "✋"
	case Scissor:
		return "✌️"
	default:
		return "👽"
	}
}

func (move Move) Play(other Move) Result {
	if move == other {
		return Draw
	}

	if move == Rock && other == Paper {
		return Lost
	}

	if move == Rock && other == Scissor {
		return Win
	}

	if move == Paper && other == Rock {
		return Win
	}

	if move == Paper && other == Scissor {
		return Lost
	}

	if move == Scissor && other == Rock {
		return Lost
	}

	if move == Scissor && other == Paper {
		return Win
	}

	return Draw
}
