package email

import (
	"archive/zip"
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net"
	"net/http"
	"net/mail"
	"net/smtp"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Client struct {
	config    SMTPConfig
	blacklist Blacklist
}

func NewClient(config SMTPConfig, blacklist Blacklist) *Client {
	if config.RetryInterval <= 0 {
		config.RetryInterval = 5 * time.Second
	}
	if config.MaxRetries <= 0 {
		config.MaxRetries = 1
	}
	return &Client{config: config, blacklist: blacklist}
}

func (c *Client) Send(message Message) error {
	recipients := c.eliminate(message.To)
	if len(recipients) == 0 {
		return ErrNoRecipients
	}
	var last error
	for attempt := 0; attempt < c.config.MaxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(c.config.RetryInterval)
		}
		if err := c.sendOnce(message, recipients); err != nil {
			last = err
			continue
		}
		return nil
	}
	return last
}

func (c *Client) sendOnce(message Message, recipients []string) error {
	raw, err := c.buildMIME(message, recipients)
	if err != nil {
		return err
	}
	addr := net.JoinHostPort(c.config.SMTPServer, c.config.Port)
	auth := smtp.PlainAuth("", c.config.Username, c.config.Password, c.config.SMTPServer)
	allRecipients := append(append([]string{}, recipients...), c.eliminate(message.CC)...)
	allRecipients = append(allRecipients, c.eliminate(message.BCC)...)
	if c.config.SSL {
		return c.sendTLS(addr, auth, allRecipients, raw)
	}
	return smtp.SendMail(addr, auth, c.config.FromAddress, allRecipients, raw)
}

func (c *Client) sendTLS(addr string, auth smtp.Auth, recipients []string, raw []byte) error {
	conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: c.config.SMTPServer})
	if err != nil {
		return err
	}
	defer conn.Close()
	client, err := smtp.NewClient(conn, c.config.SMTPServer)
	if err != nil {
		return err
	}
	defer client.Quit()
	if c.config.Auth {
		if err := client.Auth(auth); err != nil {
			return err
		}
	}
	if err := client.Mail(c.config.FromAddress); err != nil {
		return err
	}
	for _, recipient := range recipients {
		if err := client.Rcpt(recipient); err != nil {
			return err
		}
	}
	writer, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := writer.Write(raw); err != nil {
		return err
	}
	return writer.Close()
}

func (c *Client) buildMIME(message Message, recipients []string) ([]byte, error) {
	var buffer bytes.Buffer
	writer := multipart.NewWriter(&buffer)
	from := c.config.FromAddress
	if c.config.NickName != "" {
		from = (&mail.Address{Name: c.config.NickName, Address: c.config.FromAddress}).String()
	}
	headers := map[string]string{
		"From":         from,
		"To":           strings.Join(recipients, ","),
		"Subject":      mime.QEncoding.Encode("UTF-8", message.Title),
		"MIME-Version": "1.0",
		"Content-Type": "multipart/mixed; boundary=" + writer.Boundary(),
	}
	if len(message.CC) > 0 {
		headers["Cc"] = strings.Join(c.eliminate(message.CC), ",")
	}
	for key, value := range headers {
		buffer.WriteString(key + ": " + value + "\r\n")
	}
	buffer.WriteString("\r\n")
	if err := addBody(writer, message); err != nil {
		return nil, err
	}
	if message.ZipName != "" && len(message.Files) > 0 {
		if err := addZip(writer, message.ZipName, message.Files); err != nil {
			return nil, err
		}
	} else {
		for name, path := range message.Files {
			if err := addFile(writer, name, path); err != nil {
				return nil, err
			}
		}
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func addBody(writer *multipart.Writer, message Message) error {
	content := message.Body
	contentType := "text/plain; charset=utf-8"
	if message.HTMLContent != "" {
		content = replaceValues(message.HTMLContent, message.HTMLValues)
		contentType = "text/html; charset=utf-8"
	}
	part, err := writer.CreatePart(map[string][]string{
		"Content-Type":              {contentType},
		"Content-Transfer-Encoding": {"base64"},
	})
	if err != nil {
		return err
	}
	_, err = part.Write([]byte(base64.StdEncoding.EncodeToString([]byte(content))))
	return err
}

func addFile(writer *multipart.Writer, fileName string, path string) error {
	data, err := readFileOrURL(path)
	if err != nil {
		return err
	}
	return addAttachment(writer, fileName, data)
}

func addZip(writer *multipart.Writer, zipName string, files map[string]string) error {
	var buffer bytes.Buffer
	zipWriter := zip.NewWriter(&buffer)
	for fileName, path := range files {
		data, err := readFileOrURL(path)
		if err != nil {
			return err
		}
		fileWriter, err := zipWriter.Create(fileName)
		if err != nil {
			return err
		}
		if _, err := fileWriter.Write(data); err != nil {
			return err
		}
	}
	if err := zipWriter.Close(); err != nil {
		return err
	}
	if zipName == "" {
		zipName = "attachments.zip"
	}
	return addAttachment(writer, zipName, buffer.Bytes())
}

func addAttachment(writer *multipart.Writer, fileName string, data []byte) error {
	header := map[string][]string{
		"Content-Type":              {"application/octet-stream"},
		"Content-Transfer-Encoding": {"base64"},
		"Content-Disposition":       {fmt.Sprintf(`attachment; filename="%s"`, filepath.Base(fileName))},
	}
	part, err := writer.CreatePart(header)
	if err != nil {
		return err
	}
	encoded := make([]byte, base64.StdEncoding.EncodedLen(len(data)))
	base64.StdEncoding.Encode(encoded, data)
	_, err = part.Write(encoded)
	return err
}

func readFileOrURL(path string) ([]byte, error) {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		resp, err := http.Get(path)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		return io.ReadAll(resp.Body)
	}
	return os.ReadFile(path)
}

func replaceValues(content string, values map[string]string) string {
	for key, value := range values {
		content = strings.ReplaceAll(content, "${"+key+"}", value)
		content = strings.ReplaceAll(content, "{{"+key+"}}", value)
	}
	return content
}

func (c *Client) eliminate(source []string) []string {
	if c.blacklist == nil {
		return source
	}
	black := map[string]bool{}
	for _, address := range c.blacklist.GetBlacklist() {
		black[address] = true
	}
	result := make([]string, 0, len(source))
	for _, address := range source {
		if !black[address] {
			result = append(result, address)
		}
	}
	return result
}
