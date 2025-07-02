package analysis

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/smtp"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
)

// 发送邮件（支持附件）
func SendEmail(smtpServer string, smtpPort int, user, pass string, to []string, subject, body string, attachPaths []string) error {
	host := smtpServer
	addr := fmt.Sprintf("%s:%d", smtpServer, smtpPort)
	msg := bytes.NewBuffer(nil)
	writer := multipart.NewWriter(msg)
	boundary := writer.Boundary()
	// 邮件头
	headers := make(map[string]string)
	headers["From"] = user
	headers["To"] = strings.Join(to, ", ")
	headers["Subject"] = mime.QEncoding.Encode("utf-8", subject)
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "multipart/mixed; boundary=" + boundary
	for k, v := range headers {
		fmt.Fprintf(msg, "%s: %s\r\n", k, v)
	}
	fmt.Fprintf(msg, "\r\n")
	// 正文
	bodyHeader := make(textproto.MIMEHeader)
	bodyHeader.Set("Content-Type", "text/plain; charset=utf-8")
	bodyWriter, _ := writer.CreatePart(bodyHeader)
	qp := quotedprintable.NewWriter(bodyWriter)
	qp.Write([]byte(body))
	qp.Close()
	// 附件
	for _, path := range attachPaths {
		f, err := os.Open(path)
		if err != nil {
			continue
		}
		defer f.Close()
		partHeader := make(textproto.MIMEHeader)
		partHeader.Set("Content-Type", "application/octet-stream")
		partHeader.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filepath.Base(path)))
		part, _ := writer.CreatePart(partHeader)
		io.Copy(part, f)
	}
	writer.Close()
	// 发送
	tlsconfig := &tls.Config{ServerName: host, InsecureSkipVerify: true}
	smtpAuth := smtp.PlainAuth("", user, pass, host)
	conn, err := tls.Dial("tcp", addr, tlsconfig)
	if err != nil {
		return err
	}
	c, err := smtp.NewClient(conn, host)
	if err != nil {
		return err
	}
	if err = c.Auth(smtpAuth); err != nil {
		return err
	}
	if err = c.Mail(user); err != nil {
		return err
	}
	for _, addr := range to {
		if err = c.Rcpt(addr); err != nil {
			return err
		}
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	_, err = w.Write(msg.Bytes())
	if err != nil {
		return err
	}
	w.Close()
	c.Quit()
	return nil
}
