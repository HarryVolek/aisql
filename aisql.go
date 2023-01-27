package main

import (
	"bufio"
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/shomali11/xsql"
)

const tableDump = `select table_name, column_name, data_type
from INFORMATION_SCHEMA.COLUMNS
where table_schema != 'information_schema' and table_schema != 'pg_catalog';`

const model = "text-davinci-003"
const temperature = 0.5
const maxTokens = 250

const openaiCompletionUrl = "https://api.openai.com/v1/completions"

var httpClient = http.Client{}
var authKey string

type SchemaField struct {
	Column   string
	Datatype string
}

type TableSchema []SchemaField

type DatabaseSchema map[string]TableSchema

type OpenAIRequestBody struct {
	Model       string  `json:"model"`
	Prompt      string  `json:"prompt"`
	Temperature float32 `json:"temperature"`
	MaxTokens   int     `json:"max_tokens"`
}

type OpenAIResponseBody struct {
	Choices []Choice
}

type Choice struct {
	Text string `json:"text"`
}

func main() {
	flag.StringVar(&authKey, "K", "", "openai api key")
	connString := flag.String("C", "", "postgres connection string")
	flag.Parse()
	if authKey == "" || *connString == "" {
		panic("Set auth key and connection string")
	}
	db, err := sql.Open("pgx", *connString)
	if err != nil {
		panic(err)
	}
	schema, err := getTableSchemas(db)
	if err != nil {
		panic(err)
	}
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("-> ")
		nlqRaw, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println(err)
			continue
		}
		naturalLanguageQuery := strings.TrimSuffix(nlqRaw, "\n")
		response, err := getCompletion(generatePrompt(schema, naturalLanguageQuery))
		if err != nil {
			fmt.Println(err)
			continue
		}
		fmt.Println("Response start=======")
		fmt.Println(response)
		fmt.Println("Response finish======")
		fmt.Println("Execute query? [y/N]")
		exeRaw, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println(err)
			continue
		}
		exe := strings.TrimSuffix(exeRaw, "\n")
		if exe != "y" {
			continue
		}
		rows, err := db.Query(response)
		if err != nil {
			log.Println(err)
			continue
		}
		results, err := xsql.Pretty(rows)
		if err != nil {
			log.Println(err)
			continue
		}
		fmt.Println(results)
	}
}

func getTableSchemas(db *sql.DB) (DatabaseSchema, error) {
	schema := DatabaseSchema{}
	rows, err := db.Query(tableDump)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var (
			tableName string
			field     SchemaField
		)
		if err := rows.Scan(&tableName, &field.Column, &field.Datatype); err != nil {
			panic(err)
		}
		if schema[tableName] == nil {
			schema[tableName] = make([]SchemaField, 0)
		}
		schema[tableName] = append(schema[tableName], field)
	}
	return schema, nil
}

func generateSchemaSummary(schema DatabaseSchema) string {
	var sb strings.Builder
	for table, fields := range schema {
		sb.WriteString(fmt.Sprintf("Schema for table: %s\n", table))
		for _, field := range fields {
			sb.WriteString(fmt.Sprintf("\t%s %s\n", field.Column, field.Datatype))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

func generatePrompt(schema DatabaseSchema, question string) string {
	prompt := fmt.Sprintf(`%s 
	As a senior analyst, given the above schemas, write a detailed and correct Postgres sql query to answer the analytical question:
	
	"%s"
	
	Comment the query with your logic`, generateSchemaSummary(schema), question)
	return prompt
}

func getCompletion(prompt string) (string, error) {
	body, err := json.Marshal(OpenAIRequestBody{
		Model:       model,
		Prompt:      prompt,
		Temperature: temperature,
		MaxTokens:   maxTokens,
	})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequest("POST", openaiCompletionUrl, bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authKey))
	res, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	bytes, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP status %d: %s", res.StatusCode, string(bytes))
	}
	responseBody := OpenAIResponseBody{}
	err = json.Unmarshal(bytes, &responseBody)
	if err != nil {
		return "", err
	}
	if len(responseBody.Choices) == 0 {
		return "", errors.New("no response returned")
	}
	return responseBody.Choices[0].Text, nil
}
