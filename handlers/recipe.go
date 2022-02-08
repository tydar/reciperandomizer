package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/tydar/reciperandomizer/models"
)

//--- response models

type recipeResponse struct {
	Id       int
	Title    string
	Book     string
	Notes    []string
	PageNum  int
	LastMade string
}

func recipeModelToRecipeResponse(r models.Recipe) recipeResponse {
	tokNotes := strings.Split(strings.ReplaceAll(r.Notes, "\r", ""), "*")

	// if we have notes, the first item will always be the empty string
	// because we use * as a markdown-style list item
	if len(tokNotes) >= 1 {
		tokNotes = tokNotes[1:]
	}

	// if we only have one entry in tok notes and it is the empty string, discard it
	if len(tokNotes) == 1 && tokNotes[0] == "" {
		tokNotes = []string{}
	}

	// put date in format expected by HTML value prop
	dateString := "2006-01-02"
	lastMadeDateOnly := r.LastMade.Format(dateString)

	return recipeResponse{
		Id:       r.Id,
		Title:    r.Title,
		Book:     r.Book,
		Notes:    tokNotes,
		LastMade: lastMadeDateOnly,
		PageNum:  r.PageNum,
	}
}

type listRecipe struct {
	Id       int
	Title    string
	Book     string
	LastMade string
}

type listResponse []listRecipe

func recipeModelsToListResponse(rs []models.Recipe) listResponse {
	responses := make([]listRecipe, len(rs))
	for i := range rs {
		r := rs[i]
		responses[i] = listRecipe{
			Id:       r.Id,
			Title:    r.Title,
			Book:     r.Book,
			LastMade: r.LastMade.Format("2006-01-02"),
		}
	}
	return responses
}

// -- handlers

func (e *Env) AllHandler(w http.ResponseWriter, r *http.Request) {
	recipes, err := e.recipes.All(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	responseList := recipeModelsToListResponse(recipes)

	templObj := struct {
		Flash   string
		Recipes listResponse
	}{
		Flash:   "",
		Recipes: responseList,
	}

	if err := e.ExecuteTemplate("all", w, templObj); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (e *Env) IndexHandler(w http.ResponseWriter, r *http.Request) {
	recipe, err := e.recipes.GetRandom(r.Context())
	present := true
	if err != nil {
		if err == pgx.ErrNoRows {
			present = false
		} else {
			http.Error(w, fmt.Sprintf("index: %v", err), http.StatusInternalServerError)
		}
	}

	templateObj := struct {
		Recipe  recipeResponse
		Flash   string
		Present bool
	}{
		Recipe:  recipeModelToRecipeResponse(recipe),
		Flash:   "",
		Present: present,
	}

	if err := e.ExecuteTemplate("index", w, templateObj); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (e *Env) RecipeHandler(w http.ResponseWriter, r *http.Request) {
	// slice URL for id & get recipe from DB
	id := r.URL.Path[len("/recipe/"):]
	idInt, err := strconv.Atoi(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	recipe, err := e.recipes.GetById(r.Context(), idInt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	if r.Method == "GET" {

		rr := recipeModelToRecipeResponse(recipe)
		templateObj := struct {
			Recipe recipeResponse
			Flash  string
		}{
			Recipe: rr,
			Flash:  "",
		}

		if err := e.ExecuteTemplate("recipe", w, templateObj); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}

	if r.Method == "POST" {
		dateStr := r.FormValue("date")
		notes := r.FormValue("notes")
		date, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

		recipe.LastMade = date

		// not sure why, but kept getting extra white space from the text area
		// this eliminates that spacing
		recipe.Notes = strings.Join(strings.Fields(notes), " ")

		if err := e.recipes.Update(r.Context(), recipe); err != nil {
			templateObj := struct {
				Recipe recipeResponse
				Flash  string
			}{
				Recipe: recipeModelToRecipeResponse(recipe),
				Flash:  "error: " + err.Error(),
			}

			if err := e.ExecuteTemplate("recipe", w, templateObj); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		}

		templateObj := struct {
			Recipe recipeResponse
			Flash  string
		}{
			Recipe: recipeModelToRecipeResponse(recipe),
			Flash:  "update successful",
		}

		if err := e.ExecuteTemplate("recipe", w, templateObj); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func (e *Env) AddHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		tempObj := struct {
			Flash string
		}{
			Flash: "",
		}
		if err := e.ExecuteTemplate("addRecipe", w, tempObj); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	} else if r.Method == "POST" {
		title := r.FormValue("title")
		book := r.FormValue("book")
		pageNumS := r.FormValue("pageNum")
		dateS := r.FormValue("date")
		notes := r.FormValue("notes")

		pageNum, err := strconv.Atoi(pageNumS)
		if err != nil {
			tempObj := struct {
				Flash string
			}{
				Flash: fmt.Sprintf("bad page input: %s", pageNumS),
			}
			if err := e.ExecuteTemplate("addRecipe", w, tempObj); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		}

		date := time.Time{} // default zero value if no date
		if len(dateS) > 0 {
			date, err = time.Parse("2006-01-02", dateS)
			if err != nil {
				tempObj := struct {
					Flash string
				}{
					Flash: fmt.Sprintf("bad date input: %s", dateS),
				}
				if err := e.ExecuteTemplate("addRecipe", w, tempObj); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
				}
			}
		}

		if err := e.recipes.Create(r.Context(), title, book, notes, pageNum, date); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

		tempObj := struct {
			Flash string
		}{
			Flash: "recipe added successfully!",
		}
		if err := e.ExecuteTemplate("addRecipe", w, tempObj); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func (e *Env) MadeHandler(w http.ResponseWriter, r *http.Request) {
	idS := r.URL.Path[len("/made/"):]

	id, err := strconv.Atoi(idS)
	if err != nil {
		http.Error(w, fmt.Sprintf("/made/%s: bad id: %v", idS, err), http.StatusInternalServerError)
	}
	recipe, err := e.recipes.GetById(r.Context(), id)
	if err != nil {
		http.Error(w, fmt.Sprintf("/made/%s: db error: %v", idS, err), http.StatusInternalServerError)
	}

	recipe.LastMade = time.Now()
	if err := e.recipes.Update(r.Context(), recipe); err != nil {
		http.Error(w, fmt.Sprintf("/made/%s: db error: %v", idS, err), http.StatusInternalServerError)
	}

	templObj := struct {
		Flash  string
		Recipe recipeResponse
	}{
		Flash:  "Marked as made today!",
		Recipe: recipeModelToRecipeResponse(recipe),
	}

	if err := e.ExecuteTemplate("recipe", w, templObj); err != nil {
		http.Error(w, fmt.Sprintf("/made/%s: template error: %v", idS, err), http.StatusInternalServerError)
	}
}

func (e *Env) DeleteHandler(w http.ResponseWriter, r *http.Request) {
	idS := r.URL.Path[len("/recipe/delete/"):]

	id, err := strconv.Atoi(idS)
	if err != nil {
		http.Error(w, fmt.Sprintf("/delete/%s: bad id: %v", idS, err), http.StatusInternalServerError)
	}

	if err := e.recipes.Delete(r.Context(), id); err != nil {
		http.Error(w, fmt.Sprintf("/delete/%s: db error: %v", idS, err), http.StatusInternalServerError)
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (e *Env) SearchHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		if err := e.ExecuteTemplate("search", w, nil); err != nil {
			http.Error(w, fmt.Sprintf("/search/: error: %v", err), http.StatusInternalServerError)
			return
		}
	} else if r.Method == "POST" {
		searchText := r.FormValue("search")
		recipes, err := e.recipes.Search(r.Context(), searchText)
		if err != nil {
			http.Error(w,
				fmt.Sprintf("/search/ term: \"%s\": error: %v", searchText, err),
				http.StatusInternalServerError)
			return
		}

		templObj := struct {
			Recipes []models.Recipe
		}{
			Recipes: recipes,
		}

		if err := e.ExecutePartialTemplate("searchResult", w, templObj); err != nil {
			http.Error(w,
				fmt.Sprintf("/search/ term: \"%s\": error: %v", searchText, err),
				http.StatusInternalServerError)
			return
		}
	}
}
