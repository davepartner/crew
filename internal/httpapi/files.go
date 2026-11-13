package httpapi

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/davepartner/crew/internal/extract"
	"github.com/davepartner/crew/internal/store"
)

var unsafeFileChars = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

func (s *Server) handleListFiles(w http.ResponseWriter, r *http.Request) {
	company := companyFrom(r)

	files, err := s.store.ListCompanyFiles(r.Context(), company.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "We couldn't load the files.")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "files": files})
}

func (s *Server) handleUploadFile(w http.ResponseWriter, r *http.Request) {
	id := identityFrom(r)
	company := companyFrom(r)

	if count, _ := s.store.CountCompanyFiles(r.Context(), company.ID); count >= 20 {
		writeError(w, http.StatusUnprocessableEntity, "This company already has 20 files. Remove one first.")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 12<<20)
	if err := r.ParseMultipartForm(12 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "That file is too large or the upload was interrupted.")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Choose a file to upload.")
		return
	}
	defer file.Close()

	if header.Size > 10<<20 {
		writeError(w, http.StatusUnprocessableEntity, "Files must be under 10 MB.")
		return
	}

	data, err := io.ReadAll(io.LimitReader(file, 10<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, "We couldn't read that file.")
		return
	}

	mime := http.DetectContentType(data)

	name := unsafeFileChars.ReplaceAllString(filepath.Base(header.Filename), "-")
	if name == "" || name == "." {
		name = "upload"
	}
	ext := filepath.Ext(name)
	if len(ext) > 12 {
		ext = ""
	}

	fileID := store.NewID()
	directory := filepath.Join(filepath.Dir(s.cfg.DBPath), "uploads", company.ID)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		writeError(w, http.StatusInternalServerError, "We couldn't store that file.")
		return
	}

	path := filepath.Join(directory, fileID+ext)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		writeError(w, http.StatusInternalServerError, "We couldn't store that file.")
		return
	}

	extractCtx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
	text := extract.Text(extractCtx, path, mime, data)
	cancel()

	_ = s.store.DeleteCompanyAdvice(r.Context(), company.ID)
	saved, err := s.store.CreateCompanyFile(r.Context(), id.UserID, company.ID, name, mime, path, text, int64(len(data)))
	if err != nil {
		_ = os.Remove(path)
		writeError(w, http.StatusInternalServerError, "We couldn't save that file.")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{"ok": true, "file": saved})
}

func (s *Server) handleDeleteFile(w http.ResponseWriter, r *http.Request) {
	company := companyFrom(r)

	file, err := s.store.GetCompanyFile(r.Context(), company.ID, r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusNotFound, "That file doesn't exist.")
		return
	}

	if err := s.store.DeleteCompanyFile(r.Context(), company.ID, file.ID); err != nil {
		writeError(w, http.StatusNotFound, "That file doesn't exist.")
		return
	}
	_ = os.Remove(file.Path)
	_ = s.store.DeleteCompanyAdvice(r.Context(), company.ID)

	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleDownloadFile(w http.ResponseWriter, r *http.Request) {
	company := companyFrom(r)

	file, err := s.store.GetCompanyFile(r.Context(), company.ID, r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusNotFound, "That file doesn't exist.")
		return
	}

	w.Header().Set("Content-Disposition", `attachment; filename="`+file.Name+`"`)
	http.ServeFile(w, r, file.Path)
}
