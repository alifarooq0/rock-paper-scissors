package main

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
)

// InitAPI initializes the REST API.
func (p *Plugin) InitAPI() *mux.Router {
	r := mux.NewRouter()
	api := r.PathPrefix("/api").Subrouter()
	apiRouter := api.PathPrefix("/game/{id:[a-z0-9]+}").Subrouter()

	apiRouter.HandleFunc("/play", p.handleGame).Methods(http.MethodPost)
	return r
}

// ServeHTTP allows the plugin to implement the http.Handler interface. Requests destined for the
// /plugins/{id} path will be routed to the plugin.
func (p *Plugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
	p.API.LogDebug("New request:", "Host", r.Host, "RequestURI", r.RequestURI, "Method", r.Method)
	p.router.ServeHTTP(w, r)
}

func (p *Plugin) handleGame(w http.ResponseWriter, r *http.Request) {
	p.API.LogDebug("handleGame")

	vars := mux.Vars(r)
	request := model.PostActionIntegrationRequestFromJson(r.Body)
	if request == nil {
		p.API.LogWarn("failed to decode PostActionIntegrationRequest")
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	gameID := vars["id"]
	playerUserID := request.Context["userID"].(string)
	playerMove := Move(request.Context["move"].(float64))

	var game Game
	var origGame Game
	saved := false
	for i := 0; i < 3; i++ {
		if _, err := p.Helpers.KVGetJSON(gameID, &game); err != nil {
			p.API.LogWarn("failed to get game", "error", err.Error())
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		origGame = game

		if game.Player1.UserID == playerUserID {
			game.Player1.Move = &playerMove
		} else if game.Player2.UserID == playerUserID {
			game.Player2.Move = &playerMove
		}

		var err error
		saved, err = p.Helpers.KVCompareAndSetJSON(game.ID, origGame, game)
		if err != nil {
			p.API.LogWarn("failed to set json", "error", err.Error())
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}

		if saved {
			break
		}
	}
	if !saved {
		p.API.LogWarn("failed to update player move, cannot move forward")
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	// Post selected move
	now := model.GetMillis()
	post := &model.Post{
		CreateAt:  now,
		UpdateAt:  now,
		EditAt:    now,
		Id:        request.PostId,
		ChannelId: request.ChannelId,
		Message:   fmt.Sprintf("You picked %s!", ToString(playerMove)),
	}
	p.API.LogWarn("tryting 1")

	p.API.UpdateEphemeralPost(request.UserId, post)
	p.API.LogWarn("tryting 2")

	if game.Player1.Move != nil && game.Player2.Move != nil {
		// Ready to play
		result := game.Player1.Move.Play(*game.Player2.Move)

		player1, appErr := p.API.GetUser(game.Player1.UserID)
		if appErr != nil {
			p.API.LogWarn("failed to get player1", "error", appErr.Error())
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		player2, appErr := p.API.GetUser(game.Player2.UserID)
		if appErr != nil {
			p.API.LogWarn("failed to get player2", "error", appErr.Error())
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}

		postMessage := ""
		for i := 0; i < 3; i++ {
			if _, err := p.Helpers.KVGetJSON(gameID, &game); err != nil {
				p.API.LogWarn("failed to get game", "error", err.Error())
				http.Error(w, "invalid request", http.StatusBadRequest)
				return
			}
			origGame = game

			if result == Draw {
				// We don't store anything for a draw
				postMessage = fmt.Sprintf("**%s** %s draws with %s **%s**\nLet's go again!", player1.Username, ToEmoji(*game.Player1.Move), ToEmoji(*game.Player2.Move), player2.Username)
			} else if result == Win {
				game.WinnerUserID = game.Player1.UserID
				postMessage = fmt.Sprintf("**%s** %s beats %s **%s** \n%s wins! 🎉", player1.Username, ToEmoji(*game.Player1.Move), ToEmoji(*game.Player2.Move), player2.Username, player1.Username)
			} else if result == Lost {
				game.WinnerUserID = game.Player2.UserID
				postMessage = fmt.Sprintf("**%s** %s loses to %s **%s** \n%s wins! 🎉", player1.Username, ToEmoji(*game.Player1.Move), ToEmoji(*game.Player2.Move), player2.Username, player2.Username)
			}

			if result != Draw {
				// Bug where KVCompareAndSetJSON doesn't save if origGame === game

				// Save result
				var err error
				saved, err = p.Helpers.KVCompareAndSetJSON(gameID, origGame, game)
				if err != nil {
					p.API.LogWarn("failed to save game result", "error", err.Error())
					http.Error(w, "invalid request", http.StatusBadRequest)
					return
				}
			}

			if saved {
				break
			}
		}
		if !saved {
			p.API.LogWarn("failed to save the result json, cannot move forward")
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}

		// Post result
		post := &model.Post{
			UserId:    p.botUserID,
			ChannelId: request.ChannelId,
			Message:   postMessage,
		}
		if _, appErr := p.API.CreatePost(post); appErr != nil {
			p.API.LogWarn("failed to post response", "error", appErr.Error())
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}

		// Start a new game if game resulted in draw
		if result == Draw {
			game, err := p.newGame(game.ChannelID, game.Player1.UserID, game.Player2.UserID)
			if err != nil {
				p.API.LogWarn("failed to call newGame", "error", err.Error())
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}

			if err := p.Helpers.KVSetJSON(game.ID, game); err != nil {
				p.API.LogWarn("failed to save new game", "error", err.Error())
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}

			p.sendEphemeralPost(*game)
		}
	}

	resp := &model.PostActionIntegrationResponse{}
	w.WriteHeader(http.StatusOK)

	if _, err := w.Write(resp.ToJson()); err != nil {
		p.API.LogWarn("failed to write response", "error", err.Error())
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
}
