package memory

import (
	"path/filepath"
	"sort"
	"strings"
)

// GraphTopologyNode is one note in the memory vault, or one wikilink target
// that no note answers to yet.
type GraphTopologyNode struct {
	ID         string   `json:"id"`
	Title      string   `json:"title"`
	Category   string   `json:"category,omitempty"`
	Tier       string   `json:"tier,omitempty"`
	Path       string   `json:"path,omitempty"`
	Tags       []string `json:"tags,omitempty"`
	Importance float64  `json:"importance"`
	Degree     int      `json:"degree"`
	// Resolved is false for a link target with no note behind it — Obsidian's
	// "unresolved link". Keeping them visible is the point: they show where the
	// vault refers to something it has not written down.
	Resolved bool `json:"resolved"`
}

// GraphLinkRef is one written wikilink on a note-to-note edge. Heading, block,
// display text, and embed do not create extra edges.
type GraphLinkRef struct {
	Heading string `json:"heading,omitempty"`
	Block   string `json:"block,omitempty"`
	Display string `json:"display,omitempty"`
	Embed   bool   `json:"embed,omitempty"`
}

// GraphTopologyEdge is a wikilink from one note to another. Refs records how
// that link was written. Old clients can ignore it; the edge is still one
// note-to-note link.
type GraphTopologyEdge struct {
	Source string         `json:"source"`
	Target string         `json:"target"`
	Weight int            `json:"weight"`
	Refs   []GraphLinkRef `json:"refs,omitempty"`
}

// GraphTopology is a read-only snapshot of the wikilink graph.
type GraphTopology struct {
	Nodes         []GraphTopologyNode `json:"nodes"`
	Edges         []GraphTopologyEdge `json:"edges"`
	TotalNotes    int                 `json:"total_notes"`
	TotalEdges    int                 `json:"total_edges"`
	IsolatedCount int                 `json:"isolated_count"`
	Unresolved    int                 `json:"unresolved"`
	Truncated     bool                `json:"truncated"`
	Categories    []string            `json:"categories,omitempty"`
}

// GraphTopologyOptions controls what a snapshot includes.
type GraphTopologyOptions struct {
	// Limit caps the node count, keeping the most connected nodes. Zero means
	// no cap.
	Limit int
	// IncludeIsolated keeps notes that neither link out nor are linked to.
	// They dominate the node count in a typical vault while carrying no
	// topology, so they are excluded by default.
	IncludeIsolated bool
}

// GraphTopology returns the note graph derived from Obsidian wikilinks. The
// markdown notes remain the source of truth; this is a derived view for
// display, computed from the same link normalization the recall path uses.
func (s *Store) GraphTopology(opts GraphTopologyOptions) GraphTopology {
	if s == nil {
		return GraphTopology{Nodes: []GraphTopologyNode{}, Edges: []GraphTopologyEdge{}}
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	// name key -> entry ID, rebuilt here rather than read off s.graph so a
	// stale index can never produce edges that disagree with the notes.
	nameToID := make(map[string]string, len(s.entries)*2)
	for id, entry := range s.entries {
		for _, alias := range graphAliasesForEntry(entry) {
			key := graphKey(alias)
			if key == "" {
				continue
			}
			if _, taken := nameToID[key]; !taken {
				nameToID[key] = id
			}
		}
	}

	nodes := make(map[string]*GraphTopologyNode, len(s.entries))
	for id, entry := range s.entries {
		nodes[id] = &GraphTopologyNode{
			ID:         id,
			Title:      entryDisplayTitle(entry),
			Category:   entry.Category,
			Tier:       entry.Tier.String(),
			Path:       entry.Path,
			Tags:       append([]string(nil), entry.Tags...),
			Importance: entry.Importance,
			Resolved:   true,
		}
	}

	weights := make(map[string]int)
	refs := make(map[string][]GraphLinkRef)
	order := make([]string, 0, len(s.entries))
	unresolved := 0

	for id, entry := range s.entries {
		for _, written := range noteWikiLinks(entry) {
			targetID, unresolvedTitle, ok := resolveGraphLink(written.Target, nameToID)
			if !ok {
				continue
			}
			if targetID == "" {
				// Surface the dangling link as its own node instead of
				// dropping the edge and hiding the gap. Heading and block
				// stay on the edge; they are not extra nodes.
				targetID = "unresolved:" + graphKey(written.Target)
				if _, seen := nodes[targetID]; !seen {
					nodes[targetID] = &GraphTopologyNode{
						ID:       targetID,
						Title:    unresolvedTitle,
						Resolved: false,
					}
					unresolved++
				}
			}
			if targetID == id {
				continue
			}
			edgeKey := id + "\x00" + targetID
			if _, seen := weights[edgeKey]; !seen {
				order = append(order, edgeKey)
			}
			weights[edgeKey]++
			refs[edgeKey] = append(refs[edgeKey], written.Ref)
		}
	}

	for _, edgeKey := range order {
		parts := strings.SplitN(edgeKey, "\x00", 2)
		nodes[parts[0]].Degree++
		nodes[parts[1]].Degree++
	}

	totalNotes := len(s.entries)
	isolated := 0
	kept := make([]GraphTopologyNode, 0, len(nodes))
	for _, node := range nodes {
		if node.Degree == 0 {
			isolated++
			if !opts.IncludeIsolated {
				continue
			}
		}
		kept = append(kept, *node)
	}

	// Most connected first, so a cap keeps the informative core; ties break on
	// title to keep the response stable between calls.
	sort.Slice(kept, func(i, j int) bool {
		if kept[i].Degree != kept[j].Degree {
			return kept[i].Degree > kept[j].Degree
		}
		return kept[i].Title < kept[j].Title
	})

	truncated := false
	if opts.Limit > 0 && len(kept) > opts.Limit {
		kept = kept[:opts.Limit]
		truncated = true
	}

	visible := make(map[string]struct{}, len(kept))
	for _, node := range kept {
		visible[node.ID] = struct{}{}
	}

	edges := make([]GraphTopologyEdge, 0, len(order))
	for _, edgeKey := range order {
		parts := strings.SplitN(edgeKey, "\x00", 2)
		if _, ok := visible[parts[0]]; !ok {
			continue
		}
		if _, ok := visible[parts[1]]; !ok {
			continue
		}
		edges = append(edges, GraphTopologyEdge{
			Source: parts[0],
			Target: parts[1],
			Weight: weights[edgeKey],
			Refs:   graphLinkRefs(refs[edgeKey]),
		})
	}

	categorySet := make(map[string]struct{})
	for _, node := range kept {
		if node.Category != "" {
			categorySet[node.Category] = struct{}{}
		}
	}
	categories := make([]string, 0, len(categorySet))
	for category := range categorySet {
		categories = append(categories, category)
	}
	sort.Strings(categories)

	return GraphTopology{
		Nodes:         kept,
		Edges:         edges,
		TotalNotes:    totalNotes,
		TotalEdges:    len(order),
		IsolatedCount: isolated,
		Unresolved:    unresolved,
		Truncated:     truncated,
		Categories:    categories,
	}
}

// writtenWikiLink is a wikilink reduced to the note name the graph resolves,
// plus the heading, block, display text, and embed that stay on that edge.
type writtenWikiLink struct {
	Target string
	Ref    GraphLinkRef
}

func noteWikiLinks(entry *Entry) []writtenWikiLink {
	if entry == nil {
		return nil
	}
	out := make([]writtenWikiLink, 0, len(entry.Links))
	seen := make(map[string]bool)
	add := func(target string, ref GraphLinkRef) {
		target = strings.TrimSpace(target)
		if target == "" {
			return
		}
		key := strings.ToLower(target) + "\x00" + strings.ToLower(ref.Heading) + "\x00" + strings.ToLower(ref.Block) + "\x00" + ref.Display + "\x00"
		if ref.Embed {
			key += "embed"
		}
		if seen[key] {
			return
		}
		seen[key] = true
		out = append(out, writtenWikiLink{Target: target, Ref: ref})
	}
	for _, link := range entry.Links {
		add(link, GraphLinkRef{})
	}
	for _, link := range parseWikiLinks(entry.Content) {
		add(link.Target, GraphLinkRef{
			Heading: link.Heading,
			Block:   link.Block,
			Display: link.Display,
			Embed:   link.Embed,
		})
	}
	return out
}

// resolveGraphLink returns the note ID a link names. An unresolved note name
// returns ok with an empty ID so the caller can keep the dangling node.
func resolveGraphLink(link string, nameToID map[string]string) (id, title string, ok bool) {
	link = strings.TrimSpace(link)
	if link == "" {
		return "", "", false
	}
	for _, key := range graphKeysForLink(link) {
		if resolved, found := nameToID[key]; found {
			return resolved, "", true
		}
	}
	if graphKey(link) == "" {
		return "", "", false
	}
	return "", link, true
}

func graphLinkRefs(refs []GraphLinkRef) []GraphLinkRef {
	kept := make([]GraphLinkRef, 0, len(refs))
	for _, ref := range refs {
		if ref.Heading == "" && ref.Block == "" && ref.Display == "" && !ref.Embed {
			continue
		}
		kept = append(kept, ref)
	}
	if len(kept) == 0 {
		return nil
	}
	return kept
}

// entryDisplayTitle prefers the note filename, which is what wikilinks point
// at, and falls back to the first line of the body.
func entryDisplayTitle(e *Entry) string {
	if e == nil {
		return ""
	}
	if path := strings.TrimSpace(e.Path); path != "" {
		base := strings.TrimSuffix(filepath.Base(filepath.ToSlash(path)), ".md")
		if base != "" {
			return base
		}
	}
	for _, line := range strings.Split(e.Content, "\n") {
		line = strings.TrimSpace(strings.TrimLeft(line, "# "))
		if line == "" {
			continue
		}
		if len([]rune(line)) > 60 {
			return string([]rune(line)[:60]) + "…"
		}
		return line
	}
	return e.ID
}
