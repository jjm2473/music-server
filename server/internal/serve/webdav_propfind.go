package serve

import (
	"encoding/xml"
	"errors"
	"net/http"
	"os"
	"sort"
	"strconv"

	"music-server/server/internal/pathmap"
	"music-server/server/internal/security"
)

type multistatus struct {
	XMLName   xml.Name       `xml:"D:multistatus"`
	XmlnsD    string         `xml:"xmlns:D,attr"`
	Responses []propResponse `xml:"D:response"`
}

type propResponse struct {
	Href     string      `xml:"D:href"`
	Propstat propstatElt `xml:"D:propstat"`
}

type propstatElt struct {
	Prop   propElt `xml:"D:prop"`
	Status string  `xml:"D:status"`
}

type propElt struct {
	ResourceType resourceType `xml:"D:resourcetype"`
	Length       string       `xml:"D:getcontentlength,omitempty"`
	LastModified string       `xml:"D:getlastmodified,omitempty"`
	ContentType  string       `xml:"D:getcontenttype,omitempty"`
}

type resourceType struct {
	Collection *struct{} `xml:"D:collection,omitempty"`
}

func (h *Handler) handlePROPFIND(w http.ResponseWriter, r *http.Request) {
	rel, err := pathmap.URLToRelative(h.cfg.Common.Path, r.URL.Path)
	if err != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	fullPath, err := pathmap.RelativeToFS(h.cfg.Common.Root, rel)
	if err != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	safePath, err := security.ResolveSafeReadPath(h.cfg.Common.Root, fullPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	st, err := os.Stat(safePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	depth := parseDepth(r.Header.Get("Depth"))
	responses := make([]propResponse, 0, 8)
	responses = append(responses, toPropResponse(h.cfg.Common.Path, rel, st))

	if st.IsDir() && depth >= 1 {
		entries, err := os.ReadDir(safePath)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].Name() < entries[j].Name()
		})
		for _, ent := range entries {
			if security.IsHiddenBase(ent.Name()) {
				continue
			}
			childRel := splitRel(rel, ent.Name())
			childPath, err := pathmap.RelativeToFS(h.cfg.Common.Root, childRel)
			if err != nil {
				continue
			}
			safeChild, err := security.ResolveSafeReadPath(h.cfg.Common.Root, childPath)
			if err != nil {
				continue
			}
			childStat, err := os.Stat(safeChild)
			if err != nil {
				continue
			}
			responses = append(responses, toPropResponse(h.cfg.Common.Path, childRel, childStat))
		}
	}

	body, err := xml.MarshalIndent(multistatus{
		XmlnsD:    "DAV:",
		Responses: responses,
	}, "", "  ")
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.WriteHeader(207)
	_, _ = w.Write([]byte(xml.Header))
	_, _ = w.Write(body)
}

func toPropResponse(baseURL, rel string, st os.FileInfo) propResponse {
	prop := propElt{
		ResourceType: resourceType{},
		LastModified: formatHTTPDate(st.ModTime()),
		ContentType:  guessContentType(st.Name(), st.IsDir()),
	}
	if st.IsDir() {
		prop.ResourceType.Collection = &struct{}{}
	} else {
		prop.Length = int64ToString(st.Size())
	}
	return propResponse{
		Href: cleanHref(baseURL, rel, st.IsDir()),
		Propstat: propstatElt{
			Prop:   prop,
			Status: "HTTP/1.1 200 OK",
		},
	}
}

func int64ToString(v int64) string {
	return strconv.FormatInt(v, 10)
}
