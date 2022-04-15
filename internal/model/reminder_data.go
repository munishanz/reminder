package model

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"sort"
	"strconv"
	"strings"

	"reminder/pkg/utils"
)

/*
A ReminderData represents the whole reminder data-structure.
*/
type ReminderData struct {
	User         *User  `json:"user"`
	Notes        Notes  `json:"notes"`
	Tags         Tags   `json:"tags"`
	DataFile     string `json:"data_file"`
	LastBackupAt int64  `json:"last_backup_at"`
	BaseStruct
}

// UpdateDataFile updates data file.
// The msg is any additional message to be printed.
func (reminderData *ReminderData) UpdateDataFile(msg string) error {
	// update UpdatedAt field
	// note that UpdatedAt of a whole ReminderData object is different
	// from corresponding field of each note
	reminderData.UpdatedAt = utils.CurrentUnixTimestamp()
	// marshal the data
	// Refer https://pkg.go.dev/encoding/json#MarshalIndent
	// Note: String values encoded as JSON strings are coerced to valid
	// UTF-8, replacing invalid bytes with Unicode replacement rune. So
	// that the JSON will be safe to embed inside HTML <script> tags, the
	// string is encoded using HTMLEscape.
	byteValue, err := json.MarshalIndent(&reminderData, "", "    ")
	if err != nil {
		utils.PrintErrorIfPresent(err)
		return err
	}
	// persist the byte data to file
	err = ioutil.WriteFile(reminderData.DataFile, byteValue, 0755)
	if err != nil {
		utils.PrintErrorIfPresent(err)
		return err
	}
	if msg != "" {
		fmt.Println(msg)
	}
	fmt.Println("Updated the data file!")
	return nil
}

// SortedTagSlugs sorts the tags in-place and return slugs.
// Empty Tags is returned if there are no tags.
func (reminderData *ReminderData) SortedTagSlugs() []string {
	// sort tags in place
	sort.Sort(reminderData.Tags)
	// fetch slugs and return
	return reminderData.Tags.Slugs()
}

// TagFromSlug returns tag with given slug.
func (reminderData *ReminderData) TagFromSlug(slug string) *Tag {
	return reminderData.Tags.FromSlug(slug)
}

// TagsFromIds returns tags from tagIDs.
func (reminderData *ReminderData) TagsFromIds(tagIDs []int) Tags {
	return reminderData.Tags.FromIds(tagIDs)
}

// TagIdsForGroup gets tag ids for given group.
func (reminderData *ReminderData) TagIdsForGroup(group string) []int {
	return reminderData.Tags.IdsForGroup(group)
}

// FindNotesByTagId gets all notes with given tagID and given status.
func (reminderData *ReminderData) FindNotesByTagId(tagID int, status string) Notes {
	return reminderData.Notes.WithTagIdAndStatus(tagID, status)
}

// FindNotesByTagSlug gets all notes with given tagSlug and given status.
func (reminderData *ReminderData) FindNotesByTagSlug(tagSlug string, status string) Notes {
	tag := reminderData.TagFromSlug(tagSlug)
	// return empty Notes object for nil `tag`
	if tag == nil {
		return Notes{}
	}
	return reminderData.FindNotesByTagId(tag.Id, status)
}

// UpdateNoteText updates note's text.
func (reminderData *ReminderData) UpdateNoteText(note *Note, text string) error {
	err := note.UpdateText(text)
	if err != nil {
		return err
	}
	return reminderData.UpdateDataFile("")
}

// UpdateNoteSummary updates the note's summary.
func (reminderData *ReminderData) UpdateNoteSummary(note *Note, text string) error {
	err := note.UpdateSummary(text)
	if err != nil {
		return err
	}
	return reminderData.UpdateDataFile("")
}

// UpdateNoteCompleteBy updates the note's due date (complete by).
func (reminderData *ReminderData) UpdateNoteCompleteBy(note *Note, text string) error {
	err := note.UpdateCompleteBy(text)
	if err != nil {
		return err
	}
	return reminderData.UpdateDataFile("")
}

// AddNoteComment adds note's comment.
func (reminderData *ReminderData) AddNoteComment(note *Note, text string) error {
	err := note.AddComment(text)
	if err != nil {
		return err
	}
	return reminderData.UpdateDataFile("")
}

// UpdateNoteTags updates note's tags.
func (reminderData *ReminderData) UpdateNoteTags(note *Note, tagIDs []int) error {
	err := note.UpdateTags(tagIDs)
	if err != nil {
		return err
	}
	return reminderData.UpdateDataFile("")
}

// UpdateNoteStatus updates note's status.
func (reminderData *ReminderData) UpdateNoteStatus(note *Note, status string) error {
	repeatTagIDs := reminderData.TagIdsForGroup("repeat")
	err := note.UpdateStatus(status, repeatTagIDs)
	if err != nil {
		return err
	}
	return reminderData.UpdateDataFile("")
}

// ToggleNoteMainFlag toggles note's priority.
func (reminderData *ReminderData) ToggleNoteMainFlag(note *Note) error {
	err := note.ToggleMainFlag()
	if err != nil {
		return err
	}
	return reminderData.UpdateDataFile("")
}

// RegisterBasicTags registers basic tags.
func (reminderData *ReminderData) RegisterBasicTags() error {
	if len(reminderData.Tags) != 0 {
		msg := fmt.Sprintf("%v Skipped registering basic tags as tag list is not empty\n", utils.Symbols["warning"])
		return errors.New(msg)
	}
	basicTags := BasicTags()
	reminderData.Tags = basicTags
	msg := fmt.Sprintf("Added basic tags: %+v\n", reminderData.Tags)
	return reminderData.UpdateDataFile(msg)
}

// ListTags prompts a list of all tags (and their notes underneath).
// Like utils.AskOptions, it prints any encountered error, and returns that error just for information.
func (reminderData *ReminderData) ListTags() error {
	// function to return a tag sumbol
	// keep different tag symbol for empty tags
	tagSymbol := func(tagSlug string) string {
		PendingNote := reminderData.FindNotesByTagSlug(tagSlug, "pending")
		if len(PendingNote) > 0 {
			return utils.Symbols["tag"]
		} else {
			return utils.Symbols["zzz"]
		}
	}
	// get list of tags with their emojis
	// assuming there are at least 20 tags (on average)
	allTagSlugsWithEmoji := make([]string, 0, 20)
	for _, tagSlug := range reminderData.SortedTagSlugs() {
		allTagSlugsWithEmoji = append(allTagSlugsWithEmoji, fmt.Sprintf("%v %v", tagSymbol(tagSlug), tagSlug))
	}
	// ask user to select a tag
	tagIndex, _, err := utils.AskOption(append(allTagSlugsWithEmoji, fmt.Sprintf("%v %v", utils.Symbols["add"], "Add Tag")), "Select Tag")
	if (err != nil) || (tagIndex == -1) {
		// do nothing, just exit
		return err
	}
	// check if user wants to add a new tag
	if tagIndex == len(reminderData.SortedTagSlugs()) {
		// add new tag
		_, err = reminderData.NewTagRegistration()
		if err != nil {
			return err
		}
		return nil
	}
	// operate on the selected a tag
	tag := reminderData.Tags[tagIndex]
	err = reminderData.PrintNotesAndAskOptions(Notes{}, tag.Id, "pending", false)
	if err != nil {
		utils.PrintErrorIfPresent(err)
		// go back to ListTags
		err = reminderData.ListTags()
		if err != nil {
			return err
		}
	}
	return nil
}

// SearchNotes searches throught all notes.
// Like utils.AskOptions, it prints any encountered error, and returns that error just for information.
func (reminderData *ReminderData) SearchNotes() error {
	// get texts of all notes
	sort.Sort(reminderData.Notes)
	allNotes := reminderData.Notes
	// assuming the search shows 25 items in general
	allTexts := make([]string, 0, 25)
	for _, note := range allNotes {
		allTexts = append(allTexts, note.SearchableText())
	}
	// function to search across notes
	searchNotes := func(filterValue string, optValue string, optIndex int) bool {
		filterValue = strings.ToLower(filterValue)
		noteText := allTexts[optIndex]
		if strings.Contains(strings.ToLower(noteText), filterValue) {
			return true
		}
		return false
	}
	// display prompt
	fmt.Printf("Searching through a total of %v notes:\n", len(allTexts))
	index, err := utils.GenerateNoteSearchSelect(utils.ChopStrings(allTexts, utils.TerminalWidth()-10), searchNotes)
	if err != nil {
		return err
	}
	if index >= 0 {
		note := allNotes[index]
		action := reminderData.PrintNoteAndAskOptions(note)
		if action == "stay" {
			// no action was selected for the note, go one step back
			err := reminderData.SearchNotes()
			if err != nil {
				return err
			}
		}
	}
	return err
}

// NotesApprachingDueDate fetches all pending notes which are urgent.
func (reminderData *ReminderData) NotesApprachingDueDate() Notes {
	allNotes := reminderData.Notes
	pendingNotes := allNotes.WithStatus("pending")
	// assuming there are at least 100 notes (on average)
	currentNotes := make([]*Note, 0, 100)
	repeatTagIDs := reminderData.TagIdsForGroup("repeat")
	// populating currentNotes
	for _, note := range pendingNotes {
		noteIDsWithRepeat := utils.GetCommonMembersIntSlices(note.TagIds, repeatTagIDs)
		// first process notes WITHOUT tag with group "repeat"
		// start showing such notes 7 days in advance from their due date, and until they are marked done
		minDay := note.CompleteBy - 7*24*60*60
		currentTimestamp := utils.CurrentUnixTimestamp()
		if (len(noteIDsWithRepeat) == 0) && (note.CompleteBy != 0) && (currentTimestamp >= minDay) {
			currentNotes = append(currentNotes, note)
		}
		// check notes with tag with group "repeat"
		// start showing notes with "repeat-annually" 7 days in advance
		// start showing notes with "repeat-monthly" 3 days in advance
		// don't show such notes after their due date is past by 2 day
		if (len(noteIDsWithRepeat) > 0) && (note.CompleteBy != 0) {
			// check for repeat-annually tag
			// note: for the CompletedBy date of the note, we accept only date
			// so, even if there is a time element recorded the the timestamp,
			// we ignore it
			repeatAnnuallyTag := reminderData.TagFromSlug("repeat-annually")
			repeatMonthlyTag := reminderData.TagFromSlug("repeat-monthly")
			if (repeatAnnuallyTag != nil) && utils.IntPresentInSlice(repeatAnnuallyTag.Id, note.TagIds) {
				_, noteMonth, noteDay := utils.UnixTimestampToTime(note.CompleteBy).Date()
				noteTimestampCurrent := utils.UnixTimestampForCorrespondingCurrentYear(int(noteMonth), noteDay)
				noteTimestampPrevious := noteTimestampCurrent - 365*24*60*60
				noteTimestampNext := noteTimestampCurrent + 365*24*60*60
				daysBefore := int64(3) // days before to start showing the note
				daysAfter := int64(7)  // days after until to show the note
				if utils.IsTimeForRepeatNote(noteTimestampCurrent, noteTimestampPrevious, noteTimestampNext, daysBefore, daysAfter) {
					currentNotes = append(currentNotes, note)
				}
			}
			// check for repeat-monthly tag
			if (repeatMonthlyTag != nil) && utils.IntPresentInSlice(repeatMonthlyTag.Id, note.TagIds) {
				_, _, noteDay := utils.UnixTimestampToTime(note.CompleteBy).Date()
				noteTimestampCurrent := utils.UnixTimestampForCorrespondingCurrentYearMonth(noteDay)
				noteTimestampPrevious := noteTimestampCurrent - 30*24*60*60
				noteTimestampNext := noteTimestampCurrent + 30*24*60*60
				daysBefore := int64(1) // days beofre to start showing the note
				daysAfter := int64(3)  // days after until to show the note
				if utils.IsTimeForRepeatNote(noteTimestampCurrent, noteTimestampPrevious, noteTimestampNext, daysBefore, daysAfter) {
					currentNotes = append(currentNotes, note)
				}
			}
		}
	}
	return currentNotes
}

// NewTagRegistration registers a new tag.
func (reminderData *ReminderData) NewTagRegistration() (int, error) {
	// collect and ask info about the tag
	tagID := reminderData.nextPossibleTagId()

	tag, err := NewTag(tagID, "", "")

	// validate and save data
	if err != nil {
		utils.PrintErrorIfPresent(err)
		return 0, err
	} else {
		err, _ = reminderData.newTagAppend(tag), tagID
	}
	return tagID, err
}

// nextPossibleTagId gets next possible tagID.
func (reminderData *ReminderData) nextPossibleTagId() int {
	allTags := reminderData.Tags
	allTagsLen := len(allTags)
	return allTagsLen
}

// newTagAppend appends a new tag.
func (reminderData *ReminderData) newTagAppend(tag *Tag) error {
	// check if tag's slug is already present
	isNewSlug := true
	for _, existingTag := range reminderData.Tags {
		if existingTag.Slug == tag.Slug {
			isNewSlug = false
		}
	}
	if !isNewSlug {
		fmt.Printf("%v The tag already exists!\n", utils.Symbols["error"])
		return errors.New("Tag Already Exists")
	}
	// go ahead and append
	fmt.Printf("Added Tag: %v\n", *tag)
	reminderData.Tags = append(reminderData.Tags, tag)
	return reminderData.UpdateDataFile("")
}

// NewNoteRegistration registers new note.
func (reminderData *ReminderData) NewNoteRegistration(tagIDs []int) (*Note, error) {
	// collect info about the note
	if tagIDs == nil {
		// assuming each note with have on average 2 tags
		tagIDs = make([]int, 0, 2)
	}
	note, err := NewNote(tagIDs, "")
	// validate and save data
	if err != nil {
		utils.PrintErrorIfPresent(err)
		return note, err
	}
	err = reminderData.newNoteAppend(note)
	if err != nil {
		utils.PrintErrorIfPresent(err)
		return note, err
	}
	return note, nil
}

// newNoteAppend appends a new note.
func (reminderData *ReminderData) newNoteAppend(note *Note) error {
	fmt.Printf("Added Note: %+v\n", *note)
	reminderData.Notes = append(reminderData.Notes, note)
	return reminderData.UpdateDataFile("")
}

// Stats returns current status.
func (reminderData *ReminderData) Stats() string {
	reportTemplate := `
Stats of "{{.DataFile}}"
  - Number of Tags: {{.Tags | len}}
  - Pending Notes: {{.Notes | numPending}}/{{.Notes | numAll}}
`
	funcMap := template.FuncMap{
		"numPending": func(notes Notes) int { return len(notes.WithStatus("pending")) },
		"numAll":     func(notes Notes) int { return len(notes) },
	}
	return utils.TemplateResult(reportTemplate, funcMap, *reminderData)
}

// CreateBackup creates timestamped backup.
// It returns path of the data file.
// Like utils.AskOptions, it prints any encountered error, but doesn't return the error.
func (reminderData *ReminderData) CreateBackup() (string, error) {
	// get backup file name
	ext := path.Ext(reminderData.DataFile)
	dstFile := reminderData.DataFile[:len(reminderData.DataFile)-len(ext)] + "_backup_" + strconv.FormatInt(int64(utils.CurrentUnixTimestamp()), 10) + ext
	lnFile := reminderData.DataFile[:len(reminderData.DataFile)-len(ext)] + "_backup_latest" + ext
	fmt.Printf("Creating backup at %q\n", dstFile)
	// create backup
	byteValue, err := ioutil.ReadFile(reminderData.DataFile)
	if err != nil {
		return dstFile, err
	}
	err = ioutil.WriteFile(dstFile, byteValue, 0644)
	if err != nil {
		return dstFile, err
	}
	// create alias of latest backup
	fmt.Printf("Creating symlink at %q\n", lnFile)
	executable, _ := exec.LookPath("ln")
	cmd := &exec.Cmd{
		Path:   executable,
		Args:   []string{executable, "-sf", dstFile, lnFile},
		Stdout: os.Stdout,
		Stdin:  os.Stdin,
	}
	err = cmd.Run()
	if err != nil {
		return dstFile, err
	}
	return dstFile, nil
}

// DisplayDataFile displays the data file.
// Like utils.AskOptions, it prints any encountered error, and returns that error just for information.
func (reminderData *ReminderData) DisplayDataFile() error {
	fmt.Printf("Printing contents (and if possible, its difference since last backup) of %q:\n", reminderData.DataFile)
	ext := path.Ext(reminderData.DataFile)
	lnFile := reminderData.DataFile[:len(reminderData.DataFile)-len(ext)] + "_backup_latest" + ext
	err := utils.PerformWhich("wdiff")
	if err != nil {
		fmt.Printf("%v Warning: `wdiff` command is not available\n", utils.Symbols["error"])
		err = utils.PerformCat(reminderData.DataFile)
	} else {
		err = utils.PerformFilePresence(lnFile)
		if err != nil {
			fmt.Printf("Warning: `%v` file is not available yet\n", lnFile)
			err = utils.PerformCat(reminderData.DataFile)
		} else {
			err = utils.PerformCwdiff(lnFile, reminderData.DataFile)
		}
	}
	return err
}

// AutoBackup does auto backup.
func (reminderData *ReminderData) AutoBackup(gapSecs int64) (string, error) {
	var dstFile string
	currentTime := utils.CurrentUnixTimestamp()
	lastBackup := reminderData.LastBackupAt
	gap := currentTime - lastBackup
	fmt.Printf("Automatic Backup Gap = %vs/%vs\n", gap, gapSecs)
	if gap < gapSecs {
		fmt.Printf("Skipping automatic backup\n")
		return dstFile, nil
	}
	dstFile, _ = reminderData.CreateBackup()
	reminderData.LastBackupAt = currentTime
	return dstFile, reminderData.UpdateDataFile("")
}

// AskTagIds (recursive) ask tagIDs that are to be associated with a note..
// It also registers tags for you, if user asks.
func (reminderData *ReminderData) AskTagIds(tagIDs []int) []int {
	var err error
	var tagID int
	// ask user to select tag
	optionIndex, _, _ := utils.AskOption(append(reminderData.SortedTagSlugs(), fmt.Sprintf("%v %v", utils.Symbols["add"], "Add Tag")), "Select Tag")
	if optionIndex == -1 {
		return []int{}
	}
	// get tagID
	if optionIndex == len(reminderData.SortedTagSlugs()) {
		// add new tag
		tagID, err = reminderData.NewTagRegistration()
	} else {
		// existing tag selected
		tagID = reminderData.Tags[optionIndex].Id
		err = nil
	}
	// update tagIDs
	if (err == nil) && (!utils.IntPresentInSlice(tagID, tagIDs)) {
		tagIDs = append(tagIDs, tagID)
	}
	// check with user if another tag is to be added
	promptText, err := utils.GeneratePrompt("tag_another", "")
	utils.PrintErrorIfPresent(err)
	promptText = strings.ToLower(promptText)
	nextTag := false
	for _, yes := range []string{"yes", "y"} {
		if yes == promptText {
			nextTag = true
		}
	}
	if nextTag {
		return reminderData.AskTagIds(tagIDs)
	}
	return tagIDs
}

// PrintNoteAndAskOptions prints note and display options.
// Like utils.AskOptions, it prints any encountered error, but doesn't returns that error just for information.
// It return string representing workflow direction.
func (reminderData *ReminderData) PrintNoteAndAskOptions(note *Note) string {
	fmt.Print(note.ExternalText(reminderData))
	_, noteOption, _ := utils.AskOption([]string{
		fmt.Sprintf("%v %v", utils.Symbols["comment"], "Add comment"),
		fmt.Sprintf("%v %v", utils.Symbols["home"], "Exit to main menu"),
		fmt.Sprintf("%v %v", utils.Symbols["noAction"], "Do nothing"),
		fmt.Sprintf("%v %v", utils.Symbols["upVote"], "Mark as done"),
		fmt.Sprintf("%v %v", utils.Symbols["downVote"], "Mark as pending"),
		fmt.Sprintf("%v %v", utils.Symbols["calendar"], "Update due date"),
		fmt.Sprintf("%v %v", utils.Symbols["tag"], "Update tags"),
		fmt.Sprintf("%v %v", utils.Symbols["text"], "Update text"),
		fmt.Sprintf("%v %v", utils.Symbols["glossary"], "Update summary"),
		fmt.Sprintf("%v %v", utils.Symbols["hat"], "Toggle main/incidental")},
		"Select Action")
	switch noteOption {
	case fmt.Sprintf("%v %v", utils.Symbols["comment"], "Add comment"):
		promptText, err := utils.GeneratePrompt("note_comment", "")
		utils.PrintErrorIfPresent(err)
		err = reminderData.AddNoteComment(note, promptText)
		utils.PrintErrorIfPresent(err)
		fmt.Print(note.ExternalText(reminderData))
	case fmt.Sprintf("%v %v", utils.Symbols["home"], "Exit to main menu"):
		return "main-menu"
	case fmt.Sprintf("%v %v", utils.Symbols["noAction"], "Do nothing"):
		fmt.Println("No changes made")
		fmt.Print(note.ExternalText(reminderData))
	case fmt.Sprintf("%v %v", utils.Symbols["upVote"], "Mark as done"):
		err := reminderData.UpdateNoteStatus(note, "done")
		utils.PrintErrorIfPresent(err)
		fmt.Print(note.ExternalText(reminderData))
	case fmt.Sprintf("%v %v", utils.Symbols["downVote"], "Mark as pending"):
		err := reminderData.UpdateNoteStatus(note, "pending")
		utils.PrintErrorIfPresent(err)
		fmt.Print(note.ExternalText(reminderData))
	case fmt.Sprintf("%v %v", utils.Symbols["calendar"], "Update due date"):
		promptText, err := utils.GeneratePrompt("note_completed_by", "")
		utils.PrintErrorIfPresent(err)
		err = reminderData.UpdateNoteCompleteBy(note, promptText)
		utils.PrintErrorIfPresent(err)
		fmt.Print(note.ExternalText(reminderData))
	case fmt.Sprintf("%v %v", utils.Symbols["text"], "Update text"):
		promptText, err := utils.GeneratePrompt("note_text", note.Text)
		utils.PrintErrorIfPresent(err)
		err = reminderData.UpdateNoteText(note, promptText)
		utils.PrintErrorIfPresent(err)
		fmt.Print(note.ExternalText(reminderData))
	case fmt.Sprintf("%v %v", utils.Symbols["glossary"], "Update summary"):
		promptText, err := utils.GeneratePrompt("note_summary", note.Summary)
		utils.PrintErrorIfPresent(err)
		err = reminderData.UpdateNoteSummary(note, promptText)
		utils.PrintErrorIfPresent(err)
		fmt.Print(note.ExternalText(reminderData))
	case fmt.Sprintf("%v %v", utils.Symbols["tag"], "Update tags"):
		tagIDs := reminderData.AskTagIds([]int{})
		if len(tagIDs) > 0 {
			err := reminderData.UpdateNoteTags(note, tagIDs)
			utils.PrintErrorIfPresent(err)
			fmt.Print(note.ExternalText(reminderData))
		} else {
			fmt.Printf("%v Skipping updating note with empty tagIDs list\n", utils.Symbols["warning"])
		}
	case fmt.Sprintf("%v %v", utils.Symbols["hat"], "Toggle main/incidental"):
		err := reminderData.ToggleNoteMainFlag(note)
		utils.PrintErrorIfPresent(err)
		fmt.Print(note.ExternalText(reminderData))
	}
	return "stay"
}

// PrintNotesAndAskOptions (recursively) prints notes interactively.
// In some cases, updated list notes will be fetched, so blank notes can be passed in those cases.
// Unless notes are to be fetched, the passed `status` doesn't make sense, so in such cases it can be passed as "fake".
// Like utils.AskOptions, it prints any encountered error, and returns that error just for information.
// Filter only the notes tagged as "main" if `onlyMain` is true (for given status), otherwise return all.
func (reminderData *ReminderData) PrintNotesAndAskOptions(notes Notes, tagID int, status string, onlyMain bool) error {
	// check if passed notes is to be used or to fetch latest notes
	if status == "done" {
		// ignore the passed notes
		// fetch all the done notes
		notes = reminderData.Notes.WithStatus("done")
		fmt.Printf("A total of %v notes marked as 'done':\n", len(notes))
	} else if status == "pending" {
		// ignore the passed notes
		if tagID >= 0 {
			// this is for listing all notes associated with given tag, with asked status
			// fetch pending notes with given tagID
			notes = reminderData.FindNotesByTagId(tagID, status)
		} else if onlyMain {
			// this is for listing all main notes, with asked status
			notes = reminderData.Notes.OnlyMain()
			countAllMain := len(notes)
			notes = notes.WithStatus(status)
			countPendingMain := len(notes)
			fmt.Printf("A total of %v/%v notes flagged as 'main':\n", countPendingMain, countAllMain)
		} else {
			// this is for listing all notes approaching due date
			// fetch notes approaching due date
			fmt.Println("Note: Following are the pending notes with due date:")
			fmt.Println("  - within a week or already crossed (for non repeat-annually or repeat-monthly)")
			fmt.Println("  - within 3 days for repeat-annually and a week post due date (ignoring its year)")
			fmt.Println("  - within 1 day for repeat-monthly and 3 days post due date (ignoring its year and month)")
			notes = reminderData.NotesApprachingDueDate()
		}
	} else {
		// use passed notes
		fmt.Printf("Using passed notes, so the list will not be refreshed immediately.\n")
	}
	// sort notes
	sort.Sort(Notes(notes))
	texts := notes.ExternalTexts(utils.TerminalWidth() - 50)
	// ask user to select a note
	promptText := ""
	if tagID >= 0 {
		promptText = fmt.Sprintf("Select Note (for the tag %v %v)", utils.Symbols["tag"], reminderData.TagsFromIds([]int{tagID})[0].Slug)
	} else {
		promptText = fmt.Sprintf("Select Note")
	}
	noteIndex, _, err := utils.AskOption(append(texts, fmt.Sprintf("%v %v", utils.Symbols["add"], "Add Note")), promptText)
	if (err != nil) || (noteIndex == -1) {
		return err
	}
	// create new note
	if noteIndex == len(texts) {
		// add new note
		if tagID < 0 {
			return errors.New("The passed tagID is invalid!")
		}
		note, err := reminderData.NewNoteRegistration([]int{tagID})
		if err != nil {
			return err
		}
		var updatedNotes Notes
		updatedNotes = append(updatedNotes, note)
		updatedNotes = append(updatedNotes, notes...)
		err = reminderData.PrintNotesAndAskOptions(updatedNotes, tagID, status, onlyMain)
		if err != nil {
			return err
		}
		return nil
	}
	// ask options about the selected note
	note := notes[noteIndex]
	action := reminderData.PrintNoteAndAskOptions(note)
	if action == "stay" {
		// no action was selected for the note, go one step back
		err = reminderData.PrintNotesAndAskOptions(notes, tagID, status, onlyMain)
		if err != nil {
			return err
		}
	}
	return nil
}
