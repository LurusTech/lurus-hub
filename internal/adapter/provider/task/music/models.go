package music

// ModelList lists models routed through the music platform.
// These names are used for channel matching and quota lookup.
var ModelList = []string{
	"suno-v4",
	"suno-v3.5",
	"udio-v1",
	"music_generate",
}

// ChannelName is the display name for admin UI.
var ChannelName = "music"
