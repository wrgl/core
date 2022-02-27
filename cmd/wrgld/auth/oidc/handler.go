package authoidc

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/coreos/go-oidc"
	"github.com/gobwas/glob"
	"github.com/google/uuid"
	wrgldutils "github.com/wrgl/wrgl/cmd/wrgld/utils"
	"github.com/wrgl/wrgl/pkg/api"
	apiserver "github.com/wrgl/wrgl/pkg/api/server"
	"github.com/wrgl/wrgl/pkg/conf"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
)

type ClientSession struct {
	ClientID            string
	RedirectURI         string
	State               string
	CodeChallenge       string
	CodeChallengeMethod string
}

type Client struct {
	RedirectURIs []glob.Glob
}

type Handler struct {
	clients     map[string]Client
	corsOrigins []string
	provider    *oidc.Provider
	verifier    *oidc.IDTokenVerifier
	oidcConfig  *oauth2.Config
	handler     http.Handler

	stateMap    map[string]*ClientSession
	stateMutext sync.Mutex
}

func NewHandler(serverHandler http.Handler, c *conf.Config, client *http.Client) (h *Handler, err error) {
	if c == nil || c.Auth == nil {
		return nil, fmt.Errorf("empty auth config")
	}
	if c.Auth.OIDCProvider == nil {
		return nil, fmt.Errorf("empty auth.oidcProvider config")
	}
	if len(c.Auth.Clients) == 0 {
		return nil, fmt.Errorf("no registered client (empty auth.clients config)")
	}
	h = &Handler{
		stateMap: map[string]*ClientSession{},
		clients:  map[string]Client{},
	}
	for _, c := range c.Auth.Clients {
		client := &Client{}
		if len(c.RedirectURIs) == 0 {
			return nil, fmt.Errorf("empty redirectURIs for client %q", c.ID)
		}
		for _, s := range c.RedirectURIs {
			u, err := url.Parse(s)
			if err != nil {
				return nil, fmt.Errorf("error parsing url %q", s)
			}
			h.corsOrigins = append(h.corsOrigins, fmt.Sprintf("%s://%s", u.Scheme, u.Host))
			g, err := glob.Compile(s)
			if err != nil {
				return nil, fmt.Errorf("error compiling glob pattern %q", s)
			}
			client.RedirectURIs = append(client.RedirectURIs, g)
		}
		h.clients[c.ID] = *client
		log.Printf("client %q registered", c.ID)
	}
	ctx := context.Background()
	if client != nil {
		ctx = oidc.ClientContext(ctx, client)
	}
	if err = backoff.RetryNotify(
		func() (err error) {
			h.provider, err = oidc.NewProvider(ctx, c.Auth.OIDCProvider.Issuer)
			return err
		},
		backoff.NewExponentialBackOff(),
		func(e error, d time.Duration) {
			log.Printf("error creating oidc provider: %v. backoff for %s", e, d)
		},
	); err != nil {
		return nil, err
	}
	h.oidcConfig = &oauth2.Config{
		ClientID:     c.Auth.OIDCProvider.ClientID,
		ClientSecret: c.Auth.OIDCProvider.ClientSecret,
		RedirectURL:  strings.TrimRight(c.Auth.OIDCProvider.Address, "/") + "/oidc/callback/",
		Endpoint:     h.provider.Endpoint(),
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
	}
	h.verifier = h.provider.Verifier(&oidc.Config{
		ClientID: c.Auth.OIDCProvider.ClientID,
	})

	sm := http.NewServeMux()
	sm.HandleFunc("/oauth2/authorize/", h.handleAuthorize)
	sm.HandleFunc("/oauth2/token/", h.handleToken)
	sm.HandleFunc("/oidc/callback/", h.handleCallback)
	sm.Handle("/", wrgldutils.ApplyMiddlewares(
		serverHandler,
		func(h http.Handler) http.Handler {
			return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
				c := getClaims(r)
				if c != nil {
					r = apiserver.SetEmail(apiserver.SetName(r, c.Name), c.Email)
				}
				h.ServeHTTP(rw, r)
			})
		},
		apiserver.AuthorizeMiddleware(apiserver.AuthzMiddlewareOptions{
			Enforce: func(r *http.Request, scope string) bool {
				c := getClaims(r)
				if c != nil {
					for _, s := range c.Roles {
						if s == scope {
							return true
						}
					}
				}
				return false
			},
			GetConfig: func(r *http.Request) *conf.Config {
				return c
			},
		}),
		h.validateAccessToken,
	))
	h.handler = wrgldutils.ApplyMiddlewares(
		sm,
		h.CORSMiddleware,
	)

	return h, nil
}

func (h *Handler) CORSMiddleware(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		for _, s := range h.corsOrigins {
			rw.Header().Add("Access-Control-Allow-Origin", s)
		}
		if r.Method == http.MethodOptions {
			rw.Header().Set("Access-Control-Allow-Methods", strings.Join([]string{
				http.MethodOptions, http.MethodGet, http.MethodPost, http.MethodPut,
			}, ", "))
			rw.Header().Set("Access-Control-Allow-Headers", strings.Join([]string{
				"Authorization",
				"Cache-Control",
				"Pragma",
				"Content-Encoding",
				"Trailer",
				api.HeaderPurgeUploadPackSession,
			}, ", "))
		} else {
			handler.ServeHTTP(rw, r)
		}
	})
}

func (h *Handler) validateAccessToken(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if s := r.Header.Get("Authorization"); s != "" {
			rawIDToken := strings.Split(s, " ")[1]
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
			defer cancel()
			token, err := h.verifier.Verify(ctx, rawIDToken)
			if err != nil {
				log.Printf("failed to verify access_token: %v", err)
				apiserver.SendError(rw, http.StatusUnauthorized, "unauthorized")
				return
			}
			c := &Claims{}
			if err = token.Claims(c); err != nil {
				log.Printf("error parsing claims: %v", err)
				apiserver.SendError(rw, http.StatusInternalServerError, "internal server error")
				return
			}
			r = setClaims(r, c)
		}
		handler.ServeHTTP(rw, r)
	})
}

func (h *Handler) validClientID(clientID string) bool {
	log.Printf("validating client id %q", clientID)
	for id := range h.clients {
		if clientID == id {
			return true
		}
	}
	return false
}

func (h *Handler) validRedirectURI(clientID, uri string) bool {
	if c, ok := h.clients[clientID]; ok {
		for _, r := range c.RedirectURIs {
			if r.Match(uri) {
				return true
			}
		}
	}
	return false
}

func (h *Handler) cloneOauth2Config() *oauth2.Config {
	c := &oauth2.Config{}
	*c = *h.oidcConfig
	return c
}

func (h *Handler) saveSession(state string, ses *ClientSession) string {
	h.stateMutext.Lock()
	defer h.stateMutext.Unlock()
	if state == "" {
		state = uuid.New().String()
	}
	h.stateMap[state] = ses
	return state
}

func (h *Handler) getSession(state string) *ClientSession {
	h.stateMutext.Lock()
	defer h.stateMutext.Unlock()
	if v, ok := h.stateMap[state]; ok {
		delete(h.stateMap, state)
		return v
	}
	return nil
}

func (h *Handler) parseForm(r *http.Request) (url.Values, error) {
	if r.Method == http.MethodGet {
		return r.URL.Query(), nil
	}
	if r.Method == http.MethodPost {
		if s := r.Header.Get("Content-Type"); !strings.Contains(s, "application/x-www-form-urlencoded") {
			return nil, &HTTPError{http.StatusBadRequest, fmt.Sprintf("unsupported content type %q", s)}
		}
		defer r.Body.Close()
		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return nil, err
		}
		return url.ParseQuery(string(b))
	}
	return nil, &HTTPError{http.StatusMethodNotAllowed, "method not allowed"}
}

func (h *Handler) handleAuthorize(rw http.ResponseWriter, r *http.Request) {
	values, err := h.parseForm(r)
	if err != nil {
		handleError(rw, err)
		return
	}
	if s := values.Get("response_type"); s != "code" {
		handleError(rw, &Oauth2Error{"invalid_request", "response_type must be code"})
		return
	}
	clientID := values.Get("client_id")
	if !h.validClientID(clientID) {
		handleError(rw, &Oauth2Error{"invalid_client", "unknown client"})
		return
	}
	for _, key := range []string{
		"code_challenge",
		"state",
		"redirect_uri",
	} {
		if s := values.Get(key); s == "" {
			handleError(rw, &Oauth2Error{"invalid_request", fmt.Sprintf("%s required", key)})
			return
		}
	}
	if s := values.Get("code_challenge_method"); s != "S256" {
		handleError(rw, &Oauth2Error{"invalid_request", "code_challenge_method must be S256"})
		return
	}
	redirectURI := values.Get("redirect_uri")
	if !h.validRedirectURI(clientID, redirectURI) {
		handleError(rw, &Oauth2Error{"invalid_request", "invalid redirect_uri"})
		return
	}
	if _, err := url.Parse(redirectURI); err != nil {
		handleError(rw, &Oauth2Error{"invalid_request", fmt.Sprintf("invalid redirect_uri: %v", err)})
		return
	}
	state := h.saveSession("", &ClientSession{
		ClientID:            clientID,
		State:               values.Get("state"),
		RedirectURI:         redirectURI,
		CodeChallenge:       values.Get("code_challenge"),
		CodeChallengeMethod: values.Get("code_challenge_method"),
	})
	oauth2Config := h.cloneOauth2Config()
	http.Redirect(rw, r, oauth2Config.AuthCodeURL(state), http.StatusFound)
}

func (h *Handler) handleCallback(rw http.ResponseWriter, r *http.Request) {
	values := r.URL.Query()
	state := values.Get("state")
	if state == "" {
		handleError(rw, &Oauth2Error{"invalid_request", "state is missing"})
		return
	}
	session := h.getSession(state)
	if session == nil {
		handleError(rw, &Oauth2Error{"invalid_request", "invalid state"})
		return
	}
	uri, err := url.Parse(session.RedirectURI)
	if err != nil {
		log.Printf("error parsing redirect_uri: %v", err)
		apiserver.SendError(rw, http.StatusInternalServerError, "internal server error")
		return
	}
	query := uri.Query()
	query.Set("state", session.State)
	if errStr := values.Get("error"); errStr != "" {
		query.Set("error", errStr)
		query.Set("error_description", values.Get("error_description"))
	} else {
		code := values.Get("code")
		h.saveSession(code, &ClientSession{
			RedirectURI:         session.RedirectURI,
			ClientID:            session.ClientID,
			CodeChallenge:       session.CodeChallenge,
			CodeChallengeMethod: session.CodeChallengeMethod,
		})
		query.Set("code", code)
	}
	uri.RawQuery = query.Encode()
	http.Redirect(rw, r, uri.String(), http.StatusFound)
}

func (h *Handler) handleToken(rw http.ResponseWriter, r *http.Request) {
	values, err := h.parseForm(r)
	if err != nil {
		handleError(rw, err)
		return
	}
	if s := values.Get("grant_type"); s != "authorization_code" {
		handleError(rw, &Oauth2Error{"invalid_request", "grant_type must be authorization_code"})
		return
	}
	code := values.Get("code")
	if code == "" {
		handleError(rw, &Oauth2Error{"invalid_request", "code required"})
		return
	}
	session := h.getSession(values.Get("code"))
	if session == nil {
		handleError(rw, &Oauth2Error{"invalid_request", "invalid code"})
		return
	}
	if s := values.Get("client_id"); s != session.ClientID {
		handleError(rw, &Oauth2Error{"invalid_client", "invalid client_id"})
		return
	}
	redirectURI, err := url.Parse(values.Get("redirect_uri"))
	if err != nil {
		handleError(rw, &Oauth2Error{"invalid_request", "failed to parse redirect_uri"})
		return
	}
	if s := fmt.Sprintf("%s://%s%s", redirectURI.Scheme, redirectURI.Host, redirectURI.Path); s != session.RedirectURI {
		log.Printf("redirect URI does not match %q != %q", s, session.RedirectURI)
		handleError(rw, &Oauth2Error{"invalid_request", "redirect_uri does not match"})
		return
	}
	if s := values.Get("code_verifier"); s == "" {
		handleError(rw, &Oauth2Error{"invalid_grant", "code_verifier required"})
		return
	} else {
		hash := sha256.New()
		if _, err := hash.Write([]byte(s)); err != nil {
			panic(err)
		}
		if base64.RawURLEncoding.EncodeToString(hash.Sum([]byte{})) != session.CodeChallenge {
			handleError(rw, &Oauth2Error{"invalid_grant", "code challenge failed"})
			return
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	token, err := h.oidcConfig.Exchange(ctx, values.Get("code"))
	if err != nil {
		log.Printf("error: error exchanging code: %v", err)
		apiserver.SendError(rw, http.StatusInternalServerError, "internal server error")
		return
	}
	if token.TokenType != "Bearer" {
		log.Printf("error: expected bearer token, found %q", token.TokenType)
		apiserver.SendError(rw, http.StatusInternalServerError, "internal server error")
		return
	}
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		log.Printf("error: no id_token field in oauth2 token")
		apiserver.SendError(rw, http.StatusInternalServerError, "internal server error")
		return
	}
	if _, err = h.verifier.Verify(context.Background(), rawIDToken); err != nil {
		log.Printf("error: failed to verify id_token: %v", err)
		apiserver.SendError(rw, http.StatusInternalServerError, "internal server error")
		return
	}

	rw.Header().Set("Cache-Control", "no-store")
	rw.Header().Set("Pragma", "no-cache")
	apiserver.WriteJSON(rw, &TokenResponse{
		AccessToken: rawIDToken,
		TokenType:   token.TokenType,
	})
}

func (h *Handler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	h.handler.ServeHTTP(rw, r)
}
