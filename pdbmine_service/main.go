package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	//Chi Router
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
)

const (
	//DefaultPort is the default port number for the application
	DefaultPort int = 8077
	//DefaultDB is the file name of the database file
	DefaultDB string = "Database.dat"
)

//Routes sets up the Routes router for the whole service and adds each groups routes seperately
func Routes(db *ProteinDB) *chi.Mux {
	router := chi.NewRouter()
	router.Use(
		//render.SetContentType(render.ContentTypeJSON), //Set content-type headers as application/json
		middleware.Logger, //Log API request calls
		//middleware.Timeout(180000*time.Millisecond),
		//TODO: enable compression
		//middleware.DefaultCompress, //Compress results, mostly gzipping assets and JSON
		middleware.RedirectSlashes, //Redirect slashes to no slash URL versions
		middleware.Recoverer,       //Recover from panics without crashing server
	)

	//Load the routes from each of the endpoint groups
	router.Route("/v1", func(r chi.Router) {
		r.Mount("/api/protein", db.ProteinRoutes())
		r.Mount("/api/database", db.DatabaseRoutes())
		r.Mount("/api/query", db.QueryRoutes())
	})

	return router
}

func main() {
	//Get the port number from the environment variable or default
	sPort := os.Getenv("DIREDB_PORT")
	port, err := strconv.Atoi(sPort)
	if err != nil {
		port = DefaultPort
		log.Printf("DIREDB_PORT is not set or is not valid. Defaulting to %d", port)
	}

	//Get the database from the environment variable or default
	database := os.Getenv("DIREDB_DB")
	if database == "" {
		database = DefaultDB
		log.Printf("DIREDB_DB is not set. Defaulting to %s", database)
	}

	//Load the data from the database
	ProteinDatabase, err := NewProteinDB(database)
	if err != nil {
		log.Fatalf("Error loading the database: %s\n", err.Error())
	}

	err = ProteinDatabase.ConnectToRedis()
	if err != nil {
		log.Println("Unable to connect to Redis")
		log.Print(err)
	} else {
		log.Println("Connected to Redis")
	}

	if ProteinDatabase.Redis != nil {
		defer ProteinDatabase.Redis.Close()
	}

	//Load the routes
	log.Println("Creating Routes")
	router := Routes(ProteinDatabase) //Call function to setup the routes

	walkFunc := func(method string, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		log.Printf("%s %s\n", method, route) //Walk and print out all Routes
		return nil
	}
	err = chi.Walk(router, walkFunc)
	if err != nil {
		log.Panicf("Error walking the routes: %s\n", err.Error())
	}

	//Start the server
	portString := fmt.Sprintf(":%04d", port)
	log.Printf("Starting Server on port %s\n", portString)
	log.Fatal(http.ListenAndServe(portString, router))
}
