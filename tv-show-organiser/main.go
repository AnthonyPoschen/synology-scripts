package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
)

// Shows is names (also folder path relative to TVRootDir) of tv shows and the containing season folders in its root
type Shows struct {
	name    string
	Seasons []string
}

// is the root dir of the tv folder
var TVRootDir string
var tvshows []string
var testmode bool
var testtext = "TEST MODE:"
var sep string

func init() {

}

func main() {
	flag.StringVar(&TVRootDir, "folderroot", "TV", "folder root directory relative or full for the tv folder to be parsed")
	flag.StringVar(&TVRootDir, "f", "TV", "folder root directory relative or full for the tv folder to be parsed (ShortHand)")
	flag.BoolVar(&testmode, "t", false, "If present test mode is enabled")
	flag.BoolVar(&testmode, "test", false, "If present test mode is enabled")
	flag.Parse()
	tvshows = make([]string, 0)
	sep = "/"
	if runtime.GOOS == "windows" {
		sep = "\\"
	}
	os.Mkdir("logs", 0666)
	f, err := os.OpenFile("./logs/ShowOrganiserLog.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		panic("Cant Make Log File!")
	}
	defer f.Close()
	log.SetOutput(f)
	log.Println("-----------------------------------------------------------------------------")
	log.Println("----------------------------------Launching----------------------------------")
	log.Println("-----------------------------------------------------------------------------")
	fmt.Println("Changing Dir:", TVRootDir)
	os.Chdir(TVRootDir)
	filepath.Walk(".", getTvShows)
	wg := sync.WaitGroup{}
	wg.Add(len(tvshows))
	ch := make(chan Shows, len(tvshows))
	// now that we have every tv show's folder name we will be parsing.
	// lets search each one concurrently for valid Season folders.
	for _, name := range tvshows {
		go mapShowSeasons(name, &wg, ch)
	}
	// wait till we have all the seasons available in the channel
	wg.Wait()
	close(ch)

	// now we know both how many shows we have and the seasons.
	// we can parse every show season combination concurrently
	// and through a two layered heirachery incase we need to do
	// anything inbatween seasons of a show. like having a show in
	// the wrong folder of a season. and once every show is complete
	// we can end the program
	wg.Add(len(tvshows))
	for data := range ch {
		go parseShow(data, &wg)
	}
	wg.Wait()
}

func getTvShows(path string, info os.FileInfo, err error) error {
	//filepath.HasPrefix(p, prefix)
	if path == "." {
		return nil
	}
	if info.IsDir() {
		if strings.Contains(path, sep) {
			return nil
		}
		tvshows = append(tvshows, path)
	}
	return nil
}

// gets called in go routines by main
func mapShowSeasons(path string, wg *sync.WaitGroup, ch chan<- Shows) {
	var seasons []string
	seasons = make([]string, 0)

	filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if p == path {
			return nil
		}
		if info.IsDir() {
			// (?i) 	= case insensitive
			// ^ 		= start of string
			// [0-9]* 	= any number 0-9 repeated any ammount of times
			// $ 		= end of string
			if regexp.MustCompile("(?i)^Season [0-9]*$").Match([]byte(info.Name())) {
				seasons = append(seasons, p)
			}
		}
		return nil
	})
	ch <- Shows{path, seasons}
	wg.Done()
}

//inside the shows folder is where all new shows of a season should be dumped.
//it can recall itself if its seasons find folders inside them and moves them
// back into the show folder.
func parseShow(s Shows, wg *sync.WaitGroup) {
	// let all our seasons pass first. so they can move folders back into this main folder.
	// once that is done it is out job to place the shows we can match into the correct folder.
	// otherwise we log an issue.
	defer wg.Done()
	if len(s.Seasons) == 0 {
		log.Println("Cant Resolve Seasons for:", s.name)
		return
	}
	wgs := sync.WaitGroup{}
	wgs.Add(len(s.Seasons))
	for _, name := range s.Seasons {
		go parseSeason(name, &wgs)
	}
	wgs.Wait()
}

func parseSeason(path string, wgs *sync.WaitGroup) {
	f, err := filepath.Glob(path + sep + "*")

	if err != nil {
		log.Println("Failed to search path in parse season with path:", path+"/*")
	}
	// for every file and folder
	for _, n := range f {
		fi, err := os.Stat(n)
		if err != nil {
			continue
		}
		// if the object is a folder
		if fi.Mode().IsDir() {
			// some weird directory in snyology folders
			// lets skip this
			if strings.Contains(fi.Name(), "@eaDir") {
				continue
			}
			// move it back to the show directory to be handled. since it doesnt belong
			// lets get the video out
			log.Println("Found Folder in Season Folder")

			// instead of moving back a directory lets fetch our video and move that up a directory
			// and then cleanup the folder so everything is golden :).
			/*
				if testmode {
					log.Println(testtext+"Move Folder:", n, "| To new folder |", showpath+sep+fi.Name())
				} else {
					log.Println("Move Folder:", n, "| To new folder |", showpath+sep+fi.Name())
					os.Rename(n, showpath+sep+fi.Name())
				}
				continue
			*/
			// we must fetch all files in this folder incase multiple video files and simply
			// move back the biggest file since that is always the main video.

			files, err := filepath.Glob(path + sep + fi.Name() + sep + "*")
			if err != nil {
				continue
			}
			mainvideo := ""
			for _, v := range files {
				ext := filepath.Ext(v)
				if isExtAVideo(ext) {
					if mainvideo == "" {
						mainvideo = v
						continue
					}
					mf, _ := os.Stat(mainvideo)
					nf, _ := os.Stat(v)
					if nf.Size() > mf.Size() {
						mainvideo = v
					}
				}
			}

			if mainvideo == "" {
				// do not delete the folder incase its a place holder we only want to target video folders
				continue
			}
			// now that we know we have a video lets copy it out one directory.
			//dir := filepath.Dir(mainvideo)
			newDir := filepath.Dir(mainvideo)
			index := strings.LastIndex(newDir, sep)
			if index == -1 {
				// should never happen to panic basically
				continue
			}
			newDir = newDir[:index]
			//newDir := filepath.Dir(mainvideo) + sep + ".." + sep
			mf, _ := os.Stat(mainvideo)
			NewFileName := newDir + sep + mf.Name()
			if testmode {

				log.Println("Moved File,", mainvideo, ", to,", NewFileName)

			} else {
				os.Rename(mainvideo, NewFileName)
				os.RemoveAll(n)
			}
			// lets now change the fito be this new file
			n = NewFileName
			fi, err = os.Stat(NewFileName)
			if err != nil {
				log.Println("Error", err)
				continue
			}

		}

		ext := filepath.Ext(fi.Name())

		// if it is not a video
		if !isExtAVideo(ext) && !isExtIgnoreListed(ext) {
			if testmode {
				log.Println("Removing non video file:", n)
			} else {
				os.Remove(n)
			}
			continue
			// if it is a video
		}
		// lets get the current Show Name and Season number so we can properly determine format
		showName := ""
		seasonNumber := ""
		for k, v := range strings.Split(path, sep) {
			if k == 0 {
				showName = v
			}
			if k == 1 {
				if strings.Contains(v, "Season ") || strings.Contains(v, "season ") {
					seasonNumber = v[7:]
					// this section is if we want to put a 0 infront if season is less than 10
					/*
						num, err := strconv.Atoi(seasonNumber)
						if err == nil {
							if num < 10 {
								seasonNumber = "0" + seasonNumber
							}
						}
					*/
				}
			}
		}
		// now we must workout what episode number we are on
		//sxxexx
		//SxxExx
		//xxx
		// ?i = case insensitive
		// () is a group
		// | is the alternate group
		// {} is ammount of previous token
		reg := regexp.MustCompile("(?i)(\\.s[0-9]{2}e[0-9]{2}\\.|(\\.[0-9]{3}\\.)| [0-9]{3}\\.)")
		if reg.Match([]byte(fi.Name())) {
			// we found a valid file.
			// lets now fetch this match point.

			showEpisode := reg.FindString(fi.Name())

			showEpisode = showEpisode[len(showEpisode)-3 : len(showEpisode)-1]
			idealName := showName + " " + seasonNumber + showEpisode + ext
			if fi.Name() != idealName {
				if testmode {
					log.Println("Renaming File:", fi.Name(), "to", idealName)
				} else {
					desiredLoc := path + sep + idealName
					os.Rename(n, desiredLoc)
				}
			}
		}
	}
	wgs.Done()
}
