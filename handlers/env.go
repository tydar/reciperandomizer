package handlers

import (
	"context"
	"errors"
	"html/template"
	"net/http"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/tydar/reciperandomizer/models"
)

type Recipes interface {
	Create(ctx context.Context, title, book, notes string, page int, lastMade time.Time) error
	Update(ctx context.Context, recipe models.Recipe) error
	All(ctx context.Context) ([]models.Recipe, error)
	GetById(ctx context.Context, id int) (models.Recipe, error)
	GetRandom(ctx context.Context) (models.Recipe, error)
	Delete(ctx context.Context, id int) error
}

type Env struct {
	recipes   Recipes
	templates map[string]*template.Template
}

func NewEnv(pool *pgxpool.Pool) *Env {
	return &Env{
		recipes:   models.NewRecipeModel(pool),
		templates: make(map[string]*template.Template),
	}
}

// AddTemplate adds a new template to the map templates with key
// `key` and files `files` (passed directly to template.ParseFiles
func (e *Env) AddTemplate(key string, files ...string) error {
	_, prs := e.templates[key]
	if prs {
		return errors.New("template with name already exists")
	}

	e.templates[key] = template.Must(template.ParseFiles(files...))
	return nil
}

func (e *Env) ExecuteTemplate(key string, w http.ResponseWriter, data interface{}) error {
	return e.templates[key].ExecuteTemplate(w, "base", data)
}
