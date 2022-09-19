package postgres

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/arobson/spry"
	"github.com/arobson/spry/storage"
	"github.com/arobson/spry/tests"
	"github.com/gofrs/uuid"
	"github.com/jackc/pgx/v4"
)

var CONNECTION_STRING = "postgres://spry:yippyskippy@localhost:5540/sprydb"

func TruncateTable(tableName string) error {
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, CONNECTION_STRING)
	if err != nil {
		return err
	}
	defer conn.Close(ctx)
	_, err = conn.Exec(
		ctx,
		fmt.Sprintf("TRUNCATE TABLE %s;", tableName),
	)
	if err != nil {
		return err
	}
	return nil
}

func TestCommandStorage(t *testing.T) {
	store := CreatePostgresStorage(
		CONNECTION_STRING,
	)

	uid, _ := storage.GetId()
	c1 := tests.CreatePlayer{Name: "Bob"}
	cr1, _ := storage.NewCommandRecord(c1)

	cr1.CreatedOn = time.Now()
	cr1.Data = c1
	cr1.HandledBy = uid
	cr1.HandledOn = time.Now()
	cr1.HandledVersion = 0
	cr1.Id = uid
	cr1.ReceivedOn = time.Now()
	cr1.Type = "CreatePlayer"

	err := store.AddCommand("Player", cr1)
	if err != nil {
		t.Fatal("failed to store command correctly")
	}

	err = TruncateTable("player_commands")
	if err != nil {
		t.Error(err)
	}
}

func TestEventStorage(t *testing.T) {
	store := CreatePostgresStorage(
		CONNECTION_STRING,
	)

	aid1, _ := storage.GetId()

	e1 := tests.PlayerCreated{Name: "Bill"}
	er1, _ := storage.NewEventRecord(e1)
	er1.ActorId = aid1
	er1.ActorType = "Player"
	er1.CreatedBy = "Player"
	er1.CreatedById = aid1
	er1.Id, _ = storage.GetId()
	er1.Data = e1
	er1.Type = "PlayerCreated"

	e2 := tests.PlayerDamaged{Damage: 10}
	er2, _ := storage.NewEventRecord(e2)
	er2.ActorId = aid1
	er2.ActorType = "Player"
	er2.CreatedBy = "Player"
	er2.CreatedById = aid1
	er2.Id, _ = storage.GetId()
	er2.Data = e1
	er2.Type = "PlayerCreated"

	err := store.AddEvents("Player", []storage.EventRecord{
		er1,
		er2,
	})

	if err != nil {
		t.Error(err)
	}

	records, err := store.FetchEventsSince("player", aid1, uuid.Nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 2 {
		t.Fatalf("expected %d records but got %d instead", 2, len(records))
	}

	err = TruncateTable("player_events")
	if err != nil {
		t.Error(err)
	}
}

func TestMapStorage(t *testing.T) {
	store := CreatePostgresStorage(
		CONNECTION_STRING,
	)

	ids1 := spry.Identifiers{"Name": "Gandalf", "Title": "The Grey"}
	ids2 := spry.Identifiers{"Name": "Gandalf", "Title": "The White"}
	aid, _ := storage.GetId()

	err := store.AddMap("Player", ids1, aid)
	if err != nil {
		t.Fatal("failed to add id map for id1", err)
	}
	err = store.AddMap("Player", ids2, aid)
	if err != nil {
		t.Fatal("failed to add id map for id2", err)
	}

	read1, err := store.FetchId("Player", ids1)
	if err != nil {
		t.Fatal("failed to read id for ids1", err)
	}
	read2, err := store.FetchId("Player", ids2)
	if err != nil {
		t.Fatal("failed to read id for ids1", err)
	}

	if read1 != aid {
		t.Fatal("loaded the incorrect id for ids1")
	}
	if read2 != aid {
		t.Fatal("loaded the incorrect id for ids2")
	}

	err = TruncateTable("player_id_map")
	if err != nil {
		t.Error(err)
	}
}

func TestSnapshotStorage(t *testing.T) {
	store := CreatePostgresStorage(
		CONNECTION_STRING,
	)

	uid1, _ := storage.GetId()
	uid2, _ := storage.GetId()
	person1 := tests.Player{
		Name:      "Billy",
		HitPoints: 100,
		Dead:      false,
	}
	snap1 := storage.Snapshot{
		Id:            uid1,
		ActorId:       uid1,
		Type:          "Player",
		Version:       0,
		CreatedOn:     time.Now(),
		EventsApplied: 1,
		LastEventId:   uid1,
		LastCommandId: uid1,
		LastCommandOn: time.Now(),
		LastEventOn:   time.Now(),
		Data:          person1,
	}

	person2 := tests.Player{
		Name:      "Billy",
		HitPoints: 0,
		Dead:      true,
	}
	snap2 := storage.Snapshot{
		Id:            uid2,
		ActorId:       uid1,
		Type:          "Player",
		Version:       0,
		CreatedOn:     time.Now(),
		EventsApplied: 2,
		LastEventId:   uid2,
		LastCommandId: uid2,
		LastCommandOn: time.Now(),
		LastEventOn:   time.Now(),
		Data:          person2,
	}

	err := store.AddSnapshot("Player", snap1)
	if err != nil {
		t.Fatal("failed to persist snapshot 1", err)
	}
	err = store.AddSnapshot("Player", snap2)
	if err != nil {
		t.Fatal("failed to persist snapshot 2", err)
	}

	latest, err := store.FetchLatestSnapshot("player", uid1)
	if err != nil {
		t.Fatal("failed to read the latest snapshot for uuid")
	}
	if latest.ActorId != uid1 ||
		latest.Data == "" ||
		latest.EventsApplied != 2 ||
		latest.LastEventId != uid2 ||
		latest.LastCommandId != uid2 {
		t.Fatal("snapshot record did not load or deserialize correctly")
	}

	err = TruncateTable("player_snapshots")
	if err != nil {
		t.Error(err)
	}
}
