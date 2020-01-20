package server

import (
	"html/template"
	"net/http"
	"net/http/pprof"
	"time"
)

func getDebugHandler(r *runtime) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	mux.Handle("/debug/pprof/block", pprof.Handler("block"))
	mux.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
	mux.Handle("/debug/pprof/heap", pprof.Handler("heap"))
	mux.Handle("/debug/pprof/threadcreate", pprof.Handler("threadcreate"))

	mux.HandleFunc("/info", info(r))

	return mux
}

func info(rt *runtime) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")

		data := struct {
			PageTitle string
			Tags      map[string]string
			StartTime string
		}{
			PageTitle: "Server Tags",
			Tags:      rt.tags,
			StartTime: rt.startTime.Format(time.RFC3339),
		}

		if err := infoTmpl.Execute(w, data); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}

var infoTmpl = template.Must(template.New("info").Parse(`<html>
<head>
<title>{{.PageTitle}}</title>
<style>
.profile-name{
	display:inline-block;
	width:6rem;
}
</style>
</head>
<body>
<br>
<h3>Start Time</h3>
<ul><li>{{ .StartTime }}</li></ul>
<h3>Tags</h3>
<ul>
{{ range $key, $value := .Tags }}
   <li><strong>{{ $key }}</strong> = {{ $value }}</li>
{{ end }}
</ul>
</body>
</html>
`))
