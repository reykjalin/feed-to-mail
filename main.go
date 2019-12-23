package main

import (
	"encoding/binary"
	"net/url"
	"os"
	"time"

	"github.com/araddon/dateparse"

	"github.com/mmcdole/gofeed"
	"github.com/prologic/bitcask"
	mail "github.com/xhit/go-simple-mail"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	feed = kingpin.Arg("feed", "Feed URL to use for search").Required().URL()
	to   = kingpin.Arg("to", "Email to deliver feed to").Required().String()

	smtp      = kingpin.Arg("smtp", "SMTP server to send email").Required().String()
	smtp_port = kingpin.Arg("smtp_port", "Port to use for SMTP server").Required().Int()
	smtp_user = kingpin.Arg("smtp_user", "Username to use for SMTP server").Required().String()
	smtp_pw   = kingpin.Arg("smtp_pw", "Password to use for SMTP server").String()
)

func main() {
	kingpin.Parse()

	db := openDatastore()
	defer db.Close()

	feed := parseFeed(*feed)
	smtp_client := getSmtpClient()
	defer smtp_client.Close()

	last_updated_timestamp := getLastUpdateTime(db, feed.Link)
	last_updated := time.Unix(last_updated_timestamp, 0)

	new_posts := make([]string, 0)
	new_last_updated := last_updated
	for _, item := range feed.Items {
		item_date := getItemDate(item)

		if last_updated.Equal(item_date) || last_updated.After(item_date) {
			continue
		}
		new_posts = append(new_posts, item.Title+" ("+item.Link+")"+" - published "+item_date.Format("2006-01-02 at 15:04"))

		if item_date.After(new_last_updated) {
			new_last_updated = item_date
		}
	}
	putUpdateTime(db, feed.Link, new_last_updated)

	if len(new_posts) > 0 {
		email := createEmail(feed, new_posts)
		err := email.Send(smtp_client)
		if err != nil {
			panic(err)
		}
	}
}

func openDatastore() *bitcask.Bitcask {
	db, err := bitcask.Open(os.Getenv("HOME") + "/.config/feed-to-mail")
	if err != nil {
		panic(err)
	}

	return db
}

func parseFeed(url *url.URL) *gofeed.Feed {
	fp := gofeed.NewParser()
	feed, err := fp.ParseURL(url.String())
	if err != nil {
		panic(err)
	}
	return feed
}

func getSmtpClient() *mail.SMTPClient {
	server := mail.NewSMTPClient()
	server.Host = *smtp
	server.Port = *smtp_port
	server.Username = *smtp_user
	server.Password = *smtp_pw
	server.Encryption = mail.EncryptionSSL
	server.KeepAlive = true
	server.ConnectTimeout = 10 * time.Second
	server.SendTimeout = 10 * time.Second

	smtpClient, err := server.Connect()

	if err != nil {
		panic(err)
	}
	return smtpClient
}

func getLastUpdateTime(datastore *bitcask.Bitcask, feed_id string) int64 {
	lastUpdate, err := datastore.Get([]byte(feed_id))
	if err != nil {
		lastUpdate = make([]byte, 8)
		binary.LittleEndian.PutUint64(lastUpdate, 0)
	}
	return int64(binary.LittleEndian.Uint64(lastUpdate))
}

func getItemDate(item *gofeed.Item) time.Time {
	if len(item.Published) > 0 && len(item.Updated) > 0 {
		pub_date := parseDate(item.Published)
		upd_date := parseDate(item.Updated)

		if upd_date.After(pub_date) {
			return upd_date
		}
		return pub_date
	} else if len(item.Published) > 0 && "" == item.Updated {
		return parseDate(item.Published)
	} else if len(item.Updated) > 0 && "" == item.Published {
		return parseDate(item.Updated)
	}

	panic("no valid date")
}

func parseDate(date string) time.Time {
	parsed_date, err := dateparse.ParseStrict(date)
	if err != nil {
		panic(err)
	}

	return parsed_date
}

func putUpdateTime(datastore *bitcask.Bitcask, feed_id string, new_date time.Time) {
	timestampBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(timestampBytes, uint64(new_date.Unix()))
	datastore.Put([]byte(feed_id), timestampBytes)
}

func createEmail(feed *gofeed.Feed, new_posts []string) *mail.Email {
	email := mail.NewMSG()
	body := createEmailBody(new_posts)

	email.SetFrom("Feed to Mail <feed-to-mail@thorlaksson.com>").
		AddTo(*to).
		SetSubject(feed.Title+" - digest").
		SetBody(mail.TextPlain, body)

	return email
}

func createEmailBody(posts []string) string {
	body := ""
	for _, line := range posts {
		body = body + line + "\n\n--\n\n"
	}

	return body
}
