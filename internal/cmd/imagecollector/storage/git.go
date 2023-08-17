package storage

//github fetch token from https://github.com/google/go-github
// use the lib

import (
	"github.com/SDA-SE/sdase-image-collector/internal/cmd/imagecollector/library"
	"github.com/SDA-SE/sdase-image-collector/internal/cmd/imagecollector/model"
	"github.com/rs/zerolog/log"
	"os"
	"time"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
)

var url string
var directory string
var privateKeyFile string
var password string
var repository *git.Repository
var worktree *git.Worktree

func InitGit(gitParameterEntry model.GitParameterEntry) error {
	if gitParameterEntry.Url == "" {
		log.Info().Msg("git url not given, do not init git")
		return nil
	}
	url = gitParameterEntry.Url
	directory = gitParameterEntry.Directory
	privateKeyFile = gitParameterEntry.PrivateKeyFile
	password = gitParameterEntry.Password

	_, err := os.Stat(privateKeyFile)
	if err != nil {
		log.Warn().Str("privateKeyFile", privateKeyFile).Err(err).Msg("read file failed")
		return err
	}
	if _, err := os.Stat(directory); !os.IsNotExist(err) {
		err = os.RemoveAll(directory)
	}

	// Clone the given repository to the given directory
	log.Info().Str("url", url).Int64("gitParameterEntry.GithubInstallationId", gitParameterEntry.GithubInstallationId).Msg("cloning")
	var cloneOptions git.CloneOptions
	if gitParameterEntry.GithubInstallationId != 0 {
		token, err := library.GetGithubToken(gitParameterEntry)
		if err != nil {
			return err
		}
		githubUrl := "https://x-access-token:" + token + "@" + url
		cloneOptions = git.CloneOptions{
			URL:      githubUrl,
			Progress: os.Stdout,
		}
	} else {
		publicKeys, err := ssh.NewPublicKeysFromFile("git", privateKeyFile, password)
		if err != nil {
			log.Warn().Err(err).Msg("generate publickeys failed")
			return err
		}
		cloneOptions = git.CloneOptions{
			URL:      url,
			Auth:     publicKeys,
			Progress: os.Stdout,
		}
	}
	repository, err = git.PlainClone(directory, false, &cloneOptions)
	if err != nil {
		log.Warn().Err(err).Msg("could not clone")
		return err
	}

	// ... retrieving the branch being pointed by HEAD
	ref, err := repository.Head()
	// ... retrieving the commit object
	commit, _ := repository.CommitObject(ref.Hash())
	worktree, err = repository.Worktree()
	_, err = worktree.Add("output.json")

	log.Info().Str("commit", commit.String()).Msg("commit")
	return nil
}

func GitUpload(content []byte, filename string) error {
	if url == "" {
		log.Info().Msg("git url not given, do not upoad to git")
		return nil
	}

	filepath := directory + "/" + filename
	log.Info().Str("filepath", filepath).Msg("")
	library.SaveFile(filepath, content)

	_, err := worktree.Add(filename)
	commit, err := worktree.Commit("example go-git commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "ClusterImageScanner",
			Email: "",
			When:  time.Now(),
		},
	})
	if err != nil {
		log.Warn().Err(err).Msg("could not create worktree")
		return nil
	}

	obj, err := repository.CommitObject(commit)
	if err != nil {
		log.Warn().Err(err).Msg("could get committed object")
		return nil
	}
	log.Info().Str("obj", obj.String()).Msg("committed")

	err = repository.Push(&git.PushOptions{})
	if err != nil {
		log.Warn().Err(err).Msg("could not push")
		return nil
	}
	return nil
}
