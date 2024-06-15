package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

const (
	joplinBaseURL = "http://localhost:41184"
)

type PocketArticle struct {
	ItemID string `json:"item_id"`
	Title  string `json:"resolved_title"`
	URL    string `json:"resolved_url"`
}

type PocketResponse struct {
	List map[string]PocketArticle `json:"list"`
}

type JoplinTag struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type JoplinFolder struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

func main() {
	pocketConsumerKey := os.Getenv("POCKET_CONSUMER_KEY")
	pocketAccessToken := os.Getenv("POCKET_ACCESS_TOKEN")
	joplinToken := os.Getenv("JOPLIN_TOKEN")

	if pocketConsumerKey == "" || pocketAccessToken == "" {
		fmt.Println("POCKET_CONSUMER_KEY and POCKET_ACCESS_TOKEN environment variables are required")
		return
	}

	articles, err := fetchUnreadArticles(pocketConsumerKey, pocketAccessToken)
	if err != nil {
		fmt.Println("Error fetching articles from Pocket:", err)
		return
	}

	tagID, err := getOrCreateToReadTag(joplinToken)
	if err != nil {
		fmt.Println("Error getting or creating 'to_read' tag in Joplin:", err)
		return
	}

	folderID, err := getOrCreateMainFolder(joplinToken)
	if err != nil {
		fmt.Println("Error getting or creating 'Main' folder in Joplin:", err)
		return
	}

	for _, article := range articles {
		err = createJoplinNoteForArticle(tagID, folderID, joplinToken, article)
		if err != nil {
			fmt.Println("Error creating note in Joplin:", err)
		}
	}

	fmt.Println("All articles have been processed.")
}

func fetchUnreadArticles(consumerKey, accessToken string) ([]PocketArticle, error) {
	resp, err := http.Get(
		fmt.Sprintf("https://getpocket.com/v3/get?consumer_key=%s&access_token=%s&state=unread&detailType=simple",
			consumerKey,
			accessToken,
		),
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch articles, status code: %d", resp.StatusCode)
	}

	var pocketResp PocketResponse
	if err = json.NewDecoder(resp.Body).Decode(&pocketResp); err != nil {
		return nil, err
	}

	articles := make([]PocketArticle, 0, len(pocketResp.List))
	for _, article := range pocketResp.List {
		articles = append(articles, article)
	}

	return articles, nil
}

func getOrCreateToReadTag(token string) (string, error) {
	tags, err := fetchJoplinTags(token)
	if err != nil {
		return "", err
	}

	for _, tag := range tags {
		if tag.Title == "to_read" {
			return tag.ID, nil
		}
	}

	return createJoplinTag("to_read", token)
}

func fetchJoplinTags(token string) ([]JoplinTag, error) {
	resp, err := http.Get(joplinBaseURL + "/tags?token=" + token)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch tags, status code: %d", resp.StatusCode)
	}

	var respStruct struct {
		Tags []JoplinTag `json:"items"`
	}
	if err = json.NewDecoder(resp.Body).Decode(&respStruct); err != nil {
		return nil, err
	}

	return respStruct.Tags, nil
}

func createJoplinTag(title, token string) (string, error) {
	tag := JoplinTag{Title: title}
	body, err := json.Marshal(tag)
	if err != nil {
		return "", err
	}

	resp, err := http.Post(joplinBaseURL+"/tags?toke="+token, "application/json", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to create tag, status code: %d", resp.StatusCode)
	}

	var createdTag JoplinTag
	if err = json.NewDecoder(resp.Body).Decode(&createdTag); err != nil {
		return "", err
	}

	return createdTag.ID, nil
}

func getOrCreateMainFolder(token string) (string, error) {
	folders, err := fetchJoplinFolders(token)
	if err != nil {
		return "", err
	}

	for _, folder := range folders {
		if folder.Title == "Main" {
			return folder.ID, nil
		}
	}

	return createJoplinFolder("Main", token)
}

func fetchJoplinFolders(token string) ([]JoplinFolder, error) {
	resp, err := http.Get(joplinBaseURL + "/folders?token=" + token)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch folders, status code: %d", resp.StatusCode)
	}

	var respStruct struct {
		Folders []JoplinFolder `json:"items"`
	}
	if err = json.NewDecoder(resp.Body).Decode(&respStruct); err != nil {
		return nil, err
	}

	return respStruct.Folders, nil
}

func createJoplinFolder(title, token string) (string, error) {
	folder := JoplinFolder{Title: title}
	body, err := json.Marshal(folder)
	if err != nil {
		return "", err
	}

	resp, err := http.Post(joplinBaseURL+"/folders?token="+token, "application/json", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to create folder, status code: %d", resp.StatusCode)
	}

	var createdFolder JoplinFolder
	if err = json.NewDecoder(resp.Body).Decode(&createdFolder); err != nil {
		return "", err
	}

	return createdFolder.ID, nil
}

func createJoplinNoteForArticle(tagID, parentID, token string, article PocketArticle) error {
	note := map[string]string{
		"title":     article.Title,
		"body":      article.URL,
		"parent_id": parentID,
	}
	body, err := json.Marshal(note)
	if err != nil {
		return err
	}

	resp, err := http.Post(joplinBaseURL+"/notes?token="+token, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to create note, status code: %d", resp.StatusCode)
	}

	tagNoteURL := fmt.Sprintf("%s/tags/%s/notes?token="+token, joplinBaseURL, tagID)
	req, err := http.NewRequest(http.MethodPost, tagNoteURL, resp.Body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err = client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to tag note, status code: %d", resp.StatusCode)
	}

	return nil
}
