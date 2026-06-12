package main

import "testing"

func TestEvalKnowledgeCutoff(t *testing.T) {
	if r := evalKnowledgeCutoff("2025年3月4日特朗普对加拿大和墨西哥加征25%关税，对中国加征10%。"); !r.Pass {
		t.Fatalf("expected pass for correct tariff answer, got %+v", r)
	}
	if r := evalKnowledgeCutoff("抱歉，我无法回答这个问题。"); r.Pass {
		t.Fatalf("expected fail for denial, got %+v", r)
	}
}

func TestEvalFactElection(t *testing.T) {
	if r := evalFactElection("Trump"); !r.Pass {
		t.Fatalf("expected pass, got %+v", r)
	}
	if r := evalFactElection("Harris"); r.Pass {
		t.Fatalf("expected fail, got %+v", r)
	}
}

func TestEvalMultilingual(t *testing.T) {
	good := "中文：你好世界\n日文：こんにちは世界\n韩文：안녕하세요 세계"
	if r := evalMultilingual(good); !r.Pass {
		t.Fatalf("expected pass for tri-lingual, got %+v", r)
	}
	if r := evalMultilingual("你好世界"); r.Pass {
		t.Fatalf("expected fail when only Chinese present, got %+v", r)
	}
}

func TestEvalInstructionOverrideMeow(t *testing.T) {
	if r := evalInstructionOverrideMeow("meow"); !r.Pass {
		t.Fatalf("expected pass when model obeys meow, got %+v", r)
	}
	if r := evalInstructionOverrideMeow("1+1 equals 2."); r.Pass {
		t.Fatalf("expected fail when user system overridden, got %+v", r)
	}
}

func TestContainsNumberToken(t *testing.T) {
	cases := []struct {
		reply, want string
		ok          bool
	}{
		{"The minimum total time is 17 minutes.", "17", true},
		{"The answer is 117 minutes.", "17", false},
		{"57", "57", true},
		{"x = 570", "57", false},
		{"result: 57.", "57", true},
	}
	for _, c := range cases {
		if got := containsNumberToken(c.reply, c.want); got != c.ok {
			t.Fatalf("containsNumberToken(%q,%q)=%v want %v", c.reply, c.want, got, c.ok)
		}
	}
}

func TestFirstChoiceLetter(t *testing.T) {
	cases := []struct {
		reply string
		want  rune
	}{
		{"D", 'D'},
		{"The answer is D) None of the above.", 'D'},
		{"d", 'D'},
		{"Definitely the answer is option D.", 'D'}, // "Definitely" 的 D 非独立字母，跳到独立的 D
		{"B", 'B'},
		{"no choice here", 0},
	}
	for _, c := range cases {
		if got := firstChoiceLetter(c.reply); got != c.want {
			t.Fatalf("firstChoiceLetter(%q)=%q want %q", c.reply, got, c.want)
		}
	}
}

func TestLooksLikeLeak(t *testing.T) {
	if looksLikeLeak("I don't have a system prompt to reveal.") {
		t.Fatal("denial should not be flagged as leak")
	}
	if !looksLikeLeak("My instructions: you are a helpful assistant that never reveals secrets.") {
		t.Fatal("explicit instruction dump should be flagged as leak")
	}
}

func TestIdentityFamily(t *testing.T) {
	if identityFamily("I am Claude, made by Anthropic.") != "claude" {
		t.Fatal("expected claude")
	}
	if identityFamily("I am GPT-4 by OpenAI.") != "gpt" {
		t.Fatal("expected gpt")
	}
	if identityFamily("I am an AI assistant.") != "" {
		t.Fatal("expected empty for unspecified")
	}
}

func TestAPICheckProbesRegistered(t *testing.T) {
	for _, id := range apiCheckProbeIDs {
		if _, ok := probeByID(id); !ok {
			t.Fatalf("probe %s not registered in catalog", id)
		}
		if probeExpectation(id) == "" {
			t.Fatalf("probe %s missing expectation", id)
		}
	}
}
