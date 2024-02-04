package main

import (
    "fmt"
    "log"
    "strconv"
    "time"
    "os"
    "database/sql"

    _ "github.com/mattn/go-sqlite3"
    "github.com/gofiber/fiber/v2"
    "github.com/wneessen/go-mail"
    "gopkg.in/ini.v1"
    "github.com/robfig/cron"
)


const (
    STATUS_NOTIFY_NONE    = 0
    STATUS_NOTIFY_PENDING = 1
    STATUS_NOTIFY_DONE    = 2
)

type Event struct {
    Id     uint64 `json:"id"`
    Name   string `json:"name"`
    Date   string `json:"date"`
    Phone  string `json:"phone"`
    Email  string `json:"email"`
    Title  string `json:"title"`
    Send   string `json:"send"`
    Notes  string `json:"notes"`
    Begin  string `json:"begin"`
    End    string `json:"end"`
    Notify bool   `json:"notify"`
    status int
}

type SMTPConfig struct {
    host string
    port string
    username string
    password string
}

var email SMTPConfig
var db *sql.DB

func send_notification_email(to string, message string) bool {
    m := mail.NewMsg()

    m.From("erinnerung@waikiki.de")
    m.To(to)

    m.Subject("Terminerinnerung")
    m.SetBodyString(mail.TypeTextPlain, message)

    port, _ := strconv.Atoi(email.port)

    c, _ := mail.NewClient(email.host, mail.WithPort(port), mail.WithSMTPAuth(mail.SMTPAuthPlain), mail.WithUsername(email.username), mail.WithPassword(email.password))
    err := c.DialAndSend(m)
    if err != nil {
        fmt.Println(err)

        return false
    }

    return true
}

func check_for_notifications() {
    now  := time.Now()
    check_range_begin := now.Add(12 * time.Hour)
    check_range_end   := now.Add(48 * time.Hour)

    rows, err := db.Query("SELECT rowid, * FROM events WHERE date BETWEEN ? AND ?;", now.Format("2006-01-02"), check_range_end.Format("2006-01-02"))
    if err != nil {
        fmt.Println(err)

        return
    }

    type update_info struct {
        id   uint64
        date string
    }

    var updates[] update_info

    for rows.Next() {
        var event Event
        err := rows.Scan(&event.Id, &event.Name, &event.Date, &event.Phone, &event.Email, &event.Title, &event.Send, &event.Notes, &event.Notify, &event.Begin, &event.End, &event.status)
        if err != nil {
            fmt.Println(err.Error())
        }

        if event.Email == "" {
            continue
        }

        event_date_string := fmt.Sprintf("%s %s", event.Date, event.Begin)
        event_time, err := time.Parse("2006-01-02 15:04", event_date_string)
        if err != nil {
            fmt.Println(err)

            return
        }

        if event.Notify {
            if event_time.Before(check_range_begin) || event_time.After(check_range_end) {
                continue
            }

            var title = event.Name

            if event.Title != "" {
                title = event.Title
            }

            body := fmt.Sprintf("Waikiki Oase Tattoo & PMU *\nHallo %s,\nIch freue mich, dich am %s um %s Uhr zu deinem Termin begrüßen zu dürfen!", title, event_time.Format("02.01.2006"), event.Begin)

            fmt.Println(body)

            if send_notification_email(event.Email, body) == true {
                updates = append(updates, update_info{event.Id, now.Format("02.01.2006 15:04")})
            }
        }
    }

    rows.Close()

    for _, update := range updates {
        statement, _ := db.Prepare("UPDATE events SET send = ?, status = ? WHERE rowid = ?;")
        _, err := statement.Exec(update.date, STATUS_NOTIFY_DONE, update.id)

        if err != nil {
            fmt.Println(err)
        }

        statement.Close()
    }
}

func main() {
    options, err := ini.Load("config")
    if err != nil {
        fmt.Printf("Could not load config: %v", err)

        return
    }

    section := options.Section("email")
    email.host = section.Key("host").String()
    email.port = section.Key("port").String()
    email.username = section.Key("username").String()
    email.password = section.Key("password").String()

    os.Mkdir("database", os.ModePerm)

    db, err = sql.Open("sqlite3", "./database/events.db")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close();

    statement, err := db.Prepare("CREATE TABLE IF NOT EXISTS events (name TEXT, date TEXT, phone TEXT, email TEXT, title TEXT, send TEXT, notes TEXT, notify NUMERIC, begin TEXT, end TEXT, status INTEGER);")
    if err != nil {
        log.Fatal(err)
    }
    statement.Exec()
    statement.Close()

    c := cron.New();
    c.AddFunc("@every 2h", check_for_notifications)
    c.Start()
    defer c.Stop()

    check_for_notifications()

    app := fiber.New()

    app.Static("/", "./public")

    app.Get("/events/:date", func(c *fiber.Ctx) error {
        date_string := fmt.Sprintf("%s-01", c.Params("date"))
        rows, err := db.Query("SELECT rowid, * FROM events WHERE strftime('%%Y-%%m', date) = strftime('%%Y-%%m', ?);", date_string)

        if err != nil {
            fmt.Printf(err.Error())

            return c.JSON("null")
        }

        var results []Event;
        for rows.Next() {
            var event Event
            err := rows.Scan(&event.Id, &event.Name, &event.Date, &event.Phone, &event.Email, &event.Title, &event.Send, &event.Notes, &event.Notify, &event.Begin, &event.End, &event.status)
            if err != nil {
                fmt.Println(err.Error())

                return c.JSON("null")
            }

            results = append(results, event)
        }

        rows.Close();

        return c.JSON(results)
    })

    app.Post("/event", func(c *fiber.Ctx) error {
        event := new(Event)
        err := c.BodyParser(event)
        if err!= nil {
            fmt.Println(err)

            return c.SendString("0")
        }

        statement, _ := db.Prepare("INSERT INTO events (name, date, phone, email, title, send, notes, notify, begin, end, status) VALUES(?, ?, ?, ?, ?, '', ?, ?, ?, ?, ?);")

        notify := 0
        if event.Notify {
            notify = 1
        }

        status := STATUS_NOTIFY_NONE
        if notify == 1 {
            status = STATUS_NOTIFY_PENDING
        }

        res, err := statement.Exec(event.Name, event.Date, event.Phone, event.Email, event.Title, event.Notes, notify, event.Begin, event.End, status);

        if err != nil {
            fmt.Println(err)
            return c.SendString("0")
        }

        id, _ := res.LastInsertId();
        statement.Close()

        return c.SendString(strconv.FormatInt(id, 10));
    })

    app.Put("/event", func(c *fiber.Ctx) error {
        event := new(Event)
        err := c.BodyParser(event)
        if err!= nil {
            fmt.Println(err)

            return c.SendString("0")
        }

        var status int
        err = db.QueryRow("SELECT status FROM events WHERE rowid = ?;", event.Id).Scan(&status);
        if err != nil {
            fmt.Println(err)

            return c.SendString("0")
        }

        if status != STATUS_NOTIFY_DONE {
            if event.Notify {
                status = 1
            } else {
                status = 0
            }
        }

        statement, _ := db.Prepare("UPDATE events SET name = ?, date = ?, phone = ?, email = ?, title = ?, notes = ?, notify = ?, begin = ?, end = ?, status = ? WHERE rowid = ?;")
        statement.Exec(event.Name, event.Date, event.Phone, event.Email, event.Title, event.Notes, event.Notify, event.Begin, event.End, status, event.Id)

        statement.Close()
        return c.SendString("updated")
    })

    app.Listen(":3000")
}

