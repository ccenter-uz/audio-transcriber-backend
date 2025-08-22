package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"

	_ "github.com/lib/pq"
)

type Transcript struct {
	ID             int
	TranscribeText string
}

type APIResponse struct {
	Original    string `json:"original"`
	Transformed string `json:"transformed"`
}

func main() {
	// DB connection
	connStr := "host=10.145.20.8 port=5432 user=postgres password=mirxonjon dbname=voice_transcribe sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("DB connect error:", err)
	}
	defer db.Close()

	// Select all transcripts with text
	rows, err := db.Query(`SELECT id, transcribe_text FROM transcripts WHERE transcribe_text IS NOT NULL`)
	if err != nil {
		log.Fatal("Query error:", err)
	}
	defer rows.Close()

	for rows.Next() {
		var t Transcript
		if err := rows.Scan(&t.ID, &t.TranscribeText); err != nil {
			log.Println("Row scan error:", err)
			continue
		}

		// Call API
		apiURL := fmt.Sprintf("http://192.168.31.27:9512/text-transformer?req_text=%s",
			url.QueryEscape(t.TranscribeText))

		resp, err := http.Post(apiURL, "application/json", nil)
		if err != nil {
			log.Println("API error:", err)
			continue
		}
		defer resp.Body.Close()

		body, _ := ioutil.ReadAll(resp.Body)

		var apiResp APIResponse
		if err := json.Unmarshal(body, &apiResp); err != nil {
			log.Println("JSON parse error:", err, string(body))
			continue
		}

		// // Update DB
		// _, err = db.Exec(`UPDATE transcripts SET transcribe_text_normalized = $1, updated_at = NOW() WHERE id = $2`,
		// 	apiResp.Transformed, t.ID)
		// if err != nil {
		// 	log.Println("Update error:", err)
		// 	continue
		// }

		log.Printf("Updated transcript %d: %s → %s\n", t.ID, t.TranscribeText, apiResp.Transformed)
	}

	log.Println("Done.")
}
