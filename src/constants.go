package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path"
	"sort"
	"strconv"

	"github.com/Zamiell/isaac-racing-server/src/log"
)

// For holding the values of the "items.json" file
type JSONItem struct {
	Name        string  `json:"name"`
	Damage      float32 `json:"dmg"`
	DamageX     float32 `json:"dmg_x"`
	Health      int     `json:"health"`
	SoulHearts  int     `json:"soul_hearts"`
	SinHearts   int     `json:"sin_hearts"`
	Tears       float32 `json:"tears"`
	Delay       float32 `json:"delay"`
	DelayX      float32 `json:"delay_x`
	Speed       float32 `json:"speed"`
	ShotSpeed   float32 `json:"shot_speed"`
	Height      float32 `json:"height"`
	Range       float32 `json:"range"`
	Luck        float32 `json:"luck"`
	Beelzebub   bool    `json:"beelzebub`
	Bob         bool    `json:"bob"`
	Bookworm    bool    `json:"bookworm"`
	Conjoined   bool    `json:"conjoined"`
	Funguy      bool    `json:"funguy"`
	Guppy       bool    `json:"guppy"`
	Leviathan   bool    `json:"leviathan"`
	OhCrap      bool    `json:"ohcrap"`
	Seraphim    bool    `json:"seraphim"`
	SpiderBaby  bool    `json:"spiderbaby"`
	Spun        bool    `json:"spun"`
	Superbum    bool    `json:"superbum"`
	YesMother   bool    `json:"yesmother"`
	SpaceBar    bool    `json:"space"`
	HealthOnly  bool    `json:"health_only"`
	Intro       string  `json:"introduced_in"`
	Shown       bool    `json:"shown"`
	Summary     bool    `json:"in_summary"`
	SummaryName string  `json:"summary_name"`
	SummaryCond string  `json:"condition_name"`
	Text        string  `json:"text"`
}

// For holding the values oi the "builds.json" file
type IsaacItem struct {
	ID   int
	Name string
}

type TournamentInfo struct {
	Name         string `json:"name"`
	ChallongeID  string `json:"challonge_id"`
	ChallongeURL string `json:"challonge"`
	Date         string `json:"date"`
	Notability   string `json:"notability"`
	Organizer    string `json:"organizer"`
	Ruleset      string `json:"ruleset"`
	Description  string `json:"description"`
}

type NameSorter []os.FileInfo

func (a NameSorter) Len() int           { return len(a) }
func (a NameSorter) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a NameSorter) Less(i, j int) bool { return a[i].Name() > a[j].Name() }

var (
	seededBuilds = []string{
		"20/20",                     // 1
		"Chocolate Milk",            // 2
		"Cricket's Body",            // 3
		"Cricket's Head",            // 4
		"Dead Eye",                  // 5
		"Death's Touch",             // 6
		"Dr. Fetus",                 // 7
		"Epic Fetus",                // 8
		"Ipecac",                    // 9
		"Jacob's Ladder",            // 10
		"Judas' Shadow",             // 11
		"Lil' Brimstone",            // 12
		"Magic Mushroom",            // 13
		"Mom's Knife",               // 14
		"Monstro's Lung",            // 15
		"Polyphemus",                // 16
		"Proptosis",                 // 17
		"Sacrificial Dagger",        // 18
		"Tech.5",                    // 19
		"Tech X",                    // 20
		"Brimstone",                 // 21
		"Incubus",                   // 22
		"Maw of the Void",           // 23
		"Crown of Light",            // 24
		"Godhead",                   // 25
		"Sacred Heart",              // 26
		"Mutant Spider + Inner Eye", // 27
		"Technology + Coal",         // 28
		"Ludovico + Parasite",       // 29
		"Fire Mind + 13 luck",       // 30
		"Tech Zero + more",          // 31
		"Kamikaze! + Host Hat",      // 32
		"Mega Blast + more",         // 33
	}

	allItems       = make(map[string]*JSONItem)
	allItemNames   = make(map[int]string)
	allBuilds      = make([]IsaacItem, 0)
	allTournaments = make([]TournamentInfo, 0)
)

func loadAllItems() {
	jsonFilePath := path.Join(projectPath, "public", "items.json")
	jsonFile, err := ioutil.ReadFile(jsonFilePath)
	if err != nil {
		log.Fatal("Failed to open \""+jsonFilePath+"\":", err)
	}

	// Create all the items
	json.Unmarshal(jsonFile, &allItems)

	// Create 2nd map of just item names
	for k, v := range allItems {
		itemid, _ := strconv.Atoi(k)
		allItemNames[itemid] = v.Name
	}
}

func loadAllBuilds() {
	// Open the JSON file and verify it was good
	jsonFilePath := path.Join(projectPath, "public", "builds.json")
	if jsonFile, err := ioutil.ReadFile(jsonFilePath); err != nil {
		log.Fatal("Failed to open \""+jsonFilePath+"\":", err)
	} else {
		// Create all the items
		json.Unmarshal(jsonFile, &allBuilds)
	}
}

func loadAllTournaments() {

	// Temporary var for each tournament
	var tournament TournamentInfo
	// Open the JSON files for tournaments and load them into TournamentInfo
	jsonFolderPath := path.Join(projectPath, "BoIR-trueskill/tournaments")
	fileList, err := ioutil.ReadDir(jsonFolderPath)
	if err != nil {
		log.Error("Could not read the files in ", jsonFolderPath)
	}
	log.Info(fileList[0].Name())
	sort.Sort(NameSorter(fileList))
	log.Info(fileList[0].Name())
	for _, file := range fileList {
		// Create the full file path
		filePath := path.Join(jsonFolderPath, file.Name())
		if jsonFile, err := ioutil.ReadFile(filePath); err != nil {
			// Fatal error if we cannot open a file
			log.Fatal("Failed to open \""+filePath+"\":", err)
		} else {
			// Create all the tournament vars
			json.Unmarshal(jsonFile, &tournament)
			allTournaments = append(allTournaments, tournament)
		}
	}
}
