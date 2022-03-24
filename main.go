package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	http2 "github.com/go-git/go-git/v5/plumbing/transport/http"
	_ "github.com/joho/godotenv/autoload"
	"github.com/rs/zerolog/log"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	time "time"
)

var (
	path                         = "/tmp/repo"
	fullFilename                 = ""
	filename                     = ""
	branchname                   = ""
	organization                 = ""
	repoName                     = ""
	prTarget                     = ""
	file         *os.File        = nil
	repo         *git.Repository = nil
	worktree     *git.Worktree   = nil
)

func main() {
	initialize()
	log.Info().Msg("cloning repo")
	clone()
	log.Info().Msg("create branch")
	branch()
	log.Info().Msg("add new agenda")
	addFile()
	log.Info().Msg("commit agenda")
	commit()
	log.Info().Msg("push new branch")
	push()
	log.Info().Msg("create pr")
	pr()
}

func initialize() {
	today := time.Now()
	for today.Weekday() != time.Wednesday {
		today = today.Add(time.Hour * 24)
	}
	filename = fmt.Sprintf("%02d-%02d.md", today.Month(), today.Day())
	branchname = fmt.Sprintf("%02d-%02d", today.Month(), today.Day())
	fullFilename = filepath.Join(path, filename)

	organization = os.Getenv("GIT_ORGANIZATION")
	repoName = os.Getenv("GIT_REPO")
	prTarget = os.Getenv("GIT_HEAD")
}

func clone() {
	var err error
	username := os.Getenv("GIT_USERNAME")
	token := os.Getenv("GIT_TOKEN")
	auth := &http2.BasicAuth{Username: username, Password: token}
	repo, err = git.PlainClone(path, false, &git.CloneOptions{
		URL:      fmt.Sprintf("https://github.com/%s/%s.git", organization, repoName),
		Progress: os.Stdout,
		Auth:     auth,
	})
	if err != nil {
		log.Fatal().Err(err).Msgf("error cloning repo")
	}
}

func addFile() {
	var err error
	file, err = os.Create(fullFilename)
	if err != nil {
		log.Fatal().Err(err).Msgf("could not create file %s", fullFilename)
	}

	writer := bufio.NewWriter(file)
	_, err = writer.WriteString("# General\n\n# Skyblock")
	if err != nil {
		log.Fatal().Err(err).Msgf("error writing to file")
	}
	err = writer.Flush()
	if err != nil {
		log.Fatal().Err(err).Msgf("error writing to file")
	}
}

func branch() {
	var err error
	worktree, err = repo.Worktree()
	if err != nil {
		log.Fatal().Err(err).Msg("could not get worktree from repo")
	}
	branch := fmt.Sprintf("refs/heads/%s", branchname)
	b := plumbing.ReferenceName(branch)

	err = worktree.Checkout(&git.CheckoutOptions{Create: true, Force: false, Branch: b})
	if err != nil {
		log.Fatal().Err(err).Msgf("could not checkout branch %s", branchname)
	}
}

func commit() {
	_, err := worktree.Add(filename)
	if err != nil {
		log.Fatal().Err(err).Msgf("could not add file %s", filename)
	}

	_, err = worktree.Commit(fmt.Sprintf("add %s agenda", branchname), &git.CommitOptions{
		Author: &object.Signature{
			Name:  "coflnet-bot",
			Email: "ci@coflnet.com",
			When:  time.Now(),
		},
		Committer: &object.Signature{
			Name:  "coflnet-bot",
			Email: "ci@coflnet.com",
			When:  time.Now(),
		},
	})

	if err != nil {
		log.Fatal().Err(err).Msg("could not commit")
	}
}

func push() {
	username := os.Getenv("GIT_USERNAME")
	token := os.Getenv("GIT_TOKEN")
	auth := &http2.BasicAuth{Username: username, Password: token}
	err := repo.Push(&git.PushOptions{
		Progress:   os.Stdout,
		RemoteName: "origin",
		Auth:       auth,
	})

	if err != nil {
		log.Fatal().Err(err).Msg("could not push")
	}
}

func pr() {
	token := os.Getenv("GIT_TOKEN")
	title := fmt.Sprintf("Agenda %s", branchname)
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/pulls", organization, repoName)
	log.Info().Msgf("url: %s", url)
	requestData := PrRequest{
		Base:  prTarget,
		Head:  branchname,
		Title: title,
	}
	serialized, err := json.Marshal(requestData)
	if err != nil {
		log.Fatal().Err(err).Msg("could not serialized request body")
	}

	data := bytes.NewBuffer(serialized)
	req, err := http.NewRequest("POST", url, data)
	req.Header.Add("Accept", "application/vnd.github.v3+json")
	req.SetBasicAuth("Flou21", token)
	log.Info().Msgf("token: %s", token)

	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal().Err(err).Msg("pr api request was not successful")
	}
	defer resp.Body.Close()

	log.Info().Msgf("response Status: %s", resp.Status)
	body, _ := ioutil.ReadAll(resp.Body)
	log.Info().Msgf("response Body: %s", string(body))
}

type PrRequest struct {
	Head  string `json:"head"`
	Base  string `json:"base"`
	Title string `json:"title"`
}
