package git

import (
	"fmt"
	"io"
	"os"
	"time"

	"encoding/json"
	"github.com/rs/zerolog/log"
	"net/http"
	"path/filepath"

	goGit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/golang-jwt/jwt/v4"
	"strconv"
)

type GitConfig struct {
	GitUrl               string
	GitDirectory         string
	GitPrivateKeyFile    string
	GitPassword          string
	GithubAppId          int64
	GithubInstallationId int64
}

type AuthTokenClaim struct {
	*jwt.StandardClaims
}

type InstallationAuthResponse struct {
	Token       string    `json:"token"`
	ExpiresAt   time.Time `json:"expires_at"`
	Permissions struct {
		Checks       string `json:"checks"`
		Contents     string `json:"contents"`
		Deployments  string `json:"deployments"`
		Metadata     string `json:"metadata"`
		PullRequests string `json:"pull_requests"`
		Statuses     string `json:"statuses"`
	} `json:"permissions"`
	RepositorySelection string `json:"repository_selection"`
}

func GetGithubToken(privateKeyFile string, githubAppId, githubInstallationId int64) (string, error) {
	keyBytes, err := os.ReadFile(privateKeyFile)
	if err != nil {
		return "", err
	}

	rsaPrivateKey, err := jwt.ParseRSAPrivateKeyFromPEM(keyBytes)
	if err != nil {
		return "", err
	}

	jwtToken := jwt.New(jwt.SigningMethodRS256)

	jwtToken.Claims = &AuthTokenClaim{
		&jwt.StandardClaims{
			IssuedAt:  time.Now().Unix(),
			ExpiresAt: time.Now().Add(time.Minute * 9).Unix(),
			Issuer:    strconv.FormatInt(githubAppId, 10),
		},
	}

	tokenString, err := jwtToken.SignedString(rsaPrivateKey)
	if err != nil {
		return "", err
	}

	client := &http.Client{}
	url := "https://api.github.com/app/installations/" + strconv.FormatInt(githubInstallationId, 10) + "/access_tokens"
	req, _ := http.NewRequest("POST", url, nil)
	req.Header.Set("Accept", "application/vnd.github.machine-man-preview+json")
	req.Header.Set("Authorization", "Bearer "+tokenString)
	res, _ := client.Do(req)

	decoder := json.NewDecoder(res.Body)
	var installationAuthResponse InstallationAuthResponse
	err = decoder.Decode(&installationAuthResponse)
	if err != nil {
		return "", err
	}
	return installationAuthResponse.Token, nil
}

type git struct {
	repository *goGit.Repository
	fileName   string
}

func NewGit(cfg *GitConfig, filename string) (io.Writer, error) {

	if cfg.GitUrl == "" {
		log.Info().Msg("git url not given, do not init git")
		return nil, fmt.Errorf("Missing git Url")
	}

	if _, err := os.Stat(cfg.GitPrivateKeyFile); err != nil {
		log.Warn().Str("privateKeyFile", cfg.GitPrivateKeyFile).Err(err).Msg("read file failed")
		return nil, err
	}

	if _, err := os.Stat(cfg.GitDirectory); !os.IsNotExist(err) {
		err = os.RemoveAll(cfg.GitDirectory)

		if err != nil {
			log.Warn().Err(err).Msg("Could not remove directory")
		}
	}

	// Clone the given repository to the given directory
	log.Info().Str("url", cfg.GitUrl).Int64("githubInstallationId", cfg.GithubInstallationId).Msg("cloning")

	var cloneOptions goGit.CloneOptions

	// TODO: Can this be cleaned up w/o mentioning GH?
	if cfg.GithubInstallationId != 0 {

		// TODO: Review lib
		token, err := GetGithubToken(cfg.GitPrivateKeyFile, cfg.GithubAppId, cfg.GithubInstallationId)
		if err != nil {
			return nil, err
		}

		// TODO: Review is this GH specific or actually general?
		// Do we need support for Bitbucket?
		githubUrl := "https://x-access-token:" + token + "@" + cfg.GitUrl
		cloneOptions = goGit.CloneOptions{
			URL:      githubUrl,
			Progress: os.Stdout,
		}
	} else {

		publicKeys, err := ssh.NewPublicKeysFromFile("git", cfg.GitPrivateKeyFile, cfg.GitPassword)
		if err != nil {
			log.Warn().Err(err).Msg("generate publickeys failed")
			return nil, err
		}

		cloneOptions = goGit.CloneOptions{
			URL:      cfg.GitUrl,
			Auth:     publicKeys,
			Progress: os.Stdout,
		}
	}

	// What is set to false here?
	repository, err := goGit.PlainClone(cfg.GitDirectory, false, &cloneOptions)

	if err != nil {
		log.Warn().Err(err).Msg("could not clone")
		return nil, err
	}

	g := &git{
		repository: repository,
		fileName:   filepath.Join(cfg.GitDirectory, filename),
	}

	return g, nil
}

func (g git) Write(content []byte) (int, error) {
	worktree, _ := g.repository.Worktree()

	err := os.WriteFile(g.fileName, content, 0755)
	if err != nil {
		log.Info().Stack().Err(err).Str("filename", g.fileName).Msg("Error during opening file")
	}

	if _, err := worktree.Add(g.fileName); err != nil {
		return 0, err
	}

	commit, err := worktree.Commit("example go-git commit", &goGit.CommitOptions{
		Author: &object.Signature{
			Name:  "ClusterImageScanner",
			Email: "",
			When:  time.Now(),
		},
	})

	if err != nil {
		log.Warn().Err(err).Msg("could not create worktree")
		return 0, err
	}

	obj, err := g.repository.CommitObject(commit)
	if err != nil {
		log.Warn().Err(err).Msg("could not get committed object")
		return 0, err
	}
	log.Info().Str("obj", obj.String()).Msg("committed")

	err = g.repository.Push(&goGit.PushOptions{})
	if err != nil {
		log.Warn().Err(err).Msg("could not push")
		return 0, err
	}

	return len(content), nil
}
