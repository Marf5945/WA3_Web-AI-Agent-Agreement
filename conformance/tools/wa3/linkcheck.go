package main

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

const (
	eLinkUnstable = "E-LINK-UNSTABLE"
	eLinkSecret   = "E-LINK-SECRET"
)

type linkCheckOutput struct {
	State    string `json:"state"`
	Source   string `json:"source"`
	Provider string `json:"provider"`
	Note     string `json:"note"`
}

func cmdLinkCheck(args []string) error {
	if len(args) != 1 {
		return errors.New("usage: wa3 link-check <source>")
	}
	source := strings.TrimSpace(args[0])
	provider, err := validateShareSource(source)
	if err != nil {
		return err
	}
	out, err := marshalJSON(linkCheckOutput{
		State:    "byte_stable_candidate",
		Source:   source,
		Provider: provider,
		Note:     "fetch bytes only, then run trust before preview or operate",
	})
	if err != nil {
		return err
	}
	fmt.Print(string(out))
	return nil
}

func validateShareSource(source string) (string, error) {
	if source == "" {
		return "", fail(eLinkUnstable, "source is empty")
	}
	if hasSecretLikeQuery(source) {
		return "", fail(eLinkSecret, "source contains token-shaped or signed URL query parameters")
	}
	u, err := url.Parse(source)
	if err != nil || u.Scheme == "" {
		return "", fail(eLinkUnstable, "source must be a stable handle or absolute URL")
	}
	switch strings.ToLower(u.Scheme) {
	case "gdrive":
		id := strings.Trim(strings.TrimPrefix(source, "gdrive://"), "/")
		if !validDriveFileID(id) {
			return "", fail(eLinkUnstable, "gdrive source must be gdrive://FILE_ID")
		}
		return "gdrive", nil
	case "https":
		return validateHTTPSShareURL(u)
	default:
		return "", fail(eLinkUnstable, "source must use gdrive:// or https://")
	}
}

func validateHTTPSShareURL(u *url.URL) (string, error) {
	host := strings.ToLower(u.Hostname())
	path := strings.ToLower(u.EscapedPath())
	if host == "" {
		return "", fail(eLinkUnstable, "https source has no host")
	}
	if host == "docs.google.com" || strings.HasSuffix(host, ".docs.google.com") {
		return "", fail(eLinkUnstable, "Google editor pages are rendered HTML, not .tdy bytes")
	}
	if host == "drive.google.com" || strings.HasSuffix(host, ".drive.google.com") {
		if path != "/uc" || u.Query().Get("export") != "download" || u.Query().Get("id") == "" {
			return "", fail(eLinkUnstable, "Google Drive browser pages must be shared as gdrive://FILE_ID or direct byte download")
		}
		if !validDriveFileID(u.Query().Get("id")) {
			return "", fail(eLinkUnstable, "Google Drive download URL has an invalid id")
		}
		return "gdrive-download", nil
	}
	if containsRenderedPageSegment(path) {
		return "", fail(eLinkUnstable, "source path looks like an edit or preview page")
	}
	if host == "github.com" && !strings.Contains(path, "/raw/") {
		return "", fail(eLinkUnstable, "GitHub sources must use raw.githubusercontent.com or a /raw/ path")
	}
	return "https", nil
}

func containsRenderedPageSegment(path string) bool {
	for _, segment := range strings.Split(path, "/") {
		switch segment {
		case "edit", "preview":
			return true
		}
	}
	return false
}

func hasSecretLikeQuery(source string) bool {
	lower := strings.ToLower(source)
	if regexp.MustCompile(`(^|[?&])(token|access_token|api_key|key|signature|x-amz-signature|x-goog-signature|x-amz-credential|x-amz-expires|expires|googleaccessid)=`).MatchString(lower) {
		return true
	}
	for _, needle := range []string{"bearer%20", "bearer+", "ghp_", "sk-", "akia"} {
		if strings.Contains(lower, needle) {
			return true
		}
	}
	return false
}

func validDriveFileID(id string) bool {
	return regexp.MustCompile(`^[A-Za-z0-9_-]{10,}$`).MatchString(id)
}

func checkLinkGuard() error {
	valids := []string{
		"gdrive://1l0AZYWHto6vlRcZrvGixerU6J9Jt1LH3",
		"https://raw.githubusercontent.com/example/repo/main/app.tdy",
		"https://static.example.com/.well-known/wa3/app.tdy",
		"https://drive.google.com/uc?export=download&id=1l0AZYWHto6vlRcZrvGixerU6J9Jt1LH3",
	}
	for _, source := range valids {
		if _, err := validateShareSource(source); err != nil {
			return fmt.Errorf("link guard rejected valid fixture %s: %w", source, err)
		}
	}
	rejects := map[string]string{
		"":                           eLinkUnstable,
		"http://example.com/app.tdy": eLinkUnstable,
		"https://drive.google.com/file/d/1l0AZYWHto6vlRcZrvGixerU6J9Jt1LH3/view":    eLinkUnstable,
		"https://docs.google.com/document/d/1l0AZYWHto6vlRcZrvGixerU6J9Jt1LH3/edit": eLinkUnstable,
		"https://example.com/app.tdy?access_token=abc123":                           eLinkSecret,
		"https://example.com/app.tdy?X-Amz-Signature=abc123":                        eLinkSecret,
		"https://github.com/example/repo/blob/main/app.tdy":                         eLinkUnstable,
		"gdrive://edit": eLinkUnstable,
	}
	for source, code := range rejects {
		_, err := validateShareSource(source)
		if err == nil {
			return fmt.Errorf("link guard accepted reject fixture: %s", source)
		}
		var we wa3Error
		if !errors.As(err, &we) || we.code != code {
			return fmt.Errorf("link guard %s got %v, want %s", source, err, code)
		}
	}
	return nil
}
