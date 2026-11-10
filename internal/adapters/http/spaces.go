package httpadapter

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/code-corhuila/drp-space-api/internal/app"
	"github.com/code-corhuila/drp-space-api/internal/domain"
)

var canonicalUUID = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

type spaceDTO struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Kind        string `json:"kind"`
	Capacity    int    `json:"capacity"`
	Available   bool   `json:"available"`
	Description string `json:"description"`
}

type listEnvelope struct {
	Data []spaceDTO `json:"data"`
	Meta metaDTO    `json:"meta"`
}

type metaDTO struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"totalPages"`
}

func toSpaceDTO(s domain.Space) spaceDTO {
	return spaceDTO{
		ID:          s.ID,
		Name:        s.Name,
		Kind:        string(s.Kind),
		Capacity:    s.Capacity,
		Available:   s.Active,
		Description: s.Description,
	}
}

type createSpaceBody struct {
	Name        string `json:"name"`
	Kind        string `json:"kind"`
	Description string `json:"description"`
	Capacity    int    `json:"capacity"`
}

func CreateSpace(cat app.Catalog) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			writeErr(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Datos de entrada inválidos", nil)
			return
		}
		var body createSpaceBody
		if err := json.Unmarshal(raw, &body); err != nil {
			writeErr(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "JSON malformado", nil)
			return
		}
		got, cerr := cat.Create(r.Context(), app.CreateSpaceCommand{
			Actor:       principalFrom(r),
			Name:        body.Name,
			Kind:        body.Kind,
			Description: body.Description,
			Capacity:    body.Capacity,
		})
		if errors.Is(cerr, app.ErrForbidden) {
			writeErr(w, r, http.StatusForbidden, "FORBIDDEN", "No tienes permiso para esta operación", nil)
			return
		}
		if errors.Is(cerr, app.ErrConflict) {
			writeErr(w, r, http.StatusUnprocessableEntity, "BUSINESS_RULE_VIOLATION", "Ya existe un espacio con ese nombre", []map[string]string{{"field": "name", "message": "Debe ser único sin distinguir mayúsculas"}})
			return
		}
		if errors.Is(cerr, app.ErrValidation) {
			writeErr(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Datos de entrada inválidos", nil)
			return
		}
		if cerr != nil {
			writeErr(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Error interno", nil)
			return
		}
		writeJSON(w, http.StatusCreated, toSpaceDTO(got))
	}
}

type blockDTO struct {
	ID      string `json:"id"`
	SpaceID string `json:"spaceId"`
	StartAt string `json:"startAt"`
	EndAt   string `json:"endAt"`
	Reason  string `json:"reason,omitempty"`
}

type blockListEnvelope struct {
	Data []blockDTO `json:"data"`
	Meta metaDTO    `json:"meta"`
}

func toBlockDTO(b domain.BlockedPeriod) blockDTO {
	return blockDTO{
		ID:      b.ID,
		SpaceID: b.SpaceID,
		StartAt: b.Period.Start.UTC().Format(time.RFC3339),
		EndAt:   b.Period.End.UTC().Format(time.RFC3339),
		Reason:  b.Reason,
	}
}

func ListBlocks(cat app.Catalog) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("spaceId")
		if !canonicalUUID.MatchString(id) {
			writeErr(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "El identificador no es un UUID canónico", []map[string]string{{"field": "spaceId", "message": "Debe ser un UUID en minúsculas con guiones"}})
			return
		}
		page, limit, err := parsePageLimit(r)
		if err != nil {
			writeErr(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), err.details)
			return
		}
		got, lerr := cat.ListBlocks(r.Context(), principalFrom(r), id, page, limit)
		if errors.Is(lerr, app.ErrForbidden) {
			writeErr(w, r, http.StatusForbidden, "FORBIDDEN", "No tienes permiso para esta operación", nil)
			return
		}
		if errors.Is(lerr, app.ErrNotFound) {
			writeErr(w, r, http.StatusNotFound, "NOT_FOUND", "Espacio no encontrado", nil)
			return
		}
		if errors.Is(lerr, app.ErrValidation) {
			writeErr(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Parámetros de paginación inválidos", nil)
			return
		}
		if lerr != nil {
			writeErr(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Error interno", nil)
			return
		}
		data := make([]blockDTO, 0, len(got.Items))
		for _, b := range got.Items {
			data = append(data, toBlockDTO(b))
		}
		writeJSON(w, http.StatusOK, blockListEnvelope{
			Data: data,
			Meta: metaDTO{Page: got.Page, Limit: got.Limit, Total: got.Total, TotalPages: got.TotalPages},
		})
	}
}

type createBlockBody struct {
	StartAt string `json:"startAt"`
	EndAt   string `json:"endAt"`
	Reason  string `json:"reason"`
}

func CreateBlock(cat app.Catalog) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("spaceId")
		if !canonicalUUID.MatchString(id) {
			writeErr(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "El identificador no es un UUID canónico", []map[string]string{{"field": "spaceId", "message": "Debe ser un UUID en minúsculas con guiones"}})
			return
		}
		raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			writeErr(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Datos de entrada inválidos", nil)
			return
		}
		var body createBlockBody
		if err := json.Unmarshal(raw, &body); err != nil {
			writeErr(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "JSON malformado", nil)
			return
		}
		start, berr := parseInstant("startAt", body.StartAt)
		if berr != nil {
			writeErr(w, r, http.StatusBadRequest, "VALIDATION_ERROR", berr.Error(), berr.details)
			return
		}
		end, berr := parseInstant("endAt", body.EndAt)
		if berr != nil {
			writeErr(w, r, http.StatusBadRequest, "VALIDATION_ERROR", berr.Error(), berr.details)
			return
		}
		if !end.After(start) {
			writeErr(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "endAt debe ser posterior a startAt", []map[string]string{{"field": "endAt", "message": "Debe ser posterior a startAt"}})
			return
		}
		got, cerr := cat.BlockPeriod(r.Context(), app.BlockPeriodCommand{
			Actor:   principalFrom(r),
			SpaceID: id,
			StartAt: start,
			EndAt:   end,
			Reason:  body.Reason,
		})
		if errors.Is(cerr, app.ErrForbidden) {
			writeErr(w, r, http.StatusForbidden, "FORBIDDEN", "No tienes permiso para esta operación", nil)
			return
		}
		if errors.Is(cerr, app.ErrNotFound) {
			writeErr(w, r, http.StatusNotFound, "NOT_FOUND", "Espacio no encontrado", nil)
			return
		}
		if errors.Is(cerr, app.ErrValidation) {
			writeErr(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Datos de entrada inválidos", nil)
			return
		}
		if cerr != nil {
			writeErr(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Error interno", nil)
			return
		}
		writeJSON(w, http.StatusCreated, toBlockDTO(got))
	}
}

func parseInstant(field, raw string) (time.Time, *bindError) {
	if !strings.HasSuffix(raw, "Z") {
		return time.Time{}, &bindError{msg: field + " debe ser RFC 3339 UTC con Z", details: []map[string]string{{"field": field, "message": "Debe terminar en Z"}}}
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}, &bindError{msg: field + " no es RFC 3339", details: []map[string]string{{"field": field, "message": "Debe ser RFC 3339 en UTC"}}}
	}
	return t.UTC(), nil
}

func ListSpaces(cat app.Catalog) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page, limit, err := parsePageLimit(r)
		if err != nil {
			writeErr(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), err.details)
			return
		}
		filter := app.ListFilter{Page: page, Limit: limit}
		if raw := strings.TrimSpace(r.URL.Query().Get("kind")); raw != "" {
			k, perr := domain.ParseKind(raw)
			if perr != nil {
				writeErr(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "El campo kind no es válido", []map[string]string{{"field": "kind", "message": "Debe ser un tipo de espacio conocido"}})
				return
			}
			filter.Kind = &k
		}
		if raw, ok := r.URL.Query()["available"]; ok {
			if len(raw) == 0 || (raw[0] != "true" && raw[0] != "false") {
				writeErr(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "El campo available no es válido", []map[string]string{{"field": "available", "message": "Debe ser true o false"}})
				return
			}
			v := raw[0] == "true"
			filter.Available = &v
		}
		got, lerr := cat.List(r.Context(), filter)
		if errors.Is(lerr, app.ErrValidation) {
			writeErr(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Parámetros de paginación inválidos", nil)
			return
		}
		if lerr != nil {
			writeErr(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Error interno", nil)
			return
		}
		data := make([]spaceDTO, 0, len(got.Items))
		for _, s := range got.Items {
			data = append(data, toSpaceDTO(s))
		}
		writeJSON(w, http.StatusOK, listEnvelope{
			Data: data,
			Meta: metaDTO{Page: got.Page, Limit: got.Limit, Total: got.Total, TotalPages: got.TotalPages},
		})
	}
}

func GetSpace(cat app.Catalog) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("spaceId")
		if !canonicalUUID.MatchString(id) {
			writeErr(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "El identificador no es un UUID canónico", []map[string]string{{"field": "spaceId", "message": "Debe ser un UUID en minúsculas con guiones"}})
			return
		}
		s, err := cat.Get(r.Context(), id)
		if errors.Is(err, app.ErrNotFound) {
			writeErr(w, r, http.StatusNotFound, "NOT_FOUND", "Espacio no encontrado", nil)
			return
		}
		if err != nil {
			writeErr(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Error interno", nil)
			return
		}
		writeJSON(w, http.StatusOK, toSpaceDTO(s))
	}
}

type bindError struct {
	msg     string
	details []map[string]string
}

func (e bindError) Error() string { return e.msg }

func parsePageLimit(r *http.Request) (int, int, *bindError) {
	page := 1
	limit := 20
	if raw := strings.TrimSpace(r.URL.Query().Get("page")); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil {
			return 0, 0, &bindError{msg: "El parámetro page debe ser un entero", details: []map[string]string{{"field": "page", "message": "Debe ser un entero"}}}
		}
		if n < 1 {
			return 0, 0, &bindError{msg: "El parámetro page debe ser mayor o igual a 1", details: []map[string]string{{"field": "page", "message": "Debe ser mayor o igual a 1"}}}
		}
		page = n
	}
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil {
			return 0, 0, &bindError{msg: "El parámetro limit debe ser un entero", details: []map[string]string{{"field": "limit", "message": "Debe ser un entero"}}}
		}
		if n < 1 || n > 100 {
			return 0, 0, &bindError{msg: "El parámetro limit debe estar entre 1 y 100", details: []map[string]string{{"field": "limit", "message": "Debe estar entre 1 y 100"}}}
		}
		limit = n
	}
	return page, limit, nil
}
