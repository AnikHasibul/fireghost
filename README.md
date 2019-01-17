# FireGhost
A tool to check and sniff firebase config. Also generates html page to dump firebase databases.

# Installation:

Install directly with `go install`

```sh
$ go install -v github.com/anikhasibul/fireghost
```

# Usage:

This command will sniff the config of given target.
To dump the database, go to `http://localhost:1339`
```txt
$ fireghost -t https://hybrid.ai

// Initialize Firebase
  var config = {
    apiKey: "AIzaSyDXg8YMLeSK3nCOnl_-rnyygaO__PtkvQQ",
    authDomain: "just-nova-118913.firebaseapp.com",
    databaseURL: "https://just-nova-118913.firebaseio.com",
    storageBucket: "just-nova-118913.appspot.com",
    messagingSenderId: "887831387331"
  };
  firebase.initializeApp(config);
Serving on localhost:1339
```


For more uses:

```txt
$ fireghost -help
Usage of fireghost:
  -p int
        port to listen. (default 1339)
  -s    Serve the generated page. (default true)
  -t string
        target host.
  -w    Print result to stdout (default true)
```

> If you face any kind of issues with `fireghost`, feel free to open an issue or email me at anikhasibul@outlook.com
