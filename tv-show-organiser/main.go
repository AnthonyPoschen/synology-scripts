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
	tvshows = make([]string, 0, 20)
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
	showpath := strings.Split(path, sep)[0]
	// short hand season identifier for this parse. i.e 'S01'
	shSeason := pathToSeasonString(path)

	if err != nil {
		log.Println("Failed to search path in parse season with path:", path+"/*")
	}
	for _, n := range f {
		fi, err := os.Stat(n)
		if err != nil {
			continue
		}
		if fi.Mode().IsDir() {
			// some weird directory in snyology folders
			if strings.Contains(fi.Name(), "@eaDir") {
				continue
			}
			// move it back to the show directory to be handled. sinc
			if testmode {
				log.Println(testtext+"Move Folder:", n, "| To new folder |", showpath+sep+fi.Name())
			} else {
				log.Println("Move Folder:", n, "| To new folder |", showpath+sep+fi.Name())
				os.Rename(n, showpath+sep+fi.Name())
			}
		} else {
			ext := filepath.Ext(fi.Name())
			if !isExtAVideo(ext) && !isExtIgnoreListed(ext) {
				// if its not a video mark for deletion
				if testmode {
					log.Println("Removing non video file:", n)
				} else {
					os.Remove(n)
				}
			} else {
				// since it is a video lets check it follows the naming Convention
				videoName := fi.Name()

				// is video in correct season folder
				if !regexp.MustCompile("(?i).*" + shSeason + ".*").Match([]byte(videoName)) {
					// check if file format is presented correctly and we are just in the wrong folder
					if regexp.MustCompile("(?i).*S[0-9][0-9].*").Match([]byte(videoName)) {
						// ok so we are definetly in the wrong season folder so lets move this file back to the show folder.
						// to get organised later.
						if testmode {
							log.Println(testtext+"Video in wrong Season Folder Moved up a dir:", n)
						} else {
							log.Println(testtext+"Video in wrong Season Folder Moved up a dir:", n)
							os.Rename(n, showpath+sep+videoName)
						}
						// since the file is no longer in the folder lets move on
						continue
					}
					// ok so the file is using a different format then SxxExx so lets see if it starts with a number
					// TODO: the above shit
					// for now just skip files like this since can always manually change. sometimues u forget this is all still inside a for loop jesus
					// cant wait to refactor this bullshit big loop once its working.
					log.Println("Cant work out Season or Episode(add S00E00):", showpath+sep+videoName)
					continue
				}
				// presuming we have a file that is in the correct season folder.
				// lets get the episode number and see if it is already correctly formated i.e 'ShowName.SxxExx'
				videoNameNoExt := strings.Replace(videoName, ext, "", 1)
				if !regexp.MustCompile("(?i)^" + showpath + ".S[0-9][0-9]E[0-9][0-9]$").Match([]byte(videoNameNoExt)) {
					// so the file name is 100% match. lets grab the episode number and work out what it should be labeled.
					// we want the first match
					match := regexp.MustCompile("(?i)(E[0-9][0-9])").FindStringSubmatch(videoName)[0]
					if len(match) == 3 {
						actualname := showpath + "." + shSeason + match + ext

						// err will be returned if it doesn't exsist. which means we can just rename
						file1, err := os.Stat(path + sep + actualname)
						if err != nil {
							// if we cant find a file by that name and type we are good to go.
							log.Println("File Doesnt exsist cool to rename,", path+sep+videoName, "TO", path+sep+actualname)
							os.Rename(path+sep+videoName, path+sep+actualname)
						} else {
							// if one does already exist time to check file sizes. bigger is always better right ;)
							file2, _ := os.Stat(path + sep + videoName)

							if file2.Size() > file1.Size() {
								if testmode {
									log.Println(testtext+"Replacing:", path+sep+actualname, "With:", path+sep+videoName)
								} else {
									log.Println("Replacing:", path+sep+actualname, "With:", path+sep+videoName)
									os.Remove(path + sep + actualname)
									os.Rename(path+sep+videoName, path+sep+actualname)
								}

							} else {
								if testmode {
									log.Println(testtext+"Removing File:", path+sep+videoName)
								} else {
									log.Println("Removing File:", path+sep+videoName)
									os.Remove(path + sep + videoName)
								}

							}
						}
					}

				}
				// do stuff if name is already perfect maybe?
			}
		}
	}
	wgs.Done()
}
