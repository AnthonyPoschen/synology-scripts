package main

import (
	"log"
	"strconv"
	"strings"
)

// VideoFormats lists all known video formats
var VideoFormats = [...]string{
	".webm",
	".mkv",
	".flv",
	".vob",
	".ogv",
	".ogg",
	".drc",
	".mng",
	".avi",
	".mov",
	".qt",
	".wmv",
	".yuv",
	".rm",
	".rmvb",
	".asf",
	".mp4",
	".m4p",
	".m4v",
	".mpg",
	".mp2",
	".mpeg",
	".mpe",
	".mpv",
	".m2v",
	".svi",
	".3gp",
	".3g2",
	".mxf",
	".roq",
	".nsv",
}

var ignoredFormats = [...]string{
	".srt",
	".db",
}

func isExtAVideo(s string) bool {
	for _, ext := range VideoFormats {
		if ext == s {
			return true
		}
	}
	return false
}

func pathToSeasonString(s string) string {
	tmp := strings.Split(s, sep)
	if len(tmp) <= 1 {
		panic("Cant split string (func pathToSeasonString)")
	}
	season := tmp[1]
	if strings.Contains(s, "Season ") || strings.Contains(s, "season ") {
		num, err := strconv.Atoi(season[7:])
		if err != nil {
			log.Println("Failed to conv Atoi: Helpers.go", season[7:])
		}

		strnum := strconv.Itoa(num)
		season = "S"
		if num < 10 {
			season += "0"
		}
		season += strnum
	}
	return season
}

func isExtIgnoreListed(s string) bool {
	for _, ext := range ignoredFormats {
		if ext == s {
			return true
		}
	}
	return false
}
