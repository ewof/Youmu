package main

import (
	"encoding/json"
	"io/ioutil"
	"math/rand"
	"net/http"
	"time"
)

type Posts struct {
	Posts []Post `json:"post"`
}

type Post struct {
	ID            int    `json:"id"`
	CreatedAt     string `json:"created_at"`
	Score         int    `json:"score"`
	Width         int    `json:"width"`
	Height        int    `json:"height"`
	Md5           string `json:"md5"`
	Directory     string `json:"directory"`
	Image         string `json:"image"`
	Rating        string `json:"rating"`
	Source        string `json:"source"`
	Change        int    `json:"change"`
	Owner         string `json:"owner"`
	CreatorID     int    `json:"creator_id"`
	ParentID      int    `json:"parent_id"`
	Sample        int    `json:"sample"`
	PreviewHeight int    `json:"preview_height"`
	PreviewWidth  int    `json:"preview_width"`
	Tags          string `json:"tags"`
	Title         string `json:"title"`
	HasNotes      string `json:"has_notes"`
	HasComments   string `json:"has_comments"`
	FileURL       string `json:"file_url"`
	PreviewURL    string `json:"preview_url"`
	SampleURL     string `json:"sample_url"`
	SampleHeight  int    `json:"sample_height"`
	SampleWidth   int    `json:"sample_width"`
	Status        string `json:"status"`
	PostLocked    int    `json:"post_locked"`
	HasChildren   string `json:"has_children"`
}

// Gets a random post from gelbooru with specified tags, only looks for posts with "safe" rating is NSFW is false
func Gelbooru(tags string, nsfw bool) (post Post, found bool, err error) {
	found = true
	url := "https://gelbooru.com/index.php?page=dapi&s=post&q=index&limit=100&json=1&tags=" + tags
	client := http.Client{
		Timeout: time.Second * 2,
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)

	req.Header.Set("User-Agent", "Youmu")

	res, err := client.Do(req)

	if res.Body != nil {
		defer res.Body.Close()
	}

	body, err := ioutil.ReadAll(res.Body)

	posts := Posts{}
	err = json.Unmarshal(body, &posts)

	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)

	if len(posts.Posts) <= 0 {
		found = false
		return
	}

	Posts := posts.Posts

	postsLen := len(Posts)
	post = Posts[r1.Intn(postsLen)]
	if post.Rating != "safe" && !nsfw {
		count := 0
		for post.Rating != "safe" {
			if count > (postsLen - 1) {
				found = false
				break
			}
			s1 := rand.NewSource(time.Now().UnixNano())
			r1 := rand.New(s1)
			post = Posts[r1.Intn(postsLen)]
			if post.Rating == "safe" {
				break
			}
			count++
		}
	}
	return
}
