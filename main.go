package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/VannRR/roboat/inputhandler"
	"github.com/VannRR/roboat/newsboatdb"
	rofiapi "github.com/VannRR/rofi-api"
)

const (
	roboatCacheEnvVar = "ROBOAT_CACHE_PATH"
	xdgDataHomeEnvVar = "XDG_DATA_HOME"
)

func main() {
	api, err := rofiapi.NewRofiApi(inputhandler.Data{})
	handleInitError(api, err)
	if api.Data.State != inputhandler.StateErrorSelect {
		defer api.Draw()
	}

	newsBoatDbPath, err := getNewsBoatDbPath()
	if err != nil {
		inputhandler.SetMessageToError(api, err)
		return
	}

	db, err := newsboatdb.NewNewsBoatDB(newsBoatDbPath)
	if err != nil {
		inputhandler.SetMessageToError(api, err)
		return
	}

	in := inputhandler.NewInputHandler(db, api)
	handleApiInput(api, in)
}

func handleInitError(api *rofiapi.RofiApi[inputhandler.Data], err error) {
	if !api.IsRanByRofi() {
		fmt.Println("this is a rofi script, for more information check the rofi manual")
	}

	if api.Data.State == inputhandler.StateErrorShow {
		api.Data.State = inputhandler.StateErrorSelect
	}

	if err != nil {
		inputhandler.SetMessageToError(api, err)
	}
}

func getNewsBoatDbPath() (string, error) {
	if path := os.Getenv(roboatCacheEnvVar); path != "" {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	if xdgDataHomeDir := os.Getenv(xdgDataHomeEnvVar); xdgDataHomeDir != "" {
		path := filepath.Join(xdgDataHomeDir, "newsboat/cache.db")
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	if homeDir, _ := os.UserHomeDir(); homeDir != "" {
		path := filepath.Join(homeDir, ".local/share/newsboat/cache.db")
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
		path = filepath.Join(homeDir, ".newsboat/cache.db")
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf(
		"could not find newsboat cache db, try setting the env variable $%s",
		roboatCacheEnvVar)
}

func handleApiInput(api *rofiapi.RofiApi[inputhandler.Data], in *inputhandler.InputHandler) {
	if _, ok := api.GetSelectedEntry(); ok {
		in.HandleInput()
	} else {
		in.HandleRSSFeedsShow()
	}
}
