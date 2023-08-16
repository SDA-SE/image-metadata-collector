package library

import (
	"encoding/json"
	"github.com/golang-jwt/jwt/v4"
	"net/http"
	"os"
	"strconv"
	"time"
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
