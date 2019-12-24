package main

import (
	"encoding/binary"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/araddon/dateparse"

	"github.com/mmcdole/gofeed"
	"github.com/prologic/bitcask"
	mail "github.com/xhit/go-simple-mail"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	feed = kingpin.Arg("feed", "URL for the feed you wish sent to you.").Required().URL()
	to   = kingpin.Arg("to", "Recipient email address.").Required().String()

	smtp      = kingpin.Arg("smtp", "SMTP server address used to send the email.").Required().String()
	smtp_port = kingpin.Arg("smtp_port", "The port to use for the SMTP server.").Required().Int()
	smtp_user = kingpin.Arg("smtp_user", "Username to use for the SMTP server.").Required().String()
	smtp_pw   = kingpin.Arg("smtp_pw", "Password to use for the SMTP server.").String()
)

func main() {
	db := openDatastore()
	defer db.Close()

	kingpin.Parse()
	feed := parseFeed(*feed)
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
		smtp_client := getSmtpClient()
		defer smtp_client.Close()

		email := createEmail(feed, new_posts)
		err := email.Send(smtp_client)
		if err != nil {
			panic(err)
		}
	}
}

func openDatastore() *bitcask.Bitcask {
	db, err := bitcask.Open(getConfigDir())
	if err != nil {
		panic(err)
	}

	return db
}

func getConfigDir() string {
	switch runtime.GOOS {
	case "windows":
		return filepath.Join(os.Getenv("APPDATA"), "feed-to-mail")
	case "darwin":
		return filepath.Join(os.Getenv("HOME"), "Library", "Application Support", "feed-to-mail")
	case "netbsd":
		fallthrough
	case "openbsd":
		fallthrough
	case "freebsd":
		fallthrough
	case "linux":
		return filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "feed-to-mail")
	}

	panic("OS not supported")
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
