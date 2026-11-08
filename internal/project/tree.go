package project

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

// TreeNode is one entry of the project tree: a folder, a text file, or a
// file a viewer shows.
type TreeNode struct {
	Name     string
	Rel      string
	Dir      bool
	Note     *Note
	Asset    *Asset
	Children []*TreeNode
}

// BuildTree caches the project as a tree, so every frame reads what a
// refresh left instead of the disk.
func (s *Store) BuildTree() {
	if s.IsMemory() {
		root := &TreeNode{Name: "Mojian", Dir: true}
		for _, n := range s.Sorted() {
			root.Children = append(root.Children, &TreeNode{
				Name: filepath.Base(n.Rel),
				Rel:  n.Rel,
				Note: n,
			})
		}
		sortNodes(root.Children)
		s.Tree = root
		return
	}
	s.Tree = s.treeFromEntries()
}

// treeFromEntries builds the tree from the last listing, so no backend
// walks the disk twice.
func (s *Store) treeFromEntries() *TreeNode {
	root := &TreeNode{Name: filepath.Base(s.Dir), Dir: true}
	nodes := map[string]*TreeNode{"": root}
	for _, e := range s.Entries {
		rel := filepath.FromSlash(e.Rel)
		parent := filepath.Dir(rel)
		if parent == "." {
			parent = ""
		}
		p := nodes[parent]
		if p == nil {
			continue
		}
		n := &TreeNode{Name: filepath.Base(rel), Rel: rel, Dir: e.Dir}
		if e.Dir {
			nodes[rel] = n
		} else if note := s.ByRel[rel]; note != nil {
			n.Note = note
		} else if asset := s.Assets[rel]; asset != nil {
			n.Asset = asset
		} else {
			continue
		}
		p.Children = append(p.Children, n)
	}
	var sortRec func(*TreeNode)
	sortRec = func(n *TreeNode) {
		var folders, files []*TreeNode
		for _, c := range n.Children {
			if c.Dir {
				folders = append(folders, c)
			} else {
				files = append(files, c)
			}
		}
		sortNodes(folders)
		sortNodes(files)
		n.Children = append(folders, files...)
		for _, c := range n.Children {
			sortRec(c)
		}
	}
	sortRec(root)
	return root
}

func sortNodes(ns []*TreeNode) {
	sort.SliceStable(ns, func(i, j int) bool {
		a, b := strings.ToLower(ns[i].Name), strings.ToLower(ns[j].Name)
		if a == b {
			return ns[i].Name < ns[j].Name
		}
		return a < b
	})
}

// NewFolder makes a folder in parent and returns its path within the
// project. An existing name takes the next number.
func (s *Store) NewFolder(parent, name string) (string, error) {
	name = folderName(name)
	for i := 1; i < 1000; i++ {
		candidate := name
		if i > 1 {
			candidate = fmt.Sprintf("%s %d", name, i)
		}
		rel := filepath.Join(parent, candidate)
		if s.ExistsRel(rel) {
			continue
		}
		if err := s.FS.Mkdir(rel); err != nil {
			return "", err
		}
		if err := s.Refresh(); err != nil {
			return rel, err
		}
		return rel, nil
	}
	return "", fmt.Errorf("too many folders named %q", name)
}

// folderName keeps a typed name a single folder name.
func folderName(name string) string { return fileName(name, "New Folder") }
