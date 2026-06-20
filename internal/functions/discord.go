package functions

import (
	"fmt"
	"os"
	"time"

	"github.com/bwmarrin/discordgo"
)

const (
	embedColor = 3651327
	avatarURL  = "https://cdn.thepilotclub.org/fcp/tpc%20logo.png"
	footerText = "Made by the TPC Tech Team"
	footerIcon = "https://static1.squarespace.com/static/614689d3918044012d2ac1b4/t/616ff36761fabc72642806e3/1634726781251/TPC_FullColor_TransparentBg_1280x1024_72dpi.png"
)

// discordTime renders t as a Discord <t:…:f> timestamp.
func discordTime(t time.Time) string {
	return fmt.Sprintf("<t:%d:f>", t.Unix())
}

// field is a shorthand for a name/value embed field.
func field(name, value string) *discordgo.MessageEmbedField {
	return &discordgo.MessageEmbedField{Name: name, Value: value}
}

// newEmbed builds an embed with the shared TPC colour and footer.
func newEmbed(title string, fields ...*discordgo.MessageEmbedField) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:  title,
		Fields: fields,
		Color:  embedColor,
		Footer: &discordgo.MessageEmbedFooter{Text: footerText, IconURL: footerIcon},
	}
}

// Webhook posts and edits the announcement messages. It is satisfied by
// discordWebhook in production and can be faked in tests.
type Webhook interface {
	// Send posts embed as a new message and returns its ID.
	Send(username string, embed *discordgo.MessageEmbed) (messageID string, err error)
	// Edit replaces the embed of an existing message.
	Edit(messageID string, embed *discordgo.MessageEmbed) error
}

// discordWebhook talks to a single Discord webhook configured via the
// WEBHOOK_ID and WEBHOOK_TOKEN environment variables.
type discordWebhook struct {
	client *discordgo.Session
	id     string
	token  string
}

// newDiscordWebhook creates a webhook client from the WEBHOOK_ID and
// WEBHOOK_TOKEN environment variables.
func newDiscordWebhook() (*discordWebhook, error) {
	client, err := discordgo.New("")
	if err != nil {
		return nil, fmt.Errorf("create discord client: %w", err)
	}
	return &discordWebhook{
		client: client,
		id:     os.Getenv("WEBHOOK_ID"),
		token:  os.Getenv("WEBHOOK_TOKEN"),
	}, nil
}

func (w *discordWebhook) Send(username string, embed *discordgo.MessageEmbed) (string, error) {
	m, err := w.client.WebhookExecute(w.id, w.token, true, &discordgo.WebhookParams{
		Embeds:    []*discordgo.MessageEmbed{embed},
		AvatarURL: avatarURL,
		Username:  username,
	})
	if err != nil {
		return "", err
	}
	return m.ID, nil
}

func (w *discordWebhook) Edit(messageID string, embed *discordgo.MessageEmbed) error {
	_, err := w.client.WebhookMessageEdit(w.id, w.token, messageID, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{embed},
	})
	return err
}
