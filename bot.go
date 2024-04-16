package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	tele "gopkg.in/telebot.v3"
)

const configPath = "./config/dialogue.json"

func main() {
	dialogueConfig, err := LoadDialogueFromJSON()
    if (err != nil) {
        log.Fatalf("Config parsing error: %s", err)
        return
    }

	pref := tele.Settings{
		Token:  os.Getenv("BOT_TOKEN"),
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	}

	bot, err := tele.NewBot(pref)
	if err != nil {
		log.Fatal(err)
		return
	}

	bot.Handle("/start", func(c tele.Context) error {
		if len(c.Text()) < 7 {
			return nil
		}

		startArgs := c.Text()[7:]
		chatID := c.Chat().ID
		argument := startArgs

		dialogueConfig = tryAddNewUser(chatID, argument, *dialogueConfig, bot)
		return nil
	})

	bot.Handle(tele.OnText, func(c tele.Context) error {
		recipient := tryGetRecipientByChatId(c.Chat().ID, *dialogueConfig)
		if recipient == nil {
			return nil
		}

		bot.Send(recipient, c.Text())
		return nil
	})

	bot.Handle(tele.OnPhoto, func(c tele.Context) error {
		recipient := tryGetRecipientByChatId(c.Chat().ID, *dialogueConfig)
		if recipient == nil {
			return nil
		}

		photoMessage := tele.Photo{
			File:    c.Message().Photo.File,
			Caption: c.Message().Caption,
		}

		bot.Send(
			recipient,
			&photoMessage,
		)
		return nil
	})

	bot.Start()
}

func tryGetRecipientByChatId(chatID int64, dc DialogueConfig) *tele.Chat {
	if dc.User1ID == chatID && dc.User2ID != 0 {
		chat := tele.Chat{
			ID: dc.User2ID,
		}
		return &chat
	}

	if dc.User2ID == chatID && dc.User1ID != 0 {
		chat := tele.Chat{
			ID: dc.User1ID,
		}
		return &chat
	}

	log.Printf("No companion found for %d\n", chatID)
	return nil
}

func tryAddNewUser(
	chatID int64,
	userKey string,
	dc DialogueConfig,
	bot *tele.Bot,
) *DialogueConfig {
	if userKey == "" {
		return &dc
	}

	hasChanges := false
	var recipient2ID int64

	if dc.User1Key == userKey {
		log.Printf("Saved user 1: %d\n", chatID)
		dc.User1Key = ""
		dc.User1ID = chatID
		hasChanges = true
		recipient2ID = dc.User2ID
	}

	if dc.User2Key == userKey {
		log.Printf("Saved user 2: %d\n", chatID)
		dc.User2Key = ""
		dc.User2ID = chatID
		hasChanges = true
		recipient2ID = dc.User1ID
	}

	chat := tele.Chat{
		ID: chatID,
	}

	if !hasChanges {
		bot.Send(&chat, "Неправильный пригласительный код")
		return &dc
	}

	bot.Send(&chat, "Приветствую в чатике!")

	if recipient2ID != 0 {
		chat2 := tele.Chat{
			ID: recipient2ID,
		}
		bot.Send(&chat2, "Собеседник присоединился к чату")
	}

	SaveConfigToJSON(&dc)
	return &dc
}

func SaveConfigToJSON(dc *DialogueConfig) error {
	data, err := json.MarshalIndent(dc, "", "    ")
	if err != nil {
		return err
	}

	createFileWithDirs(configPath)

	return ioutil.WriteFile(configPath, data, 0644)
}

func createFileWithDirs(path string) error {
	dir := filepath.Dir(path)

	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("unable to create directories: %w", err)
	}

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("unable to create file: %w", err)
	}
	defer file.Close()

	return nil
}

func LoadDialogueFromJSON() (*DialogueConfig, error) {
	_, err := os.Stat(configPath)
	if os.IsNotExist(err) {
		exampleDc := DialogueConfig{
			0,
			0,
			makeRandomString(6),
			makeRandomString(6),
		}
		SaveConfigToJSON(&exampleDc)
		log.Printf(
			"New config created! Use links:\nhttps://t.me/your_bot_name?start=%s for the first person and\nhttps://t.me/your_bot_name?start=%s for the second one",
			exampleDc.User1Key,
			exampleDc.User2Key,
		)
	}

	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var dc DialogueConfig
	if err = json.Unmarshal(data, &dc); err != nil {
		return nil, err
	}
	return &dc, nil
}

func makeRandomString(length int) string {
	alphabet := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	var result []byte
	charsetLength := len(alphabet)

	// Инициализируем генератор случайных чисел.
	rand.Seed(time.Now().UnixNano())

	for i := 0; i < length; i++ {
		randomIndex := rand.Intn(charsetLength)
		result = append(result, alphabet[randomIndex])
	}

	return string(result)
}

type DialogueConfig struct {
	User1ID  int64
	User2ID  int64
	User1Key string
	User2Key string
}
