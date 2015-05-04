package restapi

import (
	"fmt"

	"github.com/ant0ine/go-json-rest/rest"
	"github.com/sebest/hooky/models"
	"github.com/sebest/hooky/store"
	"gopkg.in/mgo.v2/bson"
)

func GetAccount(r *rest.Request) *bson.ObjectId {
	if rv, ok := r.Env["REMOTE_USER"]; ok {
		id := bson.ObjectIdHex(rv.(string))
		return &id
	}
	return nil
}

func GetBase(r *rest.Request) *models.Base {
	if rv, ok := r.Env["MODELS_BASE"]; ok {
		return rv.(*models.Base)
	}
	panic("Missing BaseMiddleware!")
}

type BaseMiddleware struct {
	Store *store.Store
}

func (mw *BaseMiddleware) MiddlewareFunc(next rest.HandlerFunc) rest.HandlerFunc {
	return func(w rest.ResponseWriter, r *rest.Request) {
		db := mw.Store.DB()
		defer db.Session.Close()
		r.Env["MODELS_BASE"] = models.NewBase(db)
		next(w, r)
	}
}

func authenticate(account string, key string, r *rest.Request) bool {
	if r.Method == "POST" && r.RequestURI == "/accounts" {
		return true
	}
	b := GetBase(r)
	if bson.IsObjectIdHex(account) == false {
		fmt.Printf("Invalid account %s", account)
		return false
	}
	res, err := b.AuthenticateAccount(bson.ObjectIdHex(account), key)
	if err != nil {
		// TODO
	}
	return res
}

func authorize(account string, r *rest.Request) bool {
	// TODO check application id
	return true
}

// New creates a new instance of the Rest API.
func New(s *store.Store) (*rest.Api, error) {
	db := s.DB()
	models.NewBase(db).EnsureIndex()
	db.Session.Close()

	api := rest.NewApi()
	api.Use(rest.DefaultDevStack...)
	api.Use(&BaseMiddleware{
		Store: s,
	})
	api.Use(&AuthBasicMiddleware{
		Realm:         "Hooky",
		Authenticator: authenticate,
		Authorizator:  authorize,
	})
	router, err := rest.MakeRouter(
		// Rename
		// accounts -> services
		// applications -> applications
		rest.Post("/accounts", PostAccount),
		// rest.Get("/accounts/:account/applications", GetApplications),
		// rest.Delete("/accounts/:account/applications", DeleteApplications),
		rest.Put("/accounts/:account/applications/:application", PutApplication),
		// rest.Get("/accounts/:account/applications/:application", GetApplication),
		// rest.Delete("/accounts/:account/applications/:application", DeleteApplication),
		rest.Post("/accounts/:account/applications/:application/tasks", PutTask),
		rest.Put("/accounts/:account/applications/:application/tasks/:task", PutTask),
		rest.Get("/accounts/:account/applications/:application/tasks/:task", GetTask),
		rest.Delete("/accounts/:account/applications/:application/tasks/:task", DeleteTask),
	)
	if err != nil {
		return nil, err
	}
	api.SetApp(router)

	return api, nil
}
