package main

import (
	"fmt"
    "io/ioutil"
	"log"
    "strings"
	"net/http"
    "encoding/json"
    "database/sql"
    _ "github.com/go-sql-driver/mysql"
)


type Volume struct {
    Id int
    Name string
}

//Handle all requests
func Handler(response http.ResponseWriter, request *http.Request){
    response.Header().Set("Content-type", "text/html")
    webpage, err := ioutil.ReadFile("index.html")
    if err != nil {
    http.Error(response, fmt.Sprintf("home.html file error %v", err), 500)
    }
    fmt.Fprint(response, string(webpage));
}
// Connection info for MariaDB/MySQL
const (
    DB_HOST = "tcp(myserverorip:3306)"
    DB_NAME = "mydbname"
    DB_USER = "mydbuser"
    DB_PASS = "mydbpass"
)

// Respond to URLs of the form /generic/...
func APIHandler(response http.ResponseWriter, request *http.Request){

    //Connect to database
    dsn := DB_USER + ":" + DB_PASS + "@" + DB_HOST + "/" + DB_NAME + "?charset=utf8"
    db, err := sql.Open("mysql", dsn)
	if err != nil {
		fmt.Println(err.Error())
	}
	defer db.Close()

	// Open doesn't open a connection. Validate DSN data:
	err = db.Ping()
	if err != nil {
		fmt.Println(err.Error())
	}

    //set mime type to JSON
    response.Header().Set("Content-type", "application/json")
    
/*
	err := request.ParseForm()
	if err != nil {
		http.Error(response, fmt.Sprintf("error parsing url %v", err), 500)
	}
*/
	var result = make([]string,1000)

    switch request.Method {
        case "GET":
	        stmt, err := db.Prepare("select * from mydbname.VOLUMES")
            	if err != nil{
              		fmt.Print( err );
             	}
				rows, err := stmt.Query()

				if err != nil {
					fmt.Print( err )
				}

             	i := 0
             
			 	for rows.Next() {
              		var name string
              		var id int
              		err = rows.Scan( &id, &name )
              		volume := &Volume{Id: id,Name:name}
                		b, err := json.Marshal(volume)
						 fmt.Println(b)
                		if err != nil {
                    		fmt.Println(err)
                    		return
                		}
              		result[i] = fmt.Sprintf("%s", string(b))
              		i++
        		}
            result = result[:i]

        case "POST":
            name := request.PostFormValue("name")
            st, err := db.Prepare("INSERT INTO VOLUMES(name) VALUES(?)")
            	if err != nil{
                fmt.Print( err );
              	}
            res, err := st.Exec(name)
              	if err != nil {
                	fmt.Print( err )
              	}

              	if res!=nil{
                  	result[0] = "true"
             	}
            result = result[:1]

        case "PUT":
        	name := request.PostFormValue("name")
            id := request.PostFormValue("id")

            st, err := db.Prepare("UPDATE VOLUMES SET name=? WHERE id=?")
             	if err != nil{
              	fmt.Print( err );
             	}
             	res, err := st.Exec(name,id)
             	if err != nil {
              	fmt.Print( err )
             	}

             	if res!=nil{
                 	result[0] = "true"
             	}
            	result = result[:1]
        case "DELETE":
            id := strings.Replace(request.URL.Path,"/api/","",-1)
            st, err := db.Prepare("DELETE FROM VOLUMES WHERE id=?")
             	if err != nil{
              	fmt.Print( err );
             	}
             	res, err := st.Exec(id)
             	if err != nil {
              	fmt.Print( err )
             	}

             	if res!=nil{
                	 result[0] = "true"
            	 }
            	result = result[:1]

        	default:
    	}
    
    json, err := json.Marshal(result)
    if err != nil {
        fmt.Println(err)
        return
    }

	// Send the text diagnostics to the client.
    fmt.Fprintf(response,"%v",string(json))
	fmt.Fprintf(response, " request.URL.Path   '%v'\n", request.Method)
    db.Close()
}


func main(){
	port := "1234"
    var err string

	mux := http.NewServeMux()
	mux.Handle("/api/", http.HandlerFunc( APIHandler ))
	mux.Handle("/", http.HandlerFunc( Handler ))

	// Start listing on a given port with these routes on this server.
	log.Print("Listening on port " + port + " ... ")
	errs := http.ListenAndServe(":" + port, mux)
	if errs != nil {
		log.Fatal("ListenAndServe error: ", err)
	}
}
