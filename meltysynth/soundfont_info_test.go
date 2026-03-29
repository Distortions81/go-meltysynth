package meltysynth

import "testing"

func TestSoundFontInfo_AuthorCompatibilityAlias(t *testing.T) {
	soundFont := loadGM(t)

	if soundFont.Info.Author != soundFont.Info.Auther {
		t.Fatalf("author fields differ: Author=%q Auther=%q", soundFont.Info.Author, soundFont.Info.Auther)
	}
}
