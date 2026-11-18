package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	chromago "github.com/amikos-tech/chroma-go"
	chromaai "github.com/amikos-tech/chroma-go/openai"
	"github.com/amikos-tech/chroma-go/types"
	"github.com/gocarina/gocsv"
	"github.com/sashabaranov/go-openai"
)

type Database struct {
	Course               *chromago.Client
	CourseCollection     *chromago.Collection
	InstructorCollection *chromago.Collection
	SubjCollection       *chromago.Collection
}

type Course struct {
	Subject             string `csv:"SUBJ" json:"Subject"`
	CourseNum           string `csv:"CRSE NUM" json:"CourseNum"`
	Section             string `csv:"SEC" json:"Section"`
	CRN                 string `csv:"CRN" json:"CRN"`
	ScheduleType        string `csv:"Schedule Type Code" json:"ScheduleType"`
	Title               string `csv:"Title Short Desc" json:"Title"`
	InstructionMode     string `csv:"Instruction Mode Dec" json:"InstructionMode"`
	MeetingType         string `csv:"-" json:"MeetingType"`
	MeetDays            string `csv:"Meet Days" json:"MeetDays"`
	BeginTime           string `csv:"Begin Time" json:"BeginTime"`
	EndTime             string `csv:"End Time" json:"EndTime"`
	MeetStart           string `csv:"Meet Start" json:"MeetStart"`
	MeetEnd             string `csv:"Meet End" json:"MeetEnd"`
	Building            string `csv:"BLDG" json:"Building"`
	Room                string `csv:"RM" json:"Room"`
	ActualEnrollment    string `csv:"-" json:"ActualEnrollment"`
	InstructorFirstName string `csv:"Primary Instructor First Name" json:"InstructorFirstName"`
	InstructorLastName  string `csv:"Primary Instructor Last Name" json:"InstructorLastName"`
	InstructorEmail     string `csv:"Primary Instructor Email" json:"InstructorEmail"`
}

func (db *Database) Import(filename string) error {

	err := db.Reset()
	if err != nil {
		return fmt.Errorf("failed to reset database: %w", err)
	}

	csvFile, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open CSV file: %w", err)
	}
	defer csvFile.Close()

	var courses []*Course
	if err := gocsv.Unmarshal(csvFile, &courses); err != nil {
		return err
	}

	ctx := context.Background()

	batch := 500

	for i := 0; i < len(courses); i += batch {
		end := i + batch
		if end > len(courses) {
			end = len(courses)
		}

		courseTitleMap := make(map[string]bool)
		instructorsMap := make(map[string]bool)
		subjectsMap := make(map[string]bool)

		courseRS, err := types.NewRecordSet(
			types.WithEmbeddingFunction(db.CourseCollection.EmbeddingFunction),
			types.WithIDGenerator(types.NewULIDGenerator()),
		)
		if err != nil {
			return fmt.Errorf("error creating record set: %w", err)
		}

		for _, course := range courses[i:end] {

			jsonStr, err := json.Marshal(course)
			if err != nil {
				return fmt.Errorf("error converting client to JSON: %w", err)
			}

			instrName := course.InstructorFirstName + " " + course.InstructorLastName
			subj := course.Subject
			title := course.Title

			if instrName != "" {
				instructorsMap[instrName] = true
			}

			if subj != "" {
				subjectsMap[subj] = true
			}

			if title != "" {
				courseTitleMap[title] = true
			}

			courseRS.WithRecord(
				types.WithDocument(string(jsonStr)),
				types.WithMetadata("instructor", instrName),
				types.WithMetadata("subject", subj),
				types.WithMetadata("title", title),
			)
		}

		_, err = courseRS.BuildAndValidate(ctx)
		if err != nil {
			return fmt.Errorf("error validating record set: %w", err)
		}

		_, err = db.CourseCollection.AddRecords(ctx, courseRS)
		if err != nil {
			return fmt.Errorf("error adding crs records")
		}

		if len(instructorsMap) > 0 {
			instrRS, err := types.NewRecordSet(
				types.WithEmbeddingFunction(db.InstructorCollection.EmbeddingFunction),
				types.WithIDGenerator(types.NewULIDGenerator()),
			)
			if err != nil {
				fmt.Printf("error creating record set: %v", err)
				continue
			}
			for instr := range instructorsMap {
				instrRS.WithRecord(types.WithDocument(instr))
			}

			_, err = instrRS.BuildAndValidate(ctx)
			if err != nil {
				return fmt.Errorf("error validating record set: %w", err)
			}

			_, err = db.InstructorCollection.AddRecords(ctx, instrRS)
			if err != nil {
				return fmt.Errorf("error adding records instr")
			}

		}

		if len(subjectsMap) > 0 {
			subjRS, err := types.NewRecordSet(
				types.WithEmbeddingFunction(db.SubjCollection.EmbeddingFunction),
				types.WithIDGenerator(types.NewULIDGenerator()),
			)
			if err != nil {
				return fmt.Errorf("error creating record set: %w", err)
			}

			for subj := range subjectsMap {

				subjRS.WithRecord(types.WithDocument(subj))
			}

			_, err = subjRS.BuildAndValidate(ctx)
			if err != nil {
				return fmt.Errorf("error validating record set: %w", err)
			}

			_, err = db.SubjCollection.AddRecords(ctx, subjRS)
			if err != nil {
				fmt.Printf("error here for subject coll")
				return fmt.Errorf("error adding records subj")
			}

		}

	}

	return nil
}

func setupEmbedding() (types.EmbeddingFunction, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("No API key found. Please set the OPENAI_API_KEY environment variable!")
		return nil, fmt.Errorf("no API key found. Please set the OPENAI_API_KEY environment variable")
	}
	openaiEF, err := chromaai.NewOpenAIEmbeddingFunction(apiKey)
	if err != nil {
		return nil, fmt.Errorf("error creating embedding function: %w", err)
	}

	return openaiEF, nil
}

func NewDatabase(reset bool) (*Database, error) {
	client, err := chromago.NewClient("http://localhost:8000")
	if err != nil {
		return nil, err
	}

	efunc, err := setupEmbedding()
	if err != nil {
		return nil, fmt.Errorf("failed to setup embedding function: %w", err)
	}

	ctx := context.Background()

	db := &Database{
		Course: client,
	}

	if reset {
		if err := db.Reset(); err != nil {
			return nil, fmt.Errorf("error resetting database: %w", err)
		}
	}

	courseCollection, err := client.GetCollection(ctx, "course-collection", efunc)
	if err != nil {
		// If the collection doesn't exist, create a new one
		courseCollection, err = client.CreateCollection(ctx, "course-collection", nil, true, efunc, types.L2)
		if err != nil {
			return nil, fmt.Errorf("failed to create collection: %w", err)
		}
	}

	instrCollection, err := client.GetCollection(ctx, "instructors", efunc)
	if err != nil {
		instrCollection, err = client.CreateCollection(ctx, "instructors", nil, true, efunc, types.L2)
		if err != nil {
			return nil, fmt.Errorf("failed to create instructor collection: %w", err)
		}
	}

	subjCollection, err := client.GetCollection(ctx, "subjects", efunc)
	if err != nil {
		subjCollection, err = client.CreateCollection(ctx, "subjects", nil, true, efunc, types.L2)
		if err != nil {
			return nil, fmt.Errorf("failed to create subject collection: %w", err)
		}
	}

	return &Database{
		client,
		courseCollection,
		instrCollection,
		subjCollection,
	}, nil
}

func (db *Database) ExtractNameEntities(question string) (map[string]string, error) {

	client := openai.NewClient(os.Getenv("OPENAI_API_KEY"))

	prompt := fmt.Sprintf(`Please answer the question based on the provided information. If the question is vague or missing details, infer as much information as possible based on the question:
		Extract the following:
		1.Instructor full name (i.e. John Smith or "unknown")
		2.Season (i.e Fall, Spring, or "unknown")
		3.Year (i.e. 2024 or "unknown")
		4.Subject (i.e. MU for Music, PHIL for Philosophy or "unknown")
		5. Key term (most important word related to the course content (i.e. "guitar" or "unknown"))

		Question: %s
		Respond with a JSON object only, without any additional text or formatting (ensure all values are strings in JSON):
		{
			"instructor": "extracted name or unknown",
			"season": "extracted season or unknown",
			"year": "extracted year or unknown",
			"subject": "extracted subject or unknown",
			"keyTerm": "single most extracted important term or unknown"
		}`, question)

	resp, err := client.CreateChatCompletion(context.Background(), openai.ChatCompletionRequest{
		Model: openai.GPT4oMini,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: "You are a helpful assistant that extracts information and answers questions about university courses based on that information"},
			{Role: openai.ChatMessageRoleUser, Content: prompt},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("error calling OpenAI: %w", err)
	}

	var extractedData map[string]string
	if err := json.Unmarshal([]byte(resp.Choices[0].Message.Content), &extractedData); err != nil {
		return nil, fmt.Errorf("error parsing extracted JSON: %w", err)
	}

	return extractedData, nil
}

func (db *Database) VectorQuery(query string) ([]string, error) {

	ctx := context.Background()

	extractedData, err := db.ExtractNameEntities(query)
	if err != nil {
		return nil, fmt.Errorf("error extracting name entities: %w", err)
	}

	m := map[string]interface{}{}

	fuzzyName := extractedData["instructor"]
	subj := extractedData["subject"]

	if extractedData["instructor"] != "" && extractedData["instructor"] != "unknown" {
		instr, err := db.InstructorCollection.Query(ctx, []string{fuzzyName}, 5, nil, nil, nil)
		if err == nil && len(instr.Documents) > 0 && len(instr.Documents[0]) > 0 {
			m["instructor"] = instr.Documents[0][0]
		}
	}

	if extractedData["subject"] != "" && extractedData["subject"] != "unknown" {
		subject, err := db.SubjCollection.Query(ctx, []string{subj}, 5, nil, nil, nil)
		if err == nil && len(subject.Documents) > 0 && len(subject.Documents[0]) > 0 {
			m["subject"] = subject.Documents[0][0]
		}
	}

	searchQuery := extractedData["keyTerm"]
	if searchQuery == "" || searchQuery == "unknown" {
		searchQuery = query
	}

	if len(m) > 0 {
		results, err := db.CourseCollection.Query(ctx, []string{searchQuery}, 10, m, nil, nil)
		if err != nil {
			return nil, fmt.Errorf("error query collection crs: %w", err)
		}

		if len(results.Documents) == 0 || len(results.Documents[0]) == 0 {
			return nil, fmt.Errorf("no results found")
		}
		return results.Documents[0], nil

	}

	results, err := db.CourseCollection.Query(ctx, []string{searchQuery}, 10, nil, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("error querying collection crs: %w", err)
	}

	if len(results.Documents) == 0 || len(results.Documents[0]) == 0 {
		return nil, fmt.Errorf("no results found")
	}

	courses := []string{}
	for _, document := range results.Documents[0] {
		courses = append(courses, document)
	}

	return courses, nil
}

func (db *Database) Reset() error {
	ctx := context.Background()

	if db.Course == nil {
		return fmt.Errorf("database client is not initialized")
	}

	embeddingFunc, err := chromaai.NewOpenAIEmbeddingFunction(
		os.Getenv("OPENAI_API_KEY"),
	)
	if err != nil {
		return fmt.Errorf("error creating embedding function: %w", err)
	}

	collections := []struct {
		name   string
		target **chromago.Collection
	}{
		{"course-collection", &db.CourseCollection},
		{"instructors", &db.InstructorCollection},
		{"subjects", &db.SubjCollection},
	}

	for _, item := range collections {
		if _, err := db.Course.DeleteCollection(ctx, item.name); err != nil {
			log.Printf("Collection %s could not be deleted or did not exist: %v", item.name, err)
		}

		collection, err := db.Course.CreateCollection(
			ctx,
			item.name,
			nil,
			true,
			embeddingFunc,
			types.L2,
		)
		if err != nil {
			return fmt.Errorf(
				"error recreating collection %s: %w",
				item.name,
				err,
			)
		}

		*item.target = collection
		log.Printf("Collection %s reset successfully.", item.name)
	}

	return nil

}
