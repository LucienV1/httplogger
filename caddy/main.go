package webhooklogging

import (
    "encoding/json"
    "fmt"
    "io"
    "net/http"

    "github.com/caddyserver/caddy/v2"
    "github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
    "github.com/caddyserver/caddy/v2/modules/caddyhttp"
    "github.com/disgoorg/disgo/webhook"
    agents "github.com/monperrus/crawler-user-agents"
)

var client webhook.Client

func init() {
    caddy.RegisterModule(DiscordLogging{})
}

type DiscordLogging struct {
    WebhookURL     string `json:"webhook_url,omitempty"`
    ShowAllHeaders bool   `json:"show_all_headers,omitempty"`
    IncludeBots    bool   `json:"include_bots,omitempty"`
}

func (DiscordLogging) CaddyModule() caddy.ModuleInfo {
    return caddy.ModuleInfo{
        ID:  "http.handlers.discord_logging",
        New: func() caddy.Module { return new(DiscordLogging) },
    }
}

func (d *DiscordLogging) Provision(ctx caddy.Context) error {
    var err error
    client, err = webhook.NewWithURL(d.WebhookURL)
    if err != nil {
        return fmt.Errorf("failed to initialize webhook client: %v", err)
    }
    return nil
}

func (d *DiscordLogging) Validate() error {
    if d.WebhookURL == "" {
        return fmt.Errorf("webhook_url is required")
    }
    return nil
}

func (d *DiscordLogging) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
    var isBot string
    if !d.IncludeBots {
        if agents.IsCrawler(r.UserAgent()) || r.URL.Path == "/robots.txt" || r.URL.Path == "/favicon.ico" {
            w.WriteHeader(http.StatusForbidden)
            return nil
        }
        isBot = "No"
    } else {
        if agents.IsCrawler(r.UserAgent()) {
            isBot = "Yes"
        } else {
            isBot = "No"
        }
    }

    if r.URL.Path == "/robots.txt" || r.URL.Path == "/favicon.ico" {
        return nil
    }

    var remote string
    err1 := next.ServeHTTP(w, r)
    if r.Header.Get("Cf-Connecting-Ip") == "" {
        if r.Header.Get("X-Forwarded-For") == "" {
            remote = r.RemoteAddr
        } else {
            remote = r.Header.Get("X-Forwarded-For")
        }
    } else {
        remote = r.Header.Get("Cf-Connecting-Ip")
    }

    var country string
    if r.Header.Get("CF-IPCountry") != "" {
        country = "\n\t\t\\- Country: " + r.Header.Get("CF-IPCountry")
    }

    body, err := io.ReadAll(r.Body)
    if err != nil {
        return nil
    }
    bodyString := string(body)

    var headers string
    if d.ShowAllHeaders {
        headersJSON, err := json.MarshalIndent(r.Header, "", "  ")
        if err != nil {
            return nil
        }
        headers = `
        \- Headers: 
` + "```" + `
` + string(headersJSON) + `
` + "```"
    }

    client.CreateContent(`HTTP REQUEST:
        \- Method: ` + r.Method + `
        \- URL: ` + r.URL.String() + `
        \- User-Agent: ` + r.UserAgent() + `
        \- Is Bot: ` + isBot + `
        \- Remote Address: ` + remote + `
        \- Http Version: ` + r.Proto + `
        \- Host: ` + r.Host + `
        \- Referer: ` + r.Referer() + `
        \- Date and Time: ` + r.Header.Get("Date") + country + `
        \- Lookup IP: https://ip-lookup.net/?` + remote + `
        \- Request Body: ` + bodyString + headers,
    )

    if err1 != nil {
        return err1
    }
    return nil
}

func (d *DiscordLogging) UnmarshalCaddyfile(disp *caddyfile.Dispenser) error {
    for disp.Next() {
        for disp.NextBlock(0) {
            switch disp.Val() {
            case "webhook_url":
                if !disp.NextArg() {
                    return disp.ArgErr()
                }
                d.WebhookURL = disp.Val()
            case "show_all_headers":
                d.ShowAllHeaders = true
            case "include_bots":
                d.IncludeBots = true
            default:
                return disp.Errf("unrecognized directive: %s", disp.Val())
            }
        }
    }
    return nil
}

var (
    _ caddy.Provisioner           = (*DiscordLogging)(nil)
    _ caddy.Validator             = (*DiscordLogging)(nil)
    _ caddyhttp.MiddlewareHandler = (*DiscordLogging)(nil)
    _ caddyfile.Unmarshaler       = (*DiscordLogging)(nil)
)