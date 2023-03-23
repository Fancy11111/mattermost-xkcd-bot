package xkcd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/rs/zerolog/log"
)

const xkcdUrlPrefix = "https://xkcd.com/"
const xkcdUrlPostfix = "info.0.json"
const xkcdUrlDaily = xkcdUrlPrefix + xkcdUrlPostfix

type XKCDPost struct {
	Month     string
	Year      string
	Day       string
	Number    int32  `json:"num"`
	SafeTitle string `json:"safe_title"`
	Title     string
	Image     string `json:"img"`
	AltText   string
}

func GetDailyPost() (*XKCDPost, error) {
	return sendXKCDRequest(xkcdUrlDaily)
}

func GetPost(number int64) (*XKCDPost, error) {
	return sendXKCDRequest(fmt.Sprintf("%s%d/%s", xkcdUrlPrefix, number, xkcdUrlPostfix))
}

func sendXKCDRequest(url string) (*XKCDPost, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Error().Err(err).Str("url", url).Msg("Could not create request")
		return nil, err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Error().Err(err).Str("url", url).Msg("Could not send request")
		return nil, err
	}

	log.Debug().Int("status_code", res.StatusCode).Msg("client: got response!")
	if res.StatusCode == 404 {
		err = fmt.Errorf("Could not load Post")
		log.Error().Err(err).Str("url", url).Msg("post not found")
		return nil, err
	}
	defer res.Body.Close()
	resBody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Error().Err(err).Str("url", url).Msg("Could not read body")
		return nil, err
	}

	post := new(XKCDPost)
	if err = json.Unmarshal(resBody, post); err != nil {
		log.Error().Err(err).Msg("Could not unmarshal request body")
		return nil, err
	}

	return post, nil
}
