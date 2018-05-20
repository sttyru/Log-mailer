package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/smtp"
	"os"
	"strconv"
	"strings"
	"time"
)

func main() {
	var createJSON, emptyJSON bool
	var jsonfile string
	flag.BoolVar(&createJSON, "generate", false, "Generate configuration file interactively.")
	flag.BoolVar(&emptyJSON, "empty", false, "Generate an empty configuration file. Use with -generate.")
	flag.StringVar(&jsonfile, "conf", "configuration.json", "Configuration file to load.")
	flag.Parse()
	if createJSON {
		reader := bufio.NewReader(os.Stdin)
		if r := ask("Location to store configuration file (default: ./" + jsonfile + "): "); strings.TrimSpace(r) != "" {
			jsonfile = r
		}
		if fStat, err := os.Stat(jsonfile); err == nil && !fStat.IsDir() {
			fmt.Printf("Configuration file (%s) exists, overwrite? (y/n): ", jsonfile)
			if r, _ := reader.ReadString('\n'); strings.ToLower(strings.TrimSpace(r)) != "y" {
				return
			}
			err := os.Remove(jsonfile)
			if err != nil {
				log.Fatalln(err)
			}
		}
		f, err := os.OpenFile(jsonfile, os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatalln(err)
		}
		defer f.Close()
		if emptyJSON {
			j, err := json.MarshalIndent(&Config{}, "", "\t")
			if err != nil {
				log.Fatalln(err)
			}
			fmt.Fprintf(f, "%s", j)
			fmt.Printf("Empty configuration file generated: %s.\n", jsonfile)
			return
		}
		c := &Config{
			From: EmailStruct{
				Name:  ask("Enter \"From\" name:\t"),
				Email: ask("Enter \"From\" email:\t"),
			},
			To: EmailStruct{
				Name:  ask("Enter \"To\" name:\t"),
				Email: ask("Enter \"To\" email:\t"),
			},
			Subject: ask("Enter subject:\t\t"),
			Server:  ask("Enter SMTP server:\t"),
			Port:    ask("Port:\t\t\t"),
			Credentials: Credentials{
				Username: ask("Username:\t\t"),
				Password: ask("Password:\t\t"),
			},
			Logs:     ask("Location of logs:\t"),
			Interval: ask("Interval:\t\t"),
			Reset:    ask("Reset log file? (y/n):\t"),
		}
		if c.Reset != "y" {
			c.Reset = "false"
		} else {
			c.Reset = "true"
		}
		j, err := json.MarshalIndent(c, "", "\t")
		if err != nil {
			log.Fatalln(err)
		}
		fmt.Fprintf(f, "%s", j)
		fmt.Printf("Configuration file generated: %s.\n", jsonfile)
		return
	}
	if _, err := os.Stat(jsonfile); err != nil {
		log.Fatalf("Unable to find configuration file (%s).\n", jsonfile)
		return
	}
	data, err := ioutil.ReadFile(jsonfile)
	if err != nil {
		log.Fatalln(err)
	}
	c := &Config{}
	err = json.Unmarshal(data, c)
	if err != nil {
		log.Println(err)
		return
	}
	var (
		from     = fmt.Sprintf(`"%s" <%s>`, c.From.Name, c.From.Email)
		to       = fmt.Sprintf(`"%s" <%s>`, c.To.Name, c.To.Email)
		server   = c.Server
		port     = c.Port
		user     = c.Credentials.Username
		pass     = c.Credentials.Password
		sub      = c.Subject
		logs     = c.Logs
		interval = c.Interval
		reset, e = strconv.ParseBool(c.Reset)
		message  = ""
	)
	if e != nil {
		log.Fatalln(e)
	}
	if interval[0] == '+' || interval[0] == '-' {
		interval = strings.Replace(interval, string(interval[0]), "", -1)
	}
	repeat(func() {
		if i, _ := os.Stat(logs); !(i.Size() > 0) {
			log.Printf("\n%s\n\n", "Log file is empty.")
			return
		}
		message = ""
		headers := make(map[string]string)
		headers["From"] = from
		headers["To"] = to
		headers["Subject"] = sub
		headers["MIME-version"] = "1.0;\nContent-Type: text/html; charset=\"UTF-8\";"
		for title, data := range headers {
			message += fmt.Sprintf("%s: %s\r\n", title, data)
		}
		message += "\r\n"
		file, err := os.Open(logs)
		if err != nil {
			log.Println(err)
			return
		}
		defer file.Close()
		message += "<div style=\"font-family: monospace;background: #ecf0f1;padding: 20px;border-radius: 9px;font-size: 150%;margin: 30px;\">"
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			message += scanner.Text() + "<br>"
		}
		message += "<br><br>Generated by <a href=\"https://github.com/muhammadmuzzammil1998/Log-mailer\">Log Mailer</a> on " + time.Now().Format(time.RFC1123Z) + "</div>"
		err = smtp.SendMail(
			server+":"+port,
			smtp.PlainAuth("", user, pass, server),
			headers["From"],
			[]string{headers["To"]},
			[]byte(message),
		)
		if err != nil {
			log.Println(err)
			return
		}
		log.Printf("\n%s\n\n", message)
		if reset {
			if err := os.Remove(logs); err != nil {
				log.Println(err)
			}
			if _, err := os.Create(logs); err != nil {
				log.Println(err)
			}
		}
	}, interval)
}
func repeat(f func(), interval string) {
	f()
	d, err := time.ParseDuration(interval)
	if err != nil {
		log.Fatalln(err)
	}
	for range time.Tick(d) {
		f()
	}
}
func ask(s string) string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print(s)
	r, _, _ := reader.ReadLine()
	return string(r)
}
