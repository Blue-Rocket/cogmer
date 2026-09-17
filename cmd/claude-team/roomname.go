package main

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
)

// Room names are generated, never chosen (§12).
//
// A name a developer picks will be the name of a project, a client, or a ticket,
// and rooms named after projects become rooms scoped to projects by convention --
// the model this specification deliberately abandoned. Generating the name resists
// that without relying on anyone's discipline.
//
// Weather and landscape, because an invitation may be read aloud: speakable,
// unambiguous when heard, short enough to type, and drawn from a narrow neutral
// domain so no random pairing embarrasses anyone. They also name a place, which is
// what a room is.

var roomSky = []string{
	"amber", "arctic", "aurora", "autumn", "azure", "balmy", "blizzard", "breezy",
	"bright", "brisk", "calm", "cirrus", "clear", "cloudless", "cobalt", "cold",
	"crisp", "cyclone", "damp", "dawn", "daybreak", "drifting", "drizzle", "dusk",
	"dusty", "eastern", "ember", "equinox", "evening", "fair", "flurry", "foggy",
	"freezing", "frost", "gale", "gentle", "glacial", "gleaming", "golden", "gusty",
	"hail", "halcyon", "harvest", "haze", "humid", "icy", "indigo", "lightning",
	"lunar", "mild", "mist", "monsoon", "moonlit", "morning", "northern", "overcast",
	"polar", "rainy", "scarlet", "shimmer", "silver", "sleet", "snowfall", "solar",
	"solstice", "southern", "sparkling", "squall", "starlit", "stormy", "sultry",
	"summer", "sunlit", "sunrise", "sunset", "temperate", "tempest", "thunder",
	"tranquil", "twilight", "vapour", "violet", "warm", "western", "wildfire",
	"windward", "winter", "zephyr",
}

var roomLand = []string{
	"anchorage", "arbour", "atoll", "badlands", "basin", "bayou", "beacon", "bluff",
	"bog", "boulder", "brook", "burrow", "canyon", "cape", "cascade", "cavern",
	"channel", "clearing", "cliff", "coastline", "cove", "crag", "crater", "creek",
	"delta", "dunes", "estuary", "fathom", "fell", "fenland", "fjord", "foothill",
	"ford", "forest", "glacier", "glade", "glen", "gorge", "grotto", "grove",
	"harbour", "headland", "heath", "highland", "hollow", "inlet", "island", "isthmus",
	"lagoon", "lakeshore", "ledge", "lowland", "marsh", "meadow", "mesa", "moorland",
	"narrows", "oasis", "outcrop", "overlook", "pass", "pinnacle", "plateau", "prairie",
	"quarry", "rapids", "ravine", "reef", "ridge", "rivulet", "sandbar", "savanna",
	"shoal", "shoreline", "sound", "spring", "steppe", "strait", "summit", "thicket",
	"tideland", "timberline", "tundra", "vale", "wetland", "wildwood", "woodland",
}

// NewRoomName generates a name. Uniqueness is only ever needed among the rooms one
// peer hosts (§12), because a name is only resolved against a specific peer -- so
// the caller regenerates on a local collision rather than coordinating globally.
func NewRoomName() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	sky := roomSky[binary.BigEndian.Uint32(b[0:4])%uint32(len(roomSky))]
	land := roomLand[binary.BigEndian.Uint32(b[4:8])%uint32(len(roomLand))]
	return fmt.Sprintf("%s-%s", sky, land)
}

// RoomNameSpace bounds accidental collision, not impersonation: a room name is
// never a credential (§12).
func RoomNameSpace() int { return len(roomSky) * len(roomLand) }
