package main

import (
	"log"

	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/transport/http"
	"github.com/go-git/go-git/v6/storage/memory"
)

func getTemplateFileFromForge(req woodpeckerRequest, extraCABundle []byte) ([]byte, bool) {
	repo, err := git.Clone(memory.NewStorage(), nil, &git.CloneOptions{
		URL: req.Repo.Clone,
		Auth: &http.BasicAuth{
			Username: req.Netrc.Login,
			Password: req.Netrc.Password,
		},
		NoCheckout: true,
		CABundle: extraCABundle,
	})
	if err != nil {
		log.Printf("Error opening repo: '%v'", err)
		return nil, false
	}

	commit, err := repo.CommitObject(plumbing.NewHash(req.Pipeline.Commit))
	if err != nil {
		log.Printf("Error getting commit: '%v'", err)
		return nil, false
	}

	tree, err := commit.Tree()
	if err != nil {
		log.Printf("Error getting tree: '%v'", err)
	}

	dirEntry, err := tree.FindEntry(".woodpecker")
	if err != nil {
		// No log entry, not finding the file is an expected case.
		return nil, false
	}

	dirTree, err := repo.TreeObject(dirEntry.Hash)
	if err != nil {
		log.Printf("Error getting tree object: '%v'", err)
		return nil, false
	}

	file, err := dirTree.FindEntry("woodpecker-template.yaml")
	if err != nil {
		// No log entry, not finding the file is an expected case.
		return nil, false
	}

	blob, err := repo.BlobObject(file.Hash)
	if err != nil {
		log.Printf("Error getting blob object: '%v'", err)
		return nil, false
	}

	reader, err := blob.Reader()
	if err != nil {
		log.Printf("Error getting blob reader: '%v'", err)
		return nil, false
	}

	data := make([]byte, blob.Size)

	n, err := reader.Read(data)
	if err != nil {
		log.Printf("Error reading blob: '%v'", err)
		return nil, false
	}

	if int64(n) != blob.Size {
		log.Printf("Error reading blob, incorrect size: %d", n)
		return nil, false
	}

	return data, true
}
