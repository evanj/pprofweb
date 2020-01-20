package main

import (
	"flag"
	"io"
	"log"
	"net/http"
	"net/http/pprof"
	"os"
	"path"
	"strings"

	"github.com/google/pprof/driver"
)

const portEnvVar = "PORT"
const defaultPort = "8080"
const maxUploadSize = 32 << 20 // 32 MiB

const pprofFilePath = "/tmp/pprofweb-temp"

const fileFormID = "file"
const uploadPath = "/upload"
const pprofWebPath = "/pprofweb/"

type server struct {
	// serves pprof handlers after it is loaded
	pprofMux *http.ServeMux
}

func (s *server) startHTTP(args *driver.HTTPServerArgs) error {
	s.pprofMux = http.NewServeMux()
	for pattern, handler := range args.Handlers {
		var joinedPattern string
		if pattern == "/" {
			joinedPattern = pprofWebPath
		} else {
			joinedPattern = path.Join(pprofWebPath, pattern)
		}
		s.pprofMux.Handle(joinedPattern, handler)
	}
	return nil
}

func (s *server) servePprof(w http.ResponseWriter, r *http.Request) {
	if s.pprofMux == nil {
		http.Error(w, "must upload profile first", http.StatusInternalServerError)
		return
	}
	s.pprofMux.ServeHTTP(w, r)
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("rootHandler %s %s", r.Method, r.URL.String())
	if r.Method != http.MethodGet {
		http.Error(w, "wrong method", http.StatusMethodNotAllowed)
		return
	}
	if r.URL.Path != "/" {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	w.Write([]byte(rootTemplate))
}

func (s *server) uploadHandlerErrHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("uploadHandler %s %s", r.Method, r.URL.String())
	if r.Method != http.MethodPost {
		http.Error(w, "wrong method", http.StatusMethodNotAllowed)
		return
	}
	err := s.uploadHandler(w, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *server) uploadHandler(w http.ResponseWriter, r *http.Request) error {
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		return err
	}
	uploadedFile, _, err := r.FormFile(fileFormID)
	if err != nil {
		return err
	}
	defer uploadedFile.Close()

	// write the file out to a temporary location
	f, err := os.OpenFile(pprofFilePath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := io.Copy(f, uploadedFile); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := uploadedFile.Close(); err != nil {
		return err
	}

	// start the pprof web handler: pass -http and -no_browser so it starts the
	// handler but does not try to launch a browser
	// our startHTTP will do the appropriate interception
	flags := &pprofFlags{
		args: []string{"-http=localhost:0", "-no_browser", pprofFilePath},
	}
	options := &driver.Options{
		Flagset:    flags,
		HTTPServer: s.startHTTP,
	}
	if err := driver.PProf(options); err != nil {
		return err
	}

	http.Redirect(w, r, pprofWebPath, http.StatusSeeOther)
	return nil
}

func main() {
	s := &server{}

	mux := http.NewServeMux()
	mux.HandleFunc("/", rootHandler)
	mux.HandleFunc(uploadPath, s.uploadHandlerErrHandler)
	mux.HandleFunc(pprofWebPath, s.servePprof)

	// copied from net/http/pprof to avoid relying on the global http.DefaultServeMux
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	port := os.Getenv(portEnvVar)
	if port == "" {
		port = defaultPort
		log.Printf("warning: %s not specified; using default %s", portEnvVar, port)
	}

	addr := ":" + port
	log.Printf("listen addr %s (http://localhost:%s/)", addr, port)
	if err := http.ListenAndServe(addr, mux); err != nil {
		panic(err)
	}
}

const rootTemplate = `<!doctype html>
<html>
<head><title>PProf Web Interface</title></head>
<body>
<h1>PProf Web Interface</h1>
<p>Upload a file to explore it using the Pprof web interface.</p>
<h2>NOTE: Will not work with multiple users</h2>
<p>This is currently a hack: it runs in Google Cloud Run, which will restart instances whenever it wants. This means your state may get lost at any time, and it won't work if there are multiple people using it at the same time.</p>
<p>TODO: Write all the output from pprof out to static files in Google Cloud Storage and serve them from there.</p>

<form method="post" action="` + uploadPath + `" enctype="multipart/form-data">
<p>Upload file: <input type="file" name="` + fileFormID + `"> <input type="submit" value="Upload"></p>
</form>
</body>
</html>
`

// Mostly copied from https://github.com/google/pprof/blob/master/internal/driver/flags.go
type pprofFlags struct {
	args  []string
	s     flag.FlagSet
	usage []string
}

// Bool implements the plugin.FlagSet interface.
func (p *pprofFlags) Bool(o string, d bool, c string) *bool {
	return p.s.Bool(o, d, c)
}

// Int implements the plugin.FlagSet interface.
func (p *pprofFlags) Int(o string, d int, c string) *int {
	return p.s.Int(o, d, c)
}

// Float64 implements the plugin.FlagSet interface.
func (p *pprofFlags) Float64(o string, d float64, c string) *float64 {
	return p.s.Float64(o, d, c)
}

// String implements the plugin.FlagSet interface.
func (p *pprofFlags) String(o, d, c string) *string {
	return p.s.String(o, d, c)
}

// BoolVar implements the plugin.FlagSet interface.
func (p *pprofFlags) BoolVar(b *bool, o string, d bool, c string) {
	p.s.BoolVar(b, o, d, c)
}

// IntVar implements the plugin.FlagSet interface.
func (p *pprofFlags) IntVar(i *int, o string, d int, c string) {
	p.s.IntVar(i, o, d, c)
}

// Float64Var implements the plugin.FlagSet interface.
// the value of the flag.
func (p *pprofFlags) Float64Var(f *float64, o string, d float64, c string) {
	p.s.Float64Var(f, o, d, c)
}

// StringVar implements the plugin.FlagSet interface.
func (p *pprofFlags) StringVar(s *string, o, d, c string) {
	p.s.StringVar(s, o, d, c)
}

// StringList implements the plugin.FlagSet interface.
func (p *pprofFlags) StringList(o, d, c string) *[]*string {
	return &[]*string{p.s.String(o, d, c)}
}

// AddExtraUsage implements the plugin.FlagSet interface.
func (p *pprofFlags) AddExtraUsage(eu string) {
	p.usage = append(p.usage, eu)
}

// ExtraUsage implements the plugin.FlagSet interface.
func (p *pprofFlags) ExtraUsage() string {
	return strings.Join(p.usage, "\n")
}

// Parse implements the plugin.FlagSet interface.
func (p *pprofFlags) Parse(usage func()) []string {
	p.s.Usage = usage
	p.s.Parse(p.args)
	args := p.s.Args()
	if len(args) == 0 {
		usage()
	}
	return args
}
