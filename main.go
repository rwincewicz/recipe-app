package main

import (
	"encoding/json"
	"fmt"
	r "github.com/dancannon/gorethink"
	"github.com/gorilla/mux"
	"io/ioutil"
	"log"
	"net/http"
)

var session *r.Session

type Recipe struct {
	ID          string
	Name        string
	Time        string
	Method      string
	Ingredients []string
}

func main() {
	var err error
	session, err = r.Connect(r.ConnectOpts{
		Address:  "localhost:28015",
		Database: "recipes",
		MaxIdle:  10,
		MaxOpen:  10,
	})
	if err != nil {
		log.Fatalln(err.Error())
	}
	session.SetMaxOpenConns(5)

	setup()

	router := mux.NewRouter().StrictSlash(true)

	router.HandleFunc("/add", addHandler)
	router.HandleFunc("/view/{recipeId}", viewHandler)
	router.HandleFunc("/list", listHandler)
	router.HandleFunc("/edit/{recipeId}", editHandler)
	router.HandleFunc("/delete/{recipeId}", deleteHandler)

	log.Fatal(http.ListenAndServe(":8082", router))
}

func addHandler(w http.ResponseWriter, req *http.Request) {
	input, err := ioutil.ReadAll(req.Body)
	defer req.Body.Close()

	var newRecipe Recipe

	err = json.Unmarshal(input, &newRecipe)
	if err != nil {
		log.Println(err.Error())
	}
	log.Println(newRecipe)

	resp, err := r.DB("recipes").Table("recipes").Insert(newRecipe).Run(session)
	if err != nil {
		log.Println(err.Error())
	}
	defer resp.Close()

	var response map[string]interface{}

	err = resp.One(&response)

	keys := response["generated_keys"].([]interface{})

	log.Println(keys)

	key := keys[0].(string)

	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, key)
}

func viewHandler(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	recipeId := vars["recipeId"]
	log.Println(recipeId)
	res, err := r.DB("recipes").Table("recipes").Get(recipeId).Run(session)
	if err != nil {
		log.Println(err.Error())
	}
	defer res.Close()
	if res.IsNil() {
		log.Println("Could not find record")
		w.WriteHeader(http.StatusNotFound)
		return
	}

	var recipe Recipe

	err = res.One(&recipe)
	if err != nil {
		log.Println("Error handling DB result")
	}
	log.Println(recipe)
	output, err := json.Marshal(recipe)
	if err != nil {
		log.Println(err.Error())
	}

	fmt.Fprintf(w, string(output[:]))
}

func listHandler(w http.ResponseWriter, req *http.Request) {
	res, err := r.DB("recipes").Table("recipes").Run(session)
	if err != nil {
		log.Println(err.Error())
	}
	var recipes []Recipe
	err = res.All(&recipes)
	if err != nil {
		log.Println("Error handling DB result")
	}
	log.Println(recipes)

	output, err := json.Marshal(recipes)
	if err != nil {
		log.Println(err.Error())
	}

	w.Header().Set("Access-Control-Allow-Origin", "http://192.168.56.101")

	fmt.Fprintf(w, string(output[:]))
}

func editHandler(w http.ResponseWriter, req *http.Request) {
	log.Println(req)
}

func deleteHandler(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	recipeId := vars["recipeId"]
	log.Println(recipeId)

	resp, err := r.DB("recipes").Table("recipes").Get(recipeId).Delete().RunWrite(session)
	if err != nil {
		log.Println(err.Error())
	}

	log.Printf("%d row deleted", resp.Deleted)
}

func setup() {
	dbs, err := r.DBList().Run(session)
	if err != nil {
		log.Fatalln(err.Error())
	}

	defer dbs.Close()

	var response []string

	dbs.All(&response)

	log.Println(response)
	log.Println(contains(response, "test"))

	if contains(response, "recipes") {
		log.Println("DB already exists")
	} else {
		_, err := r.DBCreate("recipes").RunWrite(session)
		if err != nil {
			log.Fatalln(err.Error())
		}
	}

	tables, err := r.DB("recipes").TableList().Run(session)
	if err != nil {
		log.Fatalln(err.Error())
	}

	defer tables.Close()

	var tableList []string
	err = tables.All(&tableList)
	if contains(tableList, "recipes") {
		log.Println("Table already exists")
	} else {
		_, err := r.DB("recipes").TableCreate("recipes").RunWrite(session)
		if err != nil {
			log.Fatalln(err.Error())
		}
	}
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
