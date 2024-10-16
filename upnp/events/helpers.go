package events

import (
	"bytes"
	"crypto/md5"
	"encoding/xml"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	callbackRegexp          = regexp.MustCompile(`<([^<>]+)>`)
	DefaultSubscribeTimeout = 5 * time.Minute
	DefaultNotifyTimeout    = 5 * time.Second
)

// ParseTimeoutHeader skips all errors,
// returns
// - parsed value in format Second-[INT],
// - default value in all other cases (even if header is "Second-infinite")
func ParseTimeoutHeader(timeout string) time.Time {
	prefix := "Second-"
	if !strings.HasPrefix(timeout, prefix) {
		return time.Now().Add(DefaultSubscribeTimeout)
	}
	timeout = timeout[len(prefix):]
	if val, err := strconv.Atoi(timeout); err == nil && val > 0 {
		return time.Now().Add(time.Duration(val) * time.Second)
	}
	return time.Now().Add(DefaultSubscribeTimeout)
}

func ParseCallbackHeader(callback string) (ret []url.URL, err error) {
	if callback == "" {
		err = fmt.Errorf("empty callback")
		return
	}
	list := callbackRegexp.FindAllStringSubmatch(callback, -1)
	for _, m := range list {
		var u *url.URL
		if u, err = url.Parse(m[1]); err != nil {
			return
		}
		ret = append(ret, *u)
	}
	return
}

func NewSID(unique string) string {
	hash := md5.Sum([]byte(time.Now().String() + unique))
	return fmt.Sprintf("uuid:%x-%x-%x-%x-%x", hash[:4], hash[4:6], hash[6:8], hash[8:10], hash[10:16])
}

func BuildNotificationBody(stateVariables map[string]string) (body []byte, err error) {
	if len(stateVariables) == 0 {
		err = fmt.Errorf("empty state variables")
		return
	}
	buf := new(bytes.Buffer)
	buf.Write([]byte(xml.Header))

	enc := xml.NewEncoder(buf)
	enc.Indent("", "  ")

	start := xml.StartElement{
		Name: xml.Name{Local: "e:propertyset"},
		Attr: []xml.Attr{{Name: xml.Name{Local: "xmlns:e"}, Value: "urn:schemas-upnp-org:event-1-0"}},
	}
	if err = enc.EncodeToken(start); err != nil {
		return
	}
	propEl := xml.StartElement{Name: xml.Name{Local: "e:property"}}

	for k, v := range stateVariables {
		if err = enc.EncodeToken(propEl); err != nil {
			return
		}
		if err = enc.EncodeElement(v, xml.StartElement{Name: xml.Name{Local: k}}); err != nil {
			return
		}
		if err = enc.EncodeToken(xml.EndElement{Name: propEl.Name}); err != nil {
			return
		}
	}

	if err = enc.EncodeToken(xml.EndElement{Name: start.Name}); err != nil {
		return
	}
	if err = enc.Close(); err != nil {
		return
	}

	body = buf.Bytes()
	return
}

func SendNotification(sid string, seq uint32, u url.URL, body []byte) error {

	slog.Debug("notify",
		slog.String("sid", sid),
		slog.Uint64("seq", uint64(seq)),
		slog.String("url", u.String()),
	)

	req, err := http.NewRequest("NOTIFY", u.String(), bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("could not create a request")
	}

	req.Header["CONTENT-TYPE"] = []string{`text/xml; charset="utf-8"`}
	req.Header["NT"] = []string{"upnp:event"}
	req.Header["NTS"] = []string{"upnp:propchange"}
	req.Header["SID"] = []string{sid}
	req.Header["SEQ"] = []string{strconv.FormatUint(uint64(seq), 10)}

	client := &http.Client{Timeout: DefaultNotifyTimeout}

	res, err := client.Do(req)
	if err != nil {
		return err
	}
	_ = res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("invalid response status code: %d", res.StatusCode)
	}

	return nil
}
