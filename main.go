package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/tydar/reciperandomizer/handlers"
)

func main() {
	dbUrl := os.Getenv("DATABASE_URL")
	servePort := os.Getenv("PORT")
	if servePort == "" {
		servePort = "8080"
	}

	cf, err := pgxpool.ParseConfig(dbUrl)
	if err != nil {
		panic(err)
	}

	fmt.Println(cf.ConnString())

	pool, err := pgxpool.ConnectConfig(context.TODO(), cf)
	if err != nil {
		panic(err)
	}

	env := handlers.NewEnv(pool)

	// full page templates based on base.html
	env.AddTemplate("index", "templates/base.html", "templates/index.html")
	env.AddTemplate("recipe", "templates/base.html", "templates/recipe.html")
	env.AddTemplate("addRecipe", "templates/base.html", "templates/add.html")
	env.AddTemplate("all", "templates/base.html", "templates/all.html")
	env.AddTemplate("search", "templates/base.html", "templates/search.html")

	// partial templates
	env.AddPartial("searchResult", "templates/search_results.html")

	mux := http.NewServeMux()

	mux.HandleFunc("/", env.IndexHandler)
	mux.HandleFunc("/recipe/", env.RecipeHandler)
	mux.HandleFunc("/add/", env.AddHandler)
	mux.HandleFunc("/all/", env.AllHandler)
	mux.HandleFunc("/made/", env.MadeHandler)
	mux.HandleFunc("/recipe/delete/", env.DeleteHandler)
	mux.HandleFunc("/search/", env.SearchHandler)

	http.ListenAndServe(":"+servePort, mux)
}
