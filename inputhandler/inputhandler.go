// inputhandler, handles rofi input and app state
package inputhandler

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/VannRR/roboat/newsboatdb"
	rofiapi "github.com/VannRR/rofi-api"
)

const (
	roboatBrowserEnvVar = "ROBOAT_BROWSER"
	entryMaxLen         = 100
	opExit              = "<-- Exit"
	opBack              = "<-- Back"
)

type State byte

const (
	StateNull State = iota
	StateErrorShow
	StateErrorSelect
	StateRSSFeedsShow
	StateRSSFeedsSelect
	StateRSSItemsShow
	StateRSSItemsSelect
)

type Data struct {
	State   State
	FeedURL string
	ItemURL string
	ItemID  int
}

// InputHandler manages input from rofi and application state
type InputHandler struct {
	db      newsboatdb.DBInterface
	api     *rofiapi.RofiApi[Data]
	browser string
}

// NewInputHandler returns a new InputHandler instance
func NewInputHandler(db newsboatdb.DBInterface, api *rofiapi.RofiApi[Data]) *InputHandler {
	return &InputHandler{
		db:      db,
		api:     api,
		browser: os.Getenv(roboatBrowserEnvVar),
	}
}

// HandleInput processes the selected rofi entry/input based on the app state
func (in *InputHandler) HandleInput() {
	switch in.api.Data.State {
	case StateRSSFeedsShow:
		in.HandleRSSFeedsShow()
	case StateRSSFeedsSelect:
		in.handleRSSFeedsSelect()
	case StateRSSItemsShow:
		in.handleRSSItemsShow()
	case StateRSSItemsSelect:
		in.handleRSSItemsSelect()
	default:
		log.Printf("Unhandled state: %v", in.api.Data)
	}
}

// HandleRSSFeedsShow sets rofi's state and displays all RSS feeds
func (in *InputHandler) HandleRSSFeedsShow() {
	in.api.Options[rofiapi.OptionMessage] = formatMessage("Reload all: Alt+1")
	in.api.Options[rofiapi.OptionNoCustom] = "true"
	in.api.Options[rofiapi.OptionUseHotKeys] = "true"
	in.api.Options[rofiapi.OptionKeepSelection] = "false"

	rssFeeds, err := in.db.GetFeeds()
	if err != nil {
		SetMessageToError(in.api, err)
		return
	}
	entries := make([]rofiapi.Entry, 0, len(rssFeeds))
	for _, rf := range rssFeeds {
		text := " "
		if rf.TotalItems > 0 && rf.UnreadItems > 0 {
			text = "N"
		}

		text += fmt.Sprintf("%12s ", fmt.Sprintf("(%d/%d)", rf.UnreadItems, rf.TotalItems))
		text += rf.Title

		entries = append(entries, rofiapi.Entry{
			Text: formatEntryText(text),
			Info: fmt.Sprint(rf.RssURL),
		})
	}

	in.api.Entries = entries
	in.api.Data.State = StateRSSFeedsSelect
}

func (in *InputHandler) handleRSSFeedsSelect() {
	rofiState := in.api.GetState()

	selected, _ := in.api.GetSelectedEntry()
	in.api.Data.FeedURL = selected.Info

	switch rofiState {
	case rofiapi.StateCustomKeybinding1:
		in.handleReloadAll()
	case rofiapi.StateSelected:
		in.handleRSSItemsShow()
	default:
		in.HandleRSSFeedsShow()
	}
}

// handleRSSItemsShow sets rofi's initial state and displays all RSS items
func (in *InputHandler) handleRSSItemsShow() {
	in.api.Options[rofiapi.OptionMessage] = formatMessage("Toggle read: Alt+1")
	in.api.Options[rofiapi.OptionNoCustom] = "true"
	in.api.Options[rofiapi.OptionUseHotKeys] = "true"

	rssItems, err := in.db.GetItems(in.api.Data.FeedURL)
	if err != nil {
		SetMessageToError(in.api, err)
		return
	}
	entries := make([]rofiapi.Entry, 0, len(rssItems))
	entries = append(entries, rofiapi.Entry{Text: opBack})
	for _, ri := range rssItems {
		text := ""
		if ri.Unread {
			text += "N "
		} else {
			text += "  "
		}

		text += ri.PubDate.Format("Jan 02") + " "
		text += ri.Title

		entries = append(entries, rofiapi.Entry{
			Text: formatEntryText(text),
			Info: fmt.Sprintf("%d %s", ri.ID, ri.URL),
		})
	}

	in.api.Entries = entries
	in.api.Data.State = StateRSSItemsSelect
}

func (in *InputHandler) handleRSSItemsSelect() {
	rofiState := in.api.GetState()

	selectedEntry, _ := in.api.GetSelectedEntry()

	if selectedEntry.Text == opBack {
		in.HandleRSSFeedsShow()
		return
	}

	infoSplit := strings.SplitN(selectedEntry.Info, " ", 2)
	id, _ := strconv.ParseInt(infoSplit[0], 10, 64)
	url := infoSplit[1]

	in.api.Data.ItemURL = url
	in.api.Data.ItemID = int(id)

	switch rofiState {
	case rofiapi.StateCustomKeybinding1:
		in.api.Options[rofiapi.OptionKeepSelection] = "true"
		in.db.ToggleUnread(in.api.Data.ItemID)
		in.handleRSSItemsShow()
	case rofiapi.StateSelected:
		in.handleGotoURL()
	default:
		in.handleRSSItemsShow()
	}
}

func (in *InputHandler) handleReloadAll() {
	const config = "reload-threads 100"

	tmpFile, err := os.CreateTemp(os.TempDir(), "config*.conf")
	if err != nil {
		log.Printf("Error creating temporary file: %v", err)
		return
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(config); err != nil {
		log.Printf("Error writing to temporary file: %v", err)
		return
	}

	if err := tmpFile.Close(); err != nil {
		log.Printf("Error closing temporary file: %v", err)
		return
	}

	cmd := exec.Command("newsboat", "-C", tmpFile.Name(), "-x", "reload")
	if err := cmd.Run(); err != nil {
		SetMessageToError(in.api, fmt.Errorf("error reloading newsboat feeds: %w", err))
		return
	}

	in.HandleRSSFeedsShow()
}

func (in *InputHandler) handleGotoURL() {
	b := in.browser
	if b == "" {
		b = "xdg-open"
	}
	cmd := exec.Command(b, in.api.Data.ItemURL)
	if err := cmd.Start(); err != nil {
		e := fmt.Errorf("error opening URL: %w", err)
		if b == "xdg-open" {
			e = fmt.Errorf(
				"error opening URL: xdg-utils is not installed, to use without set env variable $%s",
				roboatBrowserEnvVar)
		}
		SetMessageToError(in.api, e)
		return
	}
	in.db.SetUnread(in.api.Data.ItemID, false)
}

// setMessageToError sets rofi's message box to the text of an error
func SetMessageToError(api *rofiapi.RofiApi[Data], err error) {
	log.Println("ERROR ", err)
	api.Options[rofiapi.OptionMessage] = fmt.Sprintf(
		"<markup><span font_weight=\"bold\">error:</span><span> %s</span></markup>",
		rofiapi.EscapePangoMarkup(err.Error()))
	api.Options[rofiapi.OptionNoCustom] = "true"
	api.Entries = []rofiapi.Entry{{Text: opExit}}
	api.Data.State = StateErrorShow
}

func formatMessage(text string) string {
	return fmt.Sprintf("<markup><span font_weight=\"bold\">%s</span></markup>",
		rofiapi.EscapePangoMarkup(text))
}

func formatEntryText(e string) string {
	e = truncateEnd(e, entryMaxLen)
	return replaceNewlines(e)
}

func truncateEnd(s string, l int) string {
	if len(s) > l && l >= 0 {
		return s[:l]
	}
	return s
}

func replaceNewlines(s string) string {
	return strings.ReplaceAll(s, "\n", " ")
}
