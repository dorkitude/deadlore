package wiki

import "testing"

func TestCacheUsesRequestedTitleRatherThanResolvedPageTitle(t *testing.T) {
	cache, err := NewCache(t.TempDir())
	if err != nil {
		t.Fatalf("NewCache: %v", err)
	}
	page := &Page{Title: "Haze", URL: "https://deadlock.wiki/Sleep_Dagger", CacheVersion: CurrentCacheVersion}
	if err := cache.Save("Sleep Dagger", page); err != nil {
		t.Fatalf("Save: %v", err)
	}

	cached, err := cache.Load("Sleep Dagger")
	if err != nil || cached == nil {
		t.Fatalf("Load requested title = (%#v, %v), want page", cached, err)
	}
	if cached.URL != page.URL {
		t.Fatalf("cached URL = %q, want %q", cached.URL, page.URL)
	}
	resolved, err := cache.Load("Haze")
	if err != nil {
		t.Fatalf("Load resolved title: %v", err)
	}
	if resolved != nil {
		t.Fatalf("resolved-title cache should be empty, got %#v", resolved)
	}
}
