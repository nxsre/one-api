package aippt

import "testing"

func TestParseChannelKey_pipe(t *testing.T) {
	app, sec, uid, err := ParseChannelKey("myApp|mySecret")
	if err != nil {
		t.Fatal(err)
	}
	if app != "myApp" || sec != "mySecret" || uid != "openclaw_default" {
		t.Fatalf("got %q %q %q", app, sec, uid)
	}
}

func TestParseChannelKey_rejects_sk_token(t *testing.T) {
	_, _, _, err := ParseChannelKey("sk-wn7p7mgtvd9CP7jg14433dAe193440419329A8D2C2B2FdA5")
	if err == nil {
		t.Fatal("expected error for sk- token in channel key")
	}
}
