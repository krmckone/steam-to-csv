package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/dnlo/struct2csv"
)

type SteamOwnedGame struct {
	AppId                   int     `json:"appid"`
	Name                    string  `json:"name"`
	PlaytimeForever         float64 `json:"playtime_forever" csv:"Total Playtime"`
	ImgIconUrl              string  `json:"img_icon_url"`
	PlaytimeWindowsForever  float64 `json:"playtime_windows_forever"`
	PlaytimeMacForever      float64 `json:"playtime_mac_forever"`
	PlaytimeLinuxForever    float64 `json:"playtime_linux_forever"`
	PlaytimeDeckForever     float64 `json:"playtime_deck_forever"`
	RTimeLastPlayed         int64   `json:"rtime_last_played"`
	FormattedTimeLastPlayed string  `csv:"Last Time Played"`
	SteamUrl                string  `csv:"Steam URL"`
}

type SteamOwnedGamesResponse struct {
	Response struct {
		GamesCount int
		Games      []SteamOwnedGame
	}
}

func GetSteamOwnedGames() ([]SteamOwnedGame, error) {
	steam_api_key, present := os.LookupEnv("STEAM_API_KEY")
	if !present || steam_api_key == "" {
		return []SteamOwnedGame{}, fmt.Errorf("STEAM_API_KEY variable not present in env")
	}
	baseUrl, err := url.Parse("https://api.steampowered.com/IPlayerService/GetOwnedGames/v1/")
	if err != nil {
		return []SteamOwnedGame{}, err
	}
	params := url.Values{}
	params.Add("key", steam_api_key)
	params.Add("steamid", "76561197988460908") // me
	params.Add("include_appinfo", "true")
	params.Add("include_extended_appinfo", "true")
	params.Add("include_played_free_games", "true")
	params.Add("include_free_sub", "true")
	params.Add("skip_unvetted_apps", "true")
	baseUrl.RawQuery = params.Encode()
	resp, err := HttpGet(baseUrl.String())
	if err != nil {
		return []SteamOwnedGame{}, err
	}
	target := SteamOwnedGamesResponse{}
	ReadHttpRespBody(resp, &target)
	return target.Response.Games, nil
}

func ReadHttpRespBody(resp *http.Response, target interface{}) error {
	defer resp.Body.Close()
	err := json.NewDecoder(resp.Body).Decode(target)
	if err != nil {
		return fmt.Errorf("error in reading HTTP response body: %s", err)
	}
	return nil
}

// Invokes HTTP GET on the URL and returns the body as a string
func HttpGet(url string) (*http.Response, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected HTTP GET return code: %d", resp.StatusCode)
	}
	return resp, nil
}

func ProcessOwnedGames(games []SteamOwnedGame) error {
	for i := range games {
		hours := games[i].PlaytimeForever / 60.0
		truncated, err := truncateFloat(hours)
		if err != nil {
			return err
		}
		games[i].PlaytimeForever = truncated

		lastPlayed := time.Unix(games[i].RTimeLastPlayed, 0)
		if lastPlayed == time.Unix(0, 0) || truncated == 0 {
			games[i].FormattedTimeLastPlayed = "Never"
		} else {
			games[i].FormattedTimeLastPlayed = fmt.Sprintf("%s %d %d", lastPlayed.Month(), lastPlayed.Day(), lastPlayed.Year())
		}
		games[i].SteamUrl = fmt.Sprintf("https://store.steampowered.com/app/%d", games[i].AppId)
	}
	return nil
}

func truncateFloat(f float64) (float64, error) {
	t, err := strconv.ParseFloat(fmt.Sprintf("%.2f", f), 64)
	if err != nil {
		return t, err
	}
	return t, nil
}

func main() {
	games, err := GetSteamOwnedGames()
	if err != nil {
		panic(err)
	}
	if err := ProcessOwnedGames(games); err != nil {
		panic(err)
	}
	enc := struct2csv.New()
	rows, err := enc.Marshal(games)
	if err != nil {
		panic(err)
	}
	f, err := os.Create("output.csv")
	if err != nil {
		panic(err)
	}
	w := csv.NewWriter(f)
	err = w.WriteAll(rows)
	if err != nil {
		panic(err)
	}
}
