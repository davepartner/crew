package httpapi

import "net/http"

func (s *Server) handleListProducts(w http.ResponseWriter, r *http.Request) {
	company := companyFrom(r)

	products, err := s.store.ListProducts(r.Context(), company.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "We couldn't load the products.")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "products": products})
}

func (s *Server) handleCreateProduct(w http.ResponseWriter, r *http.Request) {
	id := identityFrom(r)
	company := companyFrom(r)

	if company.IsProduct {
		writeError(w, http.StatusUnprocessableEntity, "Products can't contain other products.")
		return
	}

	var body struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Audience    string `json:"audience"`
		Website     string `json:"website"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	name := trim(body.Name, 80)
	if name == "" {
		writeError(w, http.StatusUnprocessableEntity, "Give the product a name.")
		return
	}

	product, err := s.store.CreateProduct(r.Context(), id.UserID, company.ID, name,
		trim(body.Description, 2000), trim(body.Audience, 200), trim(body.Website, 300))
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "We couldn't create that product.")
		return
	}

	_ = s.store.AddActivity(r.Context(), id.UserID, company.ID, "product_created",
		"You added the product "+product.Name, "", "")

	writeJSON(w, http.StatusCreated, map[string]any{"ok": true, "product": product})
}

func (s *Server) handleDeleteProduct(w http.ResponseWriter, r *http.Request) {
	id := identityFrom(r)

	company, err := s.store.GetCompany(r.Context(), id.UserID, r.PathValue("id"))
	if err != nil || !company.IsProduct {
		writeError(w, http.StatusNotFound, "That product doesn't exist.")
		return
	}

	if err := s.store.DeleteCompany(r.Context(), id.UserID, company.ID); err != nil {
		writeError(w, http.StatusNotFound, "That product doesn't exist.")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
