package gomatrixcloner

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/chzyer/readline"
	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/zerolog"
	"go.mau.fi/util/exzerolog"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/crypto/cryptohelper"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

var (
	// Matrix creds
	MatrixHost     string
	MatrixUsername string
	MatrixPassword string
	MautrixDB      = "/data/mautrix.db"
)

type MtrxClient struct {
	c         *mautrix.Client
	startTime int64
	fromRoom  id.RoomID
	toRoom    id.RoomID

	quitMeDaddy chan struct{}
}

// Run - starts bot!
func Run() {
	mtrx := MtrxClient{}
	mtrx.startTime = time.Now().UnixMilli()

	mtrx.fromRoom = id.RoomID(os.Getenv("MATRIX_SOURCE_ROOM"))
	mtrx.toRoom = id.RoomID(os.Getenv("MATRIX_DESTINATION_ROOM"))

	// boot matrix
	client, err := mautrix.NewClient(MatrixHost, "", "")
	if err != nil {
		panic(err)
	}
	mtrx.c = client

	rl, err := readline.New("[no room]> ")
	if err != nil {
		panic(err)
	}
	defer rl.Close()

	log := zerolog.New(zerolog.NewConsoleWriter(func(w *zerolog.ConsoleWriter) {
		w.Out = rl.Stdout()
		w.TimeFormat = time.Stamp
	})).With().Timestamp().Logger()
	log = log.Level(zerolog.InfoLevel)

	exzerolog.SetupDefaults(&log)
	mtrx.c.Log = log

	var lastRoomID id.RoomID
	syncer := mtrx.c.Syncer.(*mautrix.DefaultSyncer)
	syncer.OnEventType(event.EventMessage, func(ctx context.Context, evt *event.Event) {
		lastRoomID = evt.RoomID
		rl.SetPrompt(fmt.Sprintf("%s> ", lastRoomID))
		log.Info().
			Str("sender", evt.Sender.String()).
			Str("type", evt.Type.String()).
			Str("body", evt.Content.AsMessage().Body)

		// Thread for s p e e d
		go mtrx.handleMessageEvent(ctx, evt)
	})

	syncer.OnEventType(event.StateMember, func(ctx context.Context, evt *event.Event) {
		if evt.GetStateKey() == mtrx.c.UserID.String() && evt.Content.AsMember().Membership == event.MembershipInvite {
			_, err := mtrx.c.JoinRoomByID(ctx, evt.RoomID)
			if err == nil {
				lastRoomID = evt.RoomID
				rl.SetPrompt(fmt.Sprintf("%s> ", lastRoomID))
				log.Info().
					Str("room_id", evt.RoomID.String()).
					Str("inviter", evt.Sender.String()).
					Msg("Joined room after invite")
			} else {
				log.Error().Err(err).
					Str("room_id", evt.RoomID.String()).
					Str("inviter", evt.Sender.String()).
					Msg("Failed to join room after invite")
			}
		}
	})

	cryptoHelper, err := cryptohelper.NewCryptoHelper(mtrx.c, []byte("meow"), MautrixDB)
	if err != nil {
		panic(err)
	}

	cryptoHelper.LoginAs = &mautrix.ReqLogin{
		Type:       mautrix.AuthTypePassword,
		Identifier: mautrix.UserIdentifier{Type: mautrix.IdentifierTypeUser, User: MatrixUsername},
		Password:   MatrixPassword,
	}
	err = cryptoHelper.Init(context.TODO())
	if err != nil {
		panic(err)
	}
	// Set the client crypto helper in order to automatically encrypt outgoing messages
	mtrx.c.Crypto = cryptoHelper

	log.Info().Msg("Now running")
	syncCtx, cancelSync := context.WithCancel(context.Background())
	var syncStopWait sync.WaitGroup
	syncStopWait.Add(1)

	go func() {
		err = mtrx.c.SyncWithContext(syncCtx)
		defer syncStopWait.Done()
		if err != nil && !errors.Is(err, context.Canceled) {
			panic(err)
		}
	}()

	for {
		line, err := rl.Readline()
		if err != nil { // io.EOF
			break
		}
		if lastRoomID == "" {
			log.Error().Msg("Wait for an incoming message before sending messages")
			continue
		}
		resp, err := mtrx.c.SendText(context.TODO(), lastRoomID, line)
		if err != nil {
			log.Error().Err(err).Msg("Failed to send event")
		} else {
			log.Info().Str("event_id", resp.EventID.String()).Msg("Event sent")
		}
	}

	mtrx.sendMessage(context.Background(), mtrx.toRoom, "Bridge reloaded 👌")

	// Keep it running!!
	mtrx.quitMeDaddy = make(chan struct{})
	for {
		select {
		case <-mtrx.quitMeDaddy:
			log.Info().Msg("Received quit command!!!")
			cancelSync()
			syncStopWait.Wait()
			err = cryptoHelper.Close()
			if err != nil {
				log.Error().Err(err).Msg("Error closing database")
			}
			os.Exit(0)
		}
	}
}

func (mtrx *MtrxClient) handleMessageEvent(ctx context.Context, evt *event.Event) {
	if evt.RoomID != mtrx.fromRoom {
		return
	}

	// If parsing own messages.. stop that too
	if evt.Sender.String() == mtrx.c.UserID.String() {
		return
	}

	// If syncing older messages.. stop that now
	if evt.Timestamp < mtrx.startTime {
		return
	}

	mtrx.sendMessage(ctx, mtrx.toRoom, fmt.Sprintf(
		"%s:\n%s",
		evt.Sender.Localpart(),
		evt.Content.AsMessage().Body,
	))
}

func (mtrx *MtrxClient) sendMessage(ctx context.Context, roomID id.RoomID, text string) {
	resp, err := mtrx.c.SendText(ctx, roomID, text)
	if err != nil {
		mtrx.c.Log.Error().Err(err).Msg("Failed to send event")
	} else {
		mtrx.c.Log.Info().Str("event_id", resp.EventID.String()).Msg("Event sent")
	}
}
