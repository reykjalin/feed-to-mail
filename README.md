# feed-to-mail

```
usage: feed-to-inbox [<flags>] <feed> <to> <smtp> <smtp_port> <smtp_user> [<smtp_pw>]

Flags:
  --help  Show context-sensitive help (also try --help-long and --help-man).
  
Args:
  <feed>       Feed URL to use for search
  <to>         Email to deliver feed to
  <smtp>       SMTP server to send email
  <smtp_port>  Port to use for SMTP server
  <smtp_user>  Username to use for SMTP server
  [<smtp_pw>]  Password to use for SMTP server
```

Running this script will parse a provided RSS or Atom feed and send you a list of new posts along with links and publish dates.

To prevent sending posts more than once, the date of the most recent post is stored for each feed, and all posts published before that date are ignored.

The email sent will look something like this:

```
Example post title (https://example.com/post/example-post-title) - Published 2019-12-23 at 10:02

--

Example post (https://example.com/post/example-post) - Published 2019-11-02 at 14:55

--

Third example post (https://example.com/post/third-example-post) - Published 2018-05-14 at 22:34

--
```
