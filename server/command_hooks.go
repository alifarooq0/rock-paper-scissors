package main

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
)

const (
	commandTriggerRPS = "rps"
)

func (p *Plugin) registerCommands() error {
	if err := p.API.RegisterCommand(&model.Command{

		Trigger:          commandTriggerRPS,
		AutoComplete:     true,
		AutoCompleteHint: "<@mention>",
		AutoCompleteDesc: "Start a rock paper scissor game.",
	}); err != nil {
		return errors.Wrapf(err, "failed to register %s command", commandTriggerRPS)
	}

	return nil
}

// ExecuteCommand executes a command that has been previously registered via the RegisterCommand
// API.
func (p *Plugin) ExecuteCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	commandSplit := strings.Fields(args.Command)
	if len(commandSplit) != 2 {
		return &model.CommandResponse{
			ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
			Text:         fmt.Sprintf("Unknown command: " + args.Command),
		}, nil
	}

	trigger := strings.TrimPrefix(commandSplit[0], "/")
	switch trigger {
	case commandTriggerRPS:
		return p.executeCommandRPS(args, strings.TrimPrefix(commandSplit[1], "@")), nil

	default:
		return &model.CommandResponse{
			ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
			Text:         fmt.Sprintf("Unknown command: " + args.Command),
		}, nil
	}
}

func (p *Plugin) newGame(channelID, player1UserID, player2UserID string) (*Game, error) {
	return &Game{
		ID:        model.NewId(),
		ChannelID: channelID,
		Player1:   Player{UserID: player1UserID},
		Player2:   Player{UserID: player2UserID},
	}, nil
}

func (p *Plugin) executeCommandRPS(args *model.CommandArgs, player2UserID string) *model.CommandResponse {
	player1, appErr := p.API.GetUser(args.UserId)
	if appErr != nil {
		p.API.LogWarn("failed to get player1", "error", appErr.Error())
		return &model.CommandResponse{}
	}

	player2, appErr := p.API.GetUserByUsername(player2UserID)
	if appErr != nil {
		p.API.LogWarn("failed to get player2", "error", appErr.Error())
		return &model.CommandResponse{}
	}

	game, err := p.newGame(args.ChannelId, player1.Id, player2.Id)
	if err != nil {
		p.API.LogWarn("failed to call newGame", "error", err.Error())
		return &model.CommandResponse{}
	}

	if err := p.Helpers.KVSetJSON(game.ID, game); err != nil {
		p.API.LogWarn("failed to save game", "error", err.Error())
		return &model.CommandResponse{}
	}

	post := &model.Post{
		UserId:    p.botUserID,
		ChannelId: args.ChannelId,
		RootId:    args.RootId,
		Message:   fmt.Sprintf("**%s** has challenged **%s** to a game of rock-paper-scissor", player1.Username, player2.Username),
	}
	if _, appErr := p.API.CreatePost(post); appErr != nil {
		p.API.LogWarn("failed to post poll post", "error", appErr.Error())
		return &model.CommandResponse{}
	}

	p.sendEphemeralPost(*game)

	return &model.CommandResponse{}
}

func (p *Plugin) sendEphemeralPost(game Game) {
	url := "/plugins/%s/api/game/%s/play"
	// Player1
	post := &model.Post{
		UserId:    p.botUserID,
		ChannelId: game.ChannelID,
		Message:   "Choose your move!",
		Props: model.StringInterface{
			"attachments": []*model.SlackAttachment{{
				Actions: []*model.PostAction{
					{
						Integration: &model.PostActionIntegration{
							Context: model.StringInterface{
								"userID": game.Player1.UserID,
								"move":   int(Rock),
							},
							URL: fmt.Sprintf(url, manifest.Id, game.ID),
						},
						Type: model.POST_ACTION_TYPE_BUTTON,
						Name: ToStringForButton(Rock),
					},
					{
						Integration: &model.PostActionIntegration{
							Context: model.StringInterface{
								"userID": game.Player1.UserID,
								"move":   int(Paper),
							},
							URL: fmt.Sprintf(url, manifest.Id, game.ID),
						},
						Type: model.POST_ACTION_TYPE_BUTTON,
						Name: ToStringForButton(Paper),
					},
					{
						Integration: &model.PostActionIntegration{
							Context: model.StringInterface{
								"userID": game.Player1.UserID,
								"move":   int(Scissor),
							},
							URL: fmt.Sprintf(url, manifest.Id, game.ID),
						},
						Type: model.POST_ACTION_TYPE_BUTTON,
						Name: ToStringForButton(Scissor),
					},
				},
			}},
		},
	}
	_ = p.API.SendEphemeralPost(game.Player1.UserID, post)

	// Player2
	post = &model.Post{
		UserId:    p.botUserID,
		ChannelId: game.ChannelID,
		Message:   "Choose your move!",
		Props: model.StringInterface{
			"attachments": []*model.SlackAttachment{{
				Actions: []*model.PostAction{
					{
						Integration: &model.PostActionIntegration{
							Context: model.StringInterface{
								"userID": game.Player2.UserID,
								"move":   int(Rock),
							},
							URL: fmt.Sprintf(url, manifest.Id, game.ID),
						},
						Type: model.POST_ACTION_TYPE_BUTTON,
						Name: ToStringForButton(Rock),
					},
					{
						Integration: &model.PostActionIntegration{
							Context: model.StringInterface{
								"userID": game.Player2.UserID,
								"move":   int(Paper),
							},
							URL: fmt.Sprintf(url, manifest.Id, game.ID),
						},
						Type: model.POST_ACTION_TYPE_BUTTON,
						Name: ToStringForButton(Paper),
					},
					{
						Integration: &model.PostActionIntegration{
							Context: model.StringInterface{
								"userID": game.Player2.UserID,
								"move":   int(Scissor),
							},
							URL: fmt.Sprintf(url, manifest.Id, game.ID),
						},
						Type: model.POST_ACTION_TYPE_BUTTON,
						Name: ToStringForButton(Scissor),
					},
				},
			}},
		},
	}
	_ = p.API.SendEphemeralPost(game.Player2.UserID, post)
}
