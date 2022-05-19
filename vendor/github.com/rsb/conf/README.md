# Conf
Configuration package used for handling environment variables


```
import "github.com/rsb/conf"
```

## Usage
Make sure you have your environment variables set

```bash
export MYAPP_SOME_API_URL=localhost:5000
export MYAPP_SOME_API_TIMEOUT=5s
export MYAPP_DB_HOST=localhost
export MYAPP_DB_USER=postgres
export MYAPP_DB_NAME=my-db
export MYAPP_DB_PASSWORD=abc
export MYAPP_CODES="codeA:A,codeB:B,codeC:C"
export MYAPP_ID_LIST="id1,id2,id3"
```


Define your config struct and use it to initialize your app

```
package main

import(
	"fmt"
	"log"
	"time"
	
	"github.com/rsb/conf"
)


type AppConfig struct {
	SomeAPI
	DB
	Debug bool
  Codes map[string]string
	IDList []string
}

type SomeAPI struct {
  URL string	
	Timeout time.Duration
}

type DB struct {
	Name string
  Port int	
	User string
	Password string
  	
}

func main() {
  var config AppConfig	

	if err := conf.ProcessEnv(&config, "MYAPP"); err != nil {
		log.Fatal(err.Error())
  }
  
	fmt.Println(config.SomeAPI.URL)
	fmt.Println(config.SomeAPI.Timeout)

	...
}

```