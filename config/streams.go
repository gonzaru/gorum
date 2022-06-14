// by Gonzaru
// Distributed under the terms of the GNU General Public License v3

package config

// Streams urls, needs more :)
var Streams = map[int]map[string]string{
	1: {
		"name":    "Русские Песни",
		"nameIcy": "RUSSIAN SONGS",
		"url":     "https://listen.rusongs.ru:8005/ru-mp3-128",
	},
	2: {
		"name":    "Nebenwelten",
		"nameIcy": "Nebenwelten",
		"url":     "https://stream.laut.fm/nebenwelten",
	},
	3: {
		"name":    "Goa Base",
		"nameIcy": "Goa Base",
		"url":     "https://goa-base.stream.laut.fm/goa-base",
	},
	4: {
		"name":    "Hohenburg",
		"nameIcy": "Radiohohenburg",
		"url":     "https://stream.laut.fm/radiohohenburg",
	},
	5: {
		"name":    "SynthWay",
		"nameIcy": "SynthWay Radio",
		"url":     "https://c24.radioboss.fm:18014/stream",
	},
	6: {
		"name":    "Зайцев ФМ",
		"nameIcy": "zaycev.fm (metal mp3 stream 256kb)",
		"url":     "https://zaycevfm.cdnvideo.ru/ZaycevFM_metal_256.mp3",
	},
	7: {
		"name":    "7 Rays",
		"nameIcy": "7 Rays",
		"url":     "https://7rays.stream.laut.fm/7rays",
	},
	8: {
		"name":    "Ancient FM",
		"nameIcy": "Ancient FM",
		"url":     "https://mediaserv73.live-streams.nl:18058/stream",
	},
	9: {
		"name":    "Enigmatic Station",
		"nameIcy": "Enigmatic robot",
		"url":     "https://myradio24.org/8226",
	},
	10: {
		"name":    "Fly FM",
		"nameIcy": "Radio FLYFM",
		"url":     "http://flyfm.net:8000/flyfm",
	},
}
