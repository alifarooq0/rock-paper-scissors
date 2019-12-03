package main

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/blang/semver"
	"github.com/mattermost/mattermost-server/model"
	"github.com/pkg/errors"
)

const minimumServerVersion = "5.12.0"

func (p *Plugin) checkServerVersion() error {
	serverVersion, err := semver.Parse(p.API.GetServerVersion())
	if err != nil {
		return errors.Wrap(err, "failed to parse server version")
	}

	r := semver.MustParseRange(">=" + minimumServerVersion)
	if !r(serverVersion) {
		return fmt.Errorf("this plugin requires Mattermost v%s or later", minimumServerVersion)
	}

	return nil
}

const (
	botUserName    = "rps"
	botDisplayName = "Rock Paper Scissor"
)

// OnActivate is invoked when the plugin is activated.
func (p *Plugin) OnActivate() error {
	p.API.LogInfo("OnActivate")

	if err := p.checkServerVersion(); err != nil {
		return err
	}

	if err := p.registerCommands(); err != nil {
		return errors.Wrap(err, "failed to register commands")
	}

	bot := &model.Bot{
		Username:    botUserName,
		DisplayName: botDisplayName,
	}
	botUserID, appErr := p.Helpers.EnsureBot(bot)
	if appErr != nil {
		return errors.Wrap(appErr, "failed to ensure bot user")
	}
	p.botUserID = botUserID

	path, err := p.API.GetBundlePath()
	if err != nil {
		return errors.Wrap(err, "failed to get bundle path")
	}

	data, err := ioutil.ReadFile(filepath.Join(path, "/assets/image.png"))
	if err != nil {
		return errors.Wrap(err, "failed to read bot image")
	}

	if appErr := p.API.SetProfileImage(p.botUserID, data); appErr != nil {
		return errors.Wrap(appErr, "failed to set bot profile image")
	}

	p.router = p.InitAPI()

	return nil
}

// OnDeactivate is invoked when the plugin is deactivated. This is the plugin's last chance to use
// the API, and the plugin will be terminated shortly after this invocation.
func (p *Plugin) OnDeactivate() error {
	p.API.LogInfo("OnDeactivate")
	return nil
}
