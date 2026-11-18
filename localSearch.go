package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

type LocalCourse struct {
	Subject         string `json:"subject"`
	CourseNum       string `json:"courseNum"`
	CRN             string `json:"crn"`
	Title           string `json:"title"`
	MeetingDays     string `json:"meetingDays"`
	BeginTime       string `json:"beginTime"`
	EndTime         string `json:"endTime"`
	Building        string `json:"building"`
	Room            string `json:"room"`
	Instructor      string `json:"instructor"`
	InstructorEmail string `json:"instructorEmail"`
	Enrollment      string `json:"enrollment"`
	InstructionMode string `json:"instructionMode"`
}

type CourseCatalog struct {
	Courses []LocalCourse
}

func LoadCourseCatalog(filename string) (*CourseCatalog, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("open course catalog: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1

	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("read course catalog: %w", err)
	}
	if len(records) < 2 {
		return nil, fmt.Errorf("course catalog contains no course records")
	}

	headers := make(map[string]int)
	for i, header := range records[0] {
		headers[strings.TrimSpace(header)] = i
	}

	value := func(record []string, column string) string {
		index, ok := headers[column]
		if !ok || index >= len(record) {
			return ""
		}
		return strings.TrimSpace(record[index])
	}

	catalog := &CourseCatalog{}
	for _, record := range records[1:] {
		first := value(record, "Primary Instructor First Name")
		last := value(record, "Primary Instructor Last Name")

		course := LocalCourse{
			Subject:         value(record, "SUBJ"),
			CourseNum:       value(record, "CRSE NUM"),
			CRN:             value(record, "CRN"),
			Title:           value(record, "Title Short Desc"),
			MeetingDays:     value(record, "Meet Days"),
			BeginTime:       value(record, "Begin Time"),
			EndTime:         value(record, "End Time"),
			Building:        value(record, "BLDG"),
			Room:            value(record, "RM"),
			Instructor:      strings.TrimSpace(first + " " + last),
			InstructorEmail: value(record, "Primary Instructor Email"),
			Enrollment:      value(record, "Actual Enrollment"),
			InstructionMode: value(record, "Instruction Mode Desc"),
		}

		if course.Subject == "" && course.Title == "" {
			continue
		}
		catalog.Courses = append(catalog.Courses, course)
	}

	if len(catalog.Courses) == 0 {
		return nil, fmt.Errorf("no valid courses found in catalog")
	}

	return catalog, nil
}

func (catalog *CourseCatalog) Search(query string) []LocalCourse {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return nil
	}

	// Remove common words that don't help identify a course.
	stopWords := map[string]bool{
		"a": true, "an": true, "the": true, "is": true,
		"are": true, "am": true, "do": true, "does": true,
		"did": true, "what": true, "which": true, "where": true,
		"when": true, "who": true, "how": true, "can": true,
		"could": true, "would": true, "me": true, "my": true,
		"i": true, "we": true, "you": true, "they": true,
		"please": true, "show": true, "find": true, "looking": true,
		"for": true, "to": true, "of": true, "in": true,
		"on": true, "at": true, "about": true, "some": true,
		"any": true, "this": true, "that": true, "it": true,
		"course": true, "courses": true, "class": true, "classes": true,
		"offered": true, "available": true, "teaching": true,
		"teach": true, "taught": true, "take": true, "learn": true, "learning": true,
		"want": true, "like": true,
	}

	// Normalize common subject and weekday references.
	aliases := map[string]string{
		"philosophy":    "phil",
		"philosophical": "phil",
		"monday":        "day_m",
		"mondays":       "day_m",
		"tuesday":       "day_t",
		"tuesdays":      "day_t",
		"wednesday":     "day_w",
		"wednesdays":    "day_w",
		"thursday":      "day_r",
		"thursdays":     "day_r",
		"friday":        "day_f",
		"fridays":       "day_f",
		"saturday":      "day_s",
		"saturdays":     "day_s",
		"sunday":        "day_u",
		"sundays":       "day_u",
	}

	words := strings.FieldsFunc(query, func(r rune) bool {
		return !(r >= 'a' && r <= 'z') &&
			!(r >= '0' && r <= '9')
	})

	var terms []string
	for _, word := range words {
		if stopWords[word] {
			continue
		}
		if alias, ok := aliases[word]; ok {
			word = alias
		}
		terms = append(terms, word)
	}

	if len(terms) == 0 {
		return nil
	}

	type scoredCourse struct {
		course LocalCourse
		score  int
	}

	var ranked []scoredCourse

	// Detect when the user is asking about an instructor.
	instructorQuery := strings.Contains(query, "instructor") ||
		strings.Contains(query, "teaching") ||
		strings.Contains(query, "teach") ||
		strings.Contains(query, "professor")

	for _, course := range catalog.Courses {
		score := 0

		subject := strings.ToLower(course.Subject)
		code := strings.ToLower(course.CourseCode())
		title := strings.ToLower(course.Title)
		instructor := strings.ToLower(course.Instructor)
		email := strings.ToLower(course.InstructorEmail)
		days := strings.ToUpper(course.MeetingDays)
		location := strings.ToLower(course.Building + " " + course.Room)
		instruction := strings.ToLower(course.InstructionMode)

		// Exact phrase matches receive a relevance bonus.
		fullText := strings.ToLower(strings.Join([]string{
			course.CourseCode(),
			course.Title,
			course.Instructor,
			course.InstructorEmail,
			course.MeetingDays,
			course.Building,
			course.Room,
			course.InstructionMode,
		}, " "))

		if strings.Contains(fullText, query) {
			score += 12
		}

		matchedTerms := 0

		for _, term := range terms {
			termScore := 0

			switch {
			case strings.HasPrefix(term, "day_"):
				dayCode := strings.TrimPrefix(term, "day_")
				if strings.Contains(days, strings.ToUpper(dayCode)) {
					termScore = 8
				}

			case term == "phil":
				// Only interpret "phil" as Philosophy when the question
				// is not clearly asking about an instructor.
				if !instructorQuery {
					if subject == "phil" {
						termScore = 12
					} else if strings.Contains(title, "philosoph") {
						termScore = 8
					}
				}

			case strings.Contains(instructor, term):
				termScore = 10

			case strings.Contains(email, term):
				termScore = 5

			case strings.Contains(title, term):
				termScore = 8

			case strings.Contains(subject, term):
				termScore = 8

			case strings.Contains(code, term):
				termScore = 7

			case strings.Contains(location, term):
				termScore = 5

			case strings.Contains(instruction, term):
				termScore = 3
			}

			if termScore > 0 {
				matchedTerms++
				score += termScore
			}
		}

		// Handle time-of-day questions using the catalog's 24-hour times.
		begin, err := strconv.Atoi(course.BeginTime)
		if err == nil {
			if strings.Contains(query, "morning") && begin >= 600 && begin < 1200 {
				score += 12
			}
			if (strings.Contains(query, "afternoon") || strings.Contains(query, "evening")) &&
				begin >= 1200 && begin < 2200 {
				score += 12
			}
			if strings.Contains(query, "evening") && begin >= 1700 {
				score += 12
			}
		}

		// Require at least one meaningful match.
		if matchedTerms > 0 || score >= 12 {
			ranked = append(ranked, scoredCourse{
				course: course,
				score:  score,
			})
		}
	}

	sort.SliceStable(ranked, func(i, j int) bool {
		return ranked[i].score > ranked[j].score
	})

	// Limit the number of results returned to the chat interface.
	const maxResults = 20
	if len(ranked) > maxResults {
		ranked = ranked[:maxResults]
	}

	results := make([]LocalCourse, 0, len(ranked))
	for _, item := range ranked {
		results = append(results, item.course)
	}

	return results
}

func (course LocalCourse) CourseCode() string {
	return course.Subject + " " + course.CourseNum
}

func (course LocalCourse) Location() string {
	if course.Building == "" && course.Room == "" {
		return "Location unavailable"
	}
	return strings.TrimSpace(course.Building + " " + course.Room)
}

func (course LocalCourse) EnrollmentCount() int {
	count, _ := strconv.Atoi(course.Enrollment)
	return count
}
