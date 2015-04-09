package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	repo "github.com/docker/go-tuf"
	"github.com/docker/go-tuf/data"
	"github.com/docker/go-tuf/store"
	"github.com/docker/go-tuf/util"
	"github.com/docker/vetinari/errors"
	"github.com/docker/vetinari/utils"
	"github.com/gorilla/mux"
)

var db = util.GetSqliteDB()

func MainHandler(ctx utils.IContext, w http.ResponseWriter, r *http.Request) *errors.HTTPError {
	if r.Method == "GET" {
		err := json.NewEncoder(w).Encode("{}")
		if err != nil {
			w.Write([]byte("{server_error: 'Could not parse error message'}"))
		}
	} else {
		w.WriteHeader(http.StatusNotFound)
		return &errors.HTTPError{http.StatusNotFound, 9999, nil}
	}
	return nil
}

// AddHandler accepts urls in the form /<imagename>/<tag>
func AddHandler(ctx utils.IContext, w http.ResponseWriter, r *http.Request) *errors.HTTPError {
	log.Printf("AddHandler")
	vars := mux.Vars(r)
	local := store.DBStore(db, vars["imageName"])
	// parse body for correctness
	meta := data.FileMeta{}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&meta)
	defer r.Body.Close()
	if err != nil {
		return &errors.HTTPError{http.StatusInternalServerError, 9999, err}
	}
	// add to targets
	local.AddBlob(vars["tag"], meta)
	tufRepo, err := repo.NewRepo(local, "sha256", "sha512")
	if err != nil {
		return &errors.HTTPError{http.StatusInternalServerError, 9999, err}
	}
	_ = tufRepo.Init(true)
	err = tufRepo.AddTarget(vars["tag"], json.RawMessage{})
	if err != nil {
		return &errors.HTTPError{http.StatusInternalServerError, 9999, err}
	}
	err = tufRepo.Sign("targets.json")
	if err != nil {
		return &errors.HTTPError{http.StatusInternalServerError, 9999, err}
	}
	tufRepo.Snapshot(repo.CompressionTypeNone)
	err = tufRepo.Sign("snapshot.json")
	if err != nil {
		return &errors.HTTPError{http.StatusInternalServerError, 9999, err}
	}
	tufRepo.Timestamp()
	err = tufRepo.Sign("timestamp.json")
	if err != nil {
		return &errors.HTTPError{http.StatusInternalServerError, 9999, err}
	}
	return nil
}

// RemoveHandler accepts urls in the form /<imagename>/<tag>
func RemoveHandler(ctx utils.IContext, w http.ResponseWriter, r *http.Request) *errors.HTTPError {
	log.Printf("RemoveHandler")
	// remove tag from tagets list
	vars := mux.Vars(r)
	local := store.DBStore(db, vars["imageName"])
	local.RemoveBlob(vars["tag"])
	tufRepo, err := repo.NewRepo(local, "sha256", "sha512")
	if err != nil {
		return &errors.HTTPError{http.StatusInternalServerError, 9999, err}
	}
	_ = tufRepo.Init(true)
	tufRepo.RemoveTarget(vars["tag"])
	err = tufRepo.Sign("targets.json")
	if err != nil {
		return &errors.HTTPError{http.StatusInternalServerError, 9999, err}
	}
	tufRepo.Snapshot(repo.CompressionTypeNone)
	err = tufRepo.Sign("snapshot.json")
	if err != nil {
		return &errors.HTTPError{http.StatusInternalServerError, 9999, err}
	}
	tufRepo.Timestamp()
	err = tufRepo.Sign("timestamp.json")
	if err != nil {
		return &errors.HTTPError{http.StatusInternalServerError, 9999, err}
	}
	return nil
}

// GetHandler accepts urls in the form /<imagename>/<tuf file>.json
func GetHandler(ctx utils.IContext, w http.ResponseWriter, r *http.Request) *errors.HTTPError {
	log.Printf("GetHandler")
	// generate requested file and serve
	vars := mux.Vars(r)
	local := store.DBStore(db, vars["imageName"])

	meta, err := local.GetMeta()
	if err != nil {
		return &errors.HTTPError{http.StatusInternalServerError, 9999, err}
	}
	w.Write(meta[vars["tufFile"]])
	return nil
}

func GenKeysHandler(ctx utils.IContext, w http.ResponseWriter, r *http.Request) *errors.HTTPError {
	log.Printf("GenKeysHandler")
	// remove tag from tagets list
	vars := mux.Vars(r)
	local := store.DBStore(db, vars["imageName"])
	tufRepo, err := repo.NewRepo(local, "sha256", "sha512")
	if err != nil {
		return &errors.HTTPError{http.StatusInternalServerError, 9999, err}
	}
	tufRepo.GenKey("root")
	tufRepo.GenKey("targets")
	tufRepo.GenKey("snapshot")
	tufRepo.GenKey("timestamp")
	return nil
}
