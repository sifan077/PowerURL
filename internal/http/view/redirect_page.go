package view

import (
	"bytes"
	"html/template"
)

// RedirectPageData provides the dynamic fields required by the redirect template.
type RedirectPageData struct {
	Title        string
	Code         string
	TargetURL    string
	ContinueURL  string
	Mode         string
	TimerSeconds int
	Token        string
}

var redirectPageTmpl = template.Must(template.New("redirect_page").Parse(`
<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="utf-8" />
	<meta name="viewport" content="width=device-width, initial-scale=1" />
	<title>{{if .Title}}{{.Title}}{{else}}Redirecting...{{end}}</title>
	<style>
		:root {
			--bg: #090a0f;
			--card: rgba(255, 255, 255, 0.05);
			--border: rgba(255, 255, 255, 0.15);
			--text: #e7ecff;
			--muted: #a1acc5;
			--accent: #7dd3fc;
			--accent-strong: #38bdf8;
			font-family: "Inter", -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
		}
		* { box-sizing: border-box; }
		body {
			margin: 0;
			min-height: 100vh;
			display: flex;
			align-items: center;
			justify-content: center;
			background: radial-gradient(circle at 20% 20%, #111827, #030712 60%);
			color: var(--text);
		}
		.card {
			background: var(--card);
			border: 1px solid var(--border);
			border-radius: 18px;
			padding: 32px;
			width: min(520px, 92vw);
			box-shadow: 0 45px 100px rgba(0,0,0,0.35);
			backdrop-filter: blur(18px);
		}
		h1 {
			font-size: 1.5rem;
			margin-bottom: 6px;
		}
		p {
			color: var(--muted);
			margin-top: 0;
		}
		.destination {
			margin: 24px 0;
			padding: 18px;
			border-radius: 14px;
			background: rgba(125, 211, 252, 0.07);
			border: 1px solid rgba(125, 211, 252, 0.25);
			word-break: break-all;
		}
		.destination-label {
			font-size: 0.82rem;
			text-transform: uppercase;
			letter-spacing: 0.08em;
			color: var(--muted);
			margin-bottom: 8px;
		}
		.actions {
			display: flex;
			align-items: center;
			gap: 12px;
			margin-top: 24px;
			flex-wrap: wrap;
		}
		a.button {
			display: inline-flex;
			align-items: center;
			justify-content: center;
			padding: 0 28px;
			height: 48px;
			border-radius: 999px;
			background: linear-gradient(120deg, var(--accent), var(--accent-strong));
			color: #050708;
			font-weight: 600;
			text-decoration: none;
			transition: transform 0.15s ease, opacity 0.15s ease;
		}
		a.button:hover {
			transform: translateY(-1px);
			opacity: 0.92;
		}
		.timer {
			font-size: 0.95rem;
			color: var(--muted);
		}
		.meta {
			margin-top: 16px;
			font-size: 0.85rem;
			color: rgba(231, 236, 255, 0.65);
		}
	</style>
</head>
<body>
	<div class="card">
		<h1>You’re almost there</h1>
		<p>Short link <strong>/{{.Code}}</strong> resolves to:</p>

		<div class="destination">
			<div class="destination-label">Destination</div>
			<div>{{.TargetURL}}</div>
		</div>

		{{if eq .Mode "timer"}}
		<div class="timer">
			Redirecting in <span id="countdown">{{if gt .TimerSeconds 0}}{{.TimerSeconds}}{{else}}3{{end}}</span>s…
		</div>
		{{else if eq .Mode "click"}}
		<div class="timer">Review the target above and proceed when ready.</div>
		{{end}}

		<div class="actions">
			<a id="cta" class="button" href="{{if .ContinueURL}}{{.ContinueURL}}{{else}}{{.TargetURL}}{{end}}">
				{{if eq .Mode "click"}}Continue
				{{else if eq .Mode "timer"}}Skip waiting
				{{else}}Go now{{end}}
			</a>
		</div>

		<div class="meta">Token: {{if .Token}}{{.Token}}{{else}}–{{end}}</div>
	</div>

	{{if eq .Mode "timer"}}
	<script>
		(function() {
			const startSeconds = {{if gt .TimerSeconds 0}}{{.TimerSeconds}}{{else}}3{{end}};
			let remaining = startSeconds;
			const countdown = document.getElementById("countdown");
			const cta = document.getElementById("cta");
			const target = {{(or .ContinueURL .TargetURL) | js}};

			const tick = () => {
				remaining -= 1;
				if (remaining <= 0) {
					window.location.assign(target);
					return;
				}
				if (countdown) {
					countdown.textContent = remaining.toString();
				}
				setTimeout(tick, 1000);
			};
			setTimeout(tick, 1000);
			if (countdown) {
				countdown.textContent = remaining.toString();
			}
		})();
	</script>
	{{end}}
</body>
</html>
`))

// RenderRedirectPage expands the redirect page template with the provided data.
func RenderRedirectPage(data RedirectPageData) (string, error) {
	if data.Title == "" {
		data.Title = "Redirecting..."
	}
	var buf bytes.Buffer
	if err := redirectPageTmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
