package handler

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/rs/zerolog/hlog"
)

// NOTICE:
// Boldly taken from https://github.com/gorilla/mux#serving-single-page-applications.
// Adjusted to perform some tasks not for every request

// spaHandler implements the http.Handler interface, so we can use it
// to respond to HTTP requests. The path to the static directory and
// path to the index file within that static directory are used to
// serve the SPA in the given static directory.
type spaHandler struct {
	staticPath string
	indexPath  string
	fileServer http.Handler
}

func NewSpaHandler(staticPath string, indexPath string) *spaHandler {
	return &spaHandler{
		staticPath: staticPath,
		indexPath:  filepath.Join(staticPath, indexPath),
		fileServer: http.FileServer(http.Dir(staticPath)),
	}
}

func (h *spaHandler) SetFileServer(handler http.Handler) *spaHandler {
	h.fileServer = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		handler.ServeHTTP(w, r)
	})

	return h
}

// ServeHTTP inspects the URL path to locate a file within the static dir
// on the SPA handler. If a file is found, it will be served. If not, the
// file located at the index path on the SPA handler will be served. This
// is suitable behavior for serving an SPA (single page application).
func (h *spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// get the absolute path to prevent directory traversal
	path, err := filepath.Abs(r.URL.Path)
	if err != nil {
		// if we fail to get the absolute path, respond with a 400 bad request
		// and stop
		http.Error(w, err.Error(), http.StatusBadRequest)

		return
	}

	// prepend the path with the path to the static directory
	path = filepath.Join(h.staticPath, path)

	// check whether a file (and only a file, not a directory or a link to a file) exists at the given path
	stat, err := os.Lstat(path)
	if os.IsNotExist(err) || (err == nil && !stat.Mode().IsRegular()) || path == h.indexPath {
		// file does not exist, serve index page
		hlog.FromRequest(r).Debug().Str("indexPath", h.indexPath).Msg("Trying to serve default page.")

		http.ServeFile(w, r, h.indexPath)

		return
	} else if err != nil {
		// if we got an error (that wasn't that the file doesn't exist) stating the
		// file, return a 500 internal server error and stop
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	hlog.FromRequest(r).Debug().Msgf("Trying to serve: '%s'", path)

	// Otherwise, use http.FileServer to serve the static dir
	// In this case we can set a max-age of 1 year as the file names contain cache busters.
	// But, as the index file does not contain one, we cannot allow it to be cached for that long.
	w.Header().Set("Cache-Control", "max-age=31536000")
	h.fileServer.ServeHTTP(w, r)
}
