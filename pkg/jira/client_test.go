package jira

import (
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/net/html"
	"gopkg.in/andygrunwald/go-jira.v1"
)

func TestLogin(t *testing.T) {
	cookieJar, err := cookiejar.New(nil)
	require.NoError(t, err)
	client := &http.Client{
		Jar: cookieJar,
	}
	resp, err := client.Get("https://issues.redhat.com/login.jsp?os_destination=%2Fdefault.jsp")
	require.NoError(t, err)
	require.Equal(t, resp.StatusCode, http.StatusOK)
	samlToken := getSAMLRequest(resp)
	require.NotEqual(t, "", samlToken)
	require.NoError(t, resp.Body.Close())

	formData := url.Values{"SAMLRequest": {samlToken}}
	t.Logf("%s", formData.Encode())
	resp, err = client.PostForm("https://sso.redhat.com/auth/realms/redhat-external/protocol/saml", formData)
	require.NoError(t, err)
	loginURL := getFormURL(resp)
	t.Log(loginURL)
	require.NoError(t, resp.Body.Close())

	loginData := url.Values{"username": {os.Getenv("TEST_USER")}, "password": {os.Getenv("TEST_PASS")}}
	resp, err = client.PostForm(loginURL, loginData)
	require.NoError(t, err)

	// tokenizer misses the input obj for this response for some reason, parse the whole doc
	doc, err := html.Parse(resp.Body)
	require.NoError(t, err)
	samlResp := getSAMLResponse(doc)
	require.NoError(t, resp.Body.Close())

	samlRespFormData := url.Values{"SAMLResponse": {samlResp}}
	t.Logf("%s", samlRespFormData.Encode())
	resp, err = client.PostForm("https://sso.jboss.org/login?provider=RedHatExternalProvider", samlRespFormData)
	require.NoError(t, err)
	require.Equal(t, resp.StatusCode, http.StatusOK)

	jc, err := jira.NewClient(client, "https://issues.redhat.com")
	i, r, err := jc.Issue.Get("OLM-1378", &jira.GetQueryOptions{})
	require.NoError(t, err)
	t.Logf("%#v", i)
	t.Logf("%#v", i.Fields)
	t.Logf("%#v", r)
}
