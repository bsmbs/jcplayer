package main

import (
	"strconv"
)

type Track struct {
	Id    int    `json:"id"`
	Type  string `json:"type"`
	Lang  string `json:"lang"`
	Title string `json:"title"`
}

type Utrack struct {
	Id    int
	Type  string
	Track map[string]interface{}
}

func (player *Player) GetTracks() []Utrack {
	trackcount, err := player.Conn.Get("track-list/count")
	Nerr(err)

	tracks := []Utrack{}
	for i := 0; i < int(trackcount.(float64)); i++ {
		trackG, err := player.Conn.Get("track-list/" + strconv.Itoa(i))
		Nerr(err)

		trackP := trackG.(map[string]interface{})

		trackId := int(trackP["id"].(float64))
		trackType := trackP["type"].(string)

		newTrack := Utrack{Id: trackId, Type: trackType, Track: trackP}

		tracks = append(tracks, newTrack)

	}
	return tracks
}

//var tracks []Track
/*for i, d := range tracklist.([]interface{}) {
	fmt.Println("====", i, "====")
	fmt.Println(d.(Track))
}
inf1 := tracklist.([]interface{})

//json.Unmarshal([]byte(tracklist.([]string)), &tracks)
fmt.Println(inf1[0])*/
