package git

import (
	"fmt"
	"os"
	"time"

	"encoding/json"
	"github.com/rs/zerolog/log"
	"net/http"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/golang-jwt/jwt/v4"
	"strconv"
)

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

type gitParameters struct {
	url        string `validate:"required"`
	directory  string `validate:"required"`
	repository *git.Repository
}

func NewGit(url, directory, privateKeyFile, password string, githubAppId, githubInstallationId int64) (*gitParameters, error) {

	if url == "" {
		log.Info().Msg("git url not given, do not init git")
		return nil, fmt.Errorf("Missing git Url")
	}

	if _, err := os.Stat(privateKeyFile); err != nil {
		log.Warn().Str("privateKeyFile", privateKeyFile).Err(err).Msg("read file failed")
		return nil, err
	}

	if _, err := os.Stat(directory); !os.IsNotExist(err) {
		// TODO: What happens with this error?
		err = os.RemoveAll(directory)
	}

	// Clone the given repository to the given directory
	log.Info().Str("url", url).Int64("githubInstallationId", githubInstallationId).Msg("cloning")

	var cloneOptions git.CloneOptions

	// TODO: Can this be cleaned up w/o mentioning GH?
	if githubInstallationId != 0 {

		// TODO: Review lib
		token, err := GetGithubToken(privateKeyFile, githubAppId, githubInstallationId)
		if err != nil {
			return nil, err
		}

		// TODO: Review is this GH specific or actually general?
		// Do we need support for Bitbucket?
		githubUrl := "https://x-access-token:" + token + "@" + url
		cloneOptions = git.CloneOptions{
			URL:      githubUrl,
			Progress: os.Stdout,
		}
	} else {

		publicKeys, err := ssh.NewPublicKeysFromFile("git", privateKeyFile, password)
		if err != nil {
			log.Warn().Err(err).Msg("generate publickeys failed")
			return nil, err
		}

		cloneOptions = git.CloneOptions{
			URL:      url,
			Auth:     publicKeys,
			Progress: os.Stdout,
		}
	}

	// What is set to false here?
	repository, err := git.PlainClone(directory, false, &cloneOptions)

	if err != nil {
		log.Warn().Err(err).Msg("could not clone")
		return nil, err
	}

	g := &gitParameters{
		url:        url,
		directory:  directory,
		repository: repository,
	}

	return g, nil
}

func (g gitParameters) Upload(content []byte, fileName, environmentName string) error {
	filepath := g.directory + "/" + fileName
	worktree, _ := g.repository.Worktree()

	err := os.WriteFile(filepath, content, 0755)
	if err != nil {
		log.Info().Stack().Err(err).Str("filepath", filepath).Msg("Error during opening file")
	}

	if _, err := worktree.Add(fileName); err != nil {
		return err
	}

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

	obj, err := g.repository.CommitObject(commit)
	if err != nil {
		log.Warn().Err(err).Msg("could get committed object")
		return nil
	}
	log.Info().Str("obj", obj.String()).Msg("committed")

	err = g.repository.Push(&git.PushOptions{})
	if err != nil {
		log.Warn().Err(err).Msg("could not push")
		return nil
	}

	return nil
}
