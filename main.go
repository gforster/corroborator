package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"strconv"
	//  "strings"
	"database/sql"
	"encoding/json"
	"net/http"

	_ "github.com/go-sql-driver/mysql"
)

// STRUCTS
// Here, define the different structs needed.
// These should basically mirror the CMDB MySQL
// structure. Volume is the Main Struct.
type Volume struct {
	Id          int
	Name        string
	Description string
	Server      Server
	AppList     []Apps
	PortList    []Ports
}
type Server struct {
	Id       int
	Name     string
	Location Location
}
type Apps struct {
	Id   int
	Name string
}
type Ports struct {
	Id   int
	Name string
}
type Location struct {
	Id   int
	Name string
}

// HTTP HANDLER
// Setup an http server to deal with requests.
func Handler(response http.ResponseWriter, request *http.Request) {
	response.Header().Set("Content-type", "text/html")
	webpage, err := ioutil.ReadFile("index.html")
	if err != nil {
		http.Error(response, fmt.Sprintf("home.html file error %v", err), 500)
	}
	fmt.Fprint(response, string(webpage))
}

// DB CONNECTION INFO
const (
	DB_HOST = "tcp(***REMOVED***:3306)"
	DB_NAME = "***REMOVED***s"
	DB_USER = "***REMOVED***"
	DB_PASS = "***REMOVED***"
)

// Respond to URLs of the form /api
func APIHandler(response http.ResponseWriter, request *http.Request) {

	// Connect to database
	dsn := DB_USER + ":" + DB_PASS + "@" + DB_HOST + "/" + DB_NAME + "?charset=utf8"
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		fmt.Println(err.Error())
	}
	defer db.Close()

	// Open doesn't open a connection, so we must validate DSN data:
	err = db.Ping()
	if err != nil {
		fmt.Println(err.Error())
	}

	//set mime type to JSON
	response.Header().Set("Content-type", "application/json")

	// We're only going to return one volume at a time. If an array of volumes is ever needed, work with the below:
	//	result := []*Volume{}

	// Switch based on Reques Methods (GET, POST, DELETE, etc.)
	// This should probably be refactored into separate functions down the road
	switch request.Method {
	case "GET":
		// Setup the Endpoint for the request. So, the URI request  will look like:
		// http://servername.med.umich.edu/api?volname=
		volname := request.URL.Query().Get("volname")

		// First Order of Business Is To Build Out The Main Struct
		// In This Case, We Need to Deal With The Volume Struct

		// Setup Vars In The Volume Struct
		var volid int
		var volnm string
		var voldescr string
		var server_id int

		// Get All Non-Array Items First:
		// Since There Will Only Be One Row Returned, QueryRow & Scan
		err := db.QueryRow("SELECT id, name, description, server_id FROM VOLUMES where name = ?", volname).Scan(&volid, &volnm, &voldescr, &server_id)
		if err != nil {
			fmt.Print(err)
		}
		// Create a Temporary Instance Of the Volume Struct
		tmpvolume := Volume{}
		tmpvolume.Id = volid
		tmpvolume.Name = volnm
		tmpvolume.Description = voldescr

		// Setup Vars In The Server Struct
		var srvnm string
		var location_id int

		// Query For Server
		// Again, A Single Row Should Be Returned (one server per volume)
		err = db.QueryRow("SELECT name, location_id FROM SERVERS where id = ?", server_id).Scan(&srvnm, &location_id)
		if err != nil {
			fmt.Print(err)
		}
		//fmt.Println(srvnm)

		// Assign Vars Within The tmpvolume Instance
		tmpvolume.Server.Id = server_id
		tmpvolume.Server.Name = srvnm

		// Setup and Query For Location
		// Another Single Row Query. However, this ends up being part of the Server Struct,
		var locname string

		err = db.QueryRow("SELECT id, name FROM LOCATION where id = ?", location_id).Scan(&location_id, &locname)
		if err != nil {
			fmt.Print(err)
		}

		// location := Location{}
		tmpvolume.Server.Location.Id = location_id
		tmpvolume.Server.Location.Name = locname
		//	fmt.Println(locname)

		// CUSTOMER FACING SERVICES
		// Really, what we need here is to find the Applications on the Volume.
		// However, There *should not* be a Vol_Apps link table since every
		// App should be related to a Customer-facing service which is related
		// to the Volume. A Volume has Customer-Facing Services. Those Customer
		// Services have Apps.

		// APPS
		// This is how we deal with an array instead of the single rows
		var app_id int

		// Separate Query From Scan. Note using db.Query instead of db.QueryRow
		approws, err := db.Query("select app_id from VOL_APPS where volume_id = " + strconv.Itoa(tmpvolume.Id))
		// fmt.Println(tmpvolume.Id)
		if err != nil {
			fmt.Println(err.Error())
			return // only if you want to quit if there are no apps
		}
		// We wil loop over the results and put them in a temporary instance of the Apps{} struct
		for approws.Next() {
			tmpapp := Apps{}
			tmpapp.Id = app_id
			strtmpid := strconv.Itoa(tmpapp.Id)
			err = approws.Scan(&strtmpid)
			// fmt.Println(strtmpid)
			if err != nil {
				fmt.Print(err)
			}
			// Within The Loop, need to Query & Scan for the App's Id & Name
			err = db.QueryRow("SELECT id, name from APPS where id = "+strtmpid).Scan(&tmpapp.Id, &tmpapp.Name)
			if err != nil {
				fmt.Print(err)
			}

			// PORTS
			// This is how we deal with an array Subquery (Ports) of an array (Apps)
			// Still in the Apps Loop, we basically do the same thing as Apps

			var portid int

			portrows, err := db.Query("select port_id from APPS_PORTS where app_id = " + strtmpid)
			if err != nil {
				fmt.Println(err.Error())
				return // only if you want to quit if there are no Ports
			}
			for portrows.Next() {
				tmpport := Ports{}
				tmpport.Id = portid
				strtmpportid := strconv.Itoa(tmpport.Id)
				err = portrows.Scan(&strtmpportid)
				// fmt.Println(strtmpportid)
				if err != nil {
					fmt.Print(err)
				}

				err = db.QueryRow("SELECT id, name from PORTS where id = "+strtmpportid).Scan(&tmpport.Id, &tmpport.Name)
				if err != nil {
					fmt.Print(err)
				}
				// Append the ports to PortList
				tmpvolume.PortList = append(tmpvolume.PortList, tmpport)
			}
			// Append the tmpapp to AppList
			tmpvolume.AppList = append(tmpvolume.AppList, tmpapp)
		}

		// Marshal Json, Tamer of the Wild Wild West

		json, err := json.Marshal(tmpvolume)
		if err != nil {
			fmt.Println(err)
			return
		}

		// Send the text to the client.
		fmt.Fprintf(response, string(json))
		// fmt.Fprintf(response, " request.URL.Path   '%v'\n", request.Method)
		//    db.Close()
	}
}

// The Main Function
func main() {
	// What Port To Run On
	port := "1236"
	var err string

	mux := http.NewServeMux()
	mux.Handle("/api", http.HandlerFunc(APIHandler))
	mux.Handle("/", http.HandlerFunc(Handler))

	// Start listing on a given port with these routes on this server.
	log.Print("Listening on port " + port + " ... ")
	errs := http.ListenAndServe(":"+port, mux)
	if errs != nil {
		log.Fatal("ListenAndServe error: ", err)
	}
}
