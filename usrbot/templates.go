package usrbot

import (
	"bytes"
	"fmt"
	"gotgbot/model"

	"text/template"

	"github.com/gotd/contrib/storage"
	"github.com/gotd/td/telegram/query/dialogs"
)

var (
	username                = "{{$p.User.FirstName}} {{$p.User.LastName}}\nusername: @{{$p.User.Username}}"
	peername                = "{{if $p.User}}" + username + "{{else if $p.Chat}} {{$p.Chat.Title}} {{else if $p.Channel}} {{$p.Channel.Title}} {{end}}\n"
	TemplateDeletedMessages = template.Must(
		template.New("deletedMessages").
			Parse("Удалены сообщения:\n\n" +
				"{{range $i, $p := .Peers}}" +
				"======================\n" +
				"{{$i}}. " + peername + "\n" +
				"{{range $j, $m := (index $.Messages $p.Key)}} " +
				"{{$m.Date}}\n\"{{$m.Content}}\"\n\n" +
				"{{end}}" +
				"{{end}}"),
	)
)

func DeletedMessagesTemplate(msgs map[dialogs.DialogKey][]*model.Message, peers []*storage.Peer) (string, error) {
	buf := bytes.NewBuffer(nil)
	err := TemplateDeletedMessages.Execute(buf, struct {
		Messages map[dialogs.DialogKey][]*model.Message
		Peers    []*storage.Peer
	}{
		Messages: msgs,
		Peers:    peers,
	})
	if err != nil {
		return "", fmt.Errorf("failed to execute deleted messages template: %w", err)
	}

	return buf.String(), nil
}
