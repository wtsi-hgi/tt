/*******************************************************************************
 * Copyright (c) 2025 Genome Research Ltd.
 *
 * Author: Sendu Bala <sb10@sanger.ac.uk>
 *
 * Permission is hereby granted, free of charge, to any person obtaining
 * a copy of this software and associated documentation files (the
 * "Software"), to deal in the Software without restriction, including
 * without limitation the rights to use, copy, modify, merge, publish,
 * distribute, sublicense, and/or sell copies of the Software, and to
 * permit persons to whom the Software is furnished to do so, subject to
 * the following conditions:
 *
 * The above copyright notice and this permission notice shall be included
 * in all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
 * EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
 * MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
 * IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY
 * CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
 * TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
 * SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
 ******************************************************************************/

// package main is the access point to our cmd package sub-commands.

package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/go-sql-driver/mysql"
	_ "github.com/joho/godotenv/autoload"
	"github.com/spf13/cobra"

	"github.com/calvinmclean/babyapi"
	"github.com/calvinmclean/babyapi/extensions"
	"github.com/calvinmclean/babyapi/html"
	"github.com/go-chi/render"
	"github.com/wtsi-hgi/tt/db"
)

const (
	sqlDriverName = "mysql"
	sqlNetwork    = "tcp"
)

func main() {
	config, err := getSQLConfigFromEnv()
	if err != nil {
		log.Fatal(err)
	}

	if false {
		if err := tryDatabase(config); err != nil {
			log.Fatal(err)
		}
	}

	api := createAPI()
	api.RunCLI()
}

func getSQLConfigFromEnv() (*mysql.Config, error) {
	user := os.Getenv("TT_SQL_USER")
	pass := os.Getenv("TT_SQL_PASS")
	host := os.Getenv("TT_SQL_HOST")
	port := os.Getenv("TT_SQL_PORT")
	dbname := os.Getenv("TT_SQL_DB")

	if user == "" || pass == "" || host == "" || port == "" || dbname == "" {
		return nil, fmt.Errorf("missing required environment variables")
	}

	conf := mysql.NewConfig()
	conf.User = user
	conf.Passwd = pass
	conf.Net = sqlNetwork
	conf.Addr = fmt.Sprintf("%s:%s", host, port)
	conf.DBName = dbname
	conf.ParseTime = true

	return conf, nil
}

func tryDatabase(config *mysql.Config) error {
	ctx := context.Background()

	sqlDB, err := sql.Open(sqlDriverName, config.FormatDSN())
	if err != nil {
		return err
	}

	queries := db.New(sqlDB)

	err = queries.Reset(ctx)
	if err != nil {
		return err
	}

	err = queries.ResetUsers(ctx)
	if err != nil {
		return err
	}

	result, err := queries.CreateUser(ctx, db.CreateUserParams{Name: "sb10", Email: "sb10@sanger.ac.uk"})
	if err != nil {
		return err
	}

	uid, err := result.LastInsertId()
	if err != nil {
		return err
	}

	result, err = queries.CreateThing(ctx, db.CreateThingParams{
		Address: "/a/file.txt",
		Type:    db.ThingsTypeFile,
		Created: time.Now(),
		Reason:  "hgi resource",
		Remove:  time.Now().Add(time.Hour * 25),
	})
	if err != nil {
		return err
	}

	tid, err := result.LastInsertId()
	if err != nil {
		return err
	}

	_, err = queries.Subscribe(ctx, db.SubscribeParams{
		UserID:  uint32(uid),
		ThingID: uint32(tid),
	})
	if err != nil {
		return err
	}

	things, err := queries.ListThings(ctx)
	if err != nil {
		return err
	}

	log.Println(things)

	return nil
}

const (
	allThings         html.Template = "allThings"
	allThingsTemplate string        = `<!doctype html>
<html>
	<head>
		<meta charset="UTF-8">
		<meta name="viewport" content="width=device-width, initial-scale=1.0">
		<title>TODOs</title>
		<link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/uikit@3.17.11/dist/css/uikit.min.css" />
		<script src="https://unpkg.com/htmx.org@1.9.8"></script>
		<script src="https://unpkg.com/htmx.org/dist/ext/sse.js"></script>
	</head>

	<style>
	tr.htmx-swapping td {
		opacity: 0;
		transition: opacity 1s ease-out;
	}
	</style>

	<body>
		<table class="uk-table uk-table-divider uk-margin-left uk-margin-right">
			<colgroup>
				<col>
				<col>
				<col>
				<col>
				<col>
				<col style="width: 300px;">
			</colgroup>

			<thead>
				<tr>
					<th>Address</th>
					<th>Type</th>
					<th>Reason</th>
					<th>Description</th>
					<th>Removal Date</th>
					<th></th>
				</tr>
			</thead>

			<tbody hx-ext="sse" sse-connect="/things/listen" sse-swap="newThing" hx-swap="beforeend">
				<form hx-post="/things" hx-swap="none" hx-on::after-request="this.reset()">
					<td>
						<input class="uk-input" name="Address" type="text">
					</td>
					<td>
						<input class="uk-input" name="Type" type="text">
					</td>
					<td>
						<input class="uk-input" name="Reason" type="text">
					</td>
					<td>
						<input class="uk-input" name="Description" type="text">
					</td>
					<td>
						<input class="uk-input" name="Remove" type="text">
					</td>
					<td>
						<button type="submit" class="uk-button uk-button-primary">Add Thing</button>
					</td>
				</form>

				{{ range . }}
				{{ template "thingRow" . }}
				{{ end }}
			</tbody>
		</table>
	</body>
</html>`

	thingRow      html.Template = "thingRow"
	thingTemplate string        = `<tr hx-target="this" hx-swap="outerHTML">
	<td>{{ .Address }}</td>
	<td>{{ .Type }}</td>
	<td>{{ .Reason }}</td>
	<td>{{ .Description }}</td>
	<td>{{ .Remove }}</td>
	<td>
		<button class="uk-button uk-button-danger" hx-delete="/things/{{ .ID }}" hx-swap="swap:1s">
			Delete
		</button>
	</td>
</tr>`
)

type Thing struct {
	babyapi.DefaultResource
	db.Thing
}

func (t *Thing) HTML(_ http.ResponseWriter, r *http.Request) string {
	return thingRow.Render(r, t)
}

func (t Thing) GetID() string {
	return fmt.Sprint(t.Thing.ID)
}

func (t *Thing) Bind(r *http.Request) error {
	if r.Method == http.MethodPost {
		t.Thing.Created = time.Now()
	}

	return nil
}

func (t Thing) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

type AllThings struct {
	babyapi.ResourceList[*Thing]
}

func (at AllThings) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

func (at AllThings) HTML(_ http.ResponseWriter, r *http.Request) string {
	return allThings.Render(r, at.Items)
}

func createAPI() *babyapi.API[*Thing] {
	api := babyapi.NewAPI("Things", "/things", func() *Thing { return &Thing{} })

	api.AddCustomRootRoute(http.MethodGet, "/", http.RedirectHandler("/things", http.StatusFound))

	// Use AllThings in the GetAll response since it implements HTMLer
	api.SetGetAllResponseWrapper(func(things []*Thing) render.Renderer {
		return AllThings{ResourceList: babyapi.ResourceList[*Thing]{Items: things}}
	})

	api.ApplyExtension(extensions.HTMX[*Thing]{})

	// Add SSE handler endpoint which will receive events on the returned channel and write them to the front-end
	todoChan := api.AddServerSentEventHandler("/listen")

	// Push events onto the SSE channel when new TODOs are created
	api.SetOnCreateOrUpdate(func(w http.ResponseWriter, r *http.Request, t *Thing) *babyapi.ErrResponse {
		if r.Method != http.MethodPost {
			return nil
		}

		select {
		case todoChan <- &babyapi.ServerSentEvent{Event: "newThing", Data: t.HTML(w, r)}:
		default:
			logger := babyapi.GetLoggerFromContext(r.Context())
			logger.Info("no listeners for server-sent event")
		}
		return nil
	})

	html.SetMap(map[string]string{
		string(allThings): allThingsTemplate,
		string(thingRow):  thingTemplate,
	})

	cmd := api.Command()

	cmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		// Only setup DB for the serve command
		if cmd.Name() != "serve" {
			return nil
		}

		storage := &Storage{}
		return storage.Apply(api)
	}

	// cmd.Execute()

	return api
}

type Storage struct {
	*db.Queries
}

func (s Storage) Get(ctx context.Context, idStr string) (*Thing, error) {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return nil, err
	}

	t, err := s.Queries.GetThingByID(ctx, uint32(id))
	if err != nil {
		return nil, err
	}

	return &Thing{Thing: t}, nil
}

func (s Storage) GetAll(ctx context.Context, query url.Values) ([]*Thing, error) {
	var things []db.Thing
	var err error

	thingType := query.Get("type")
	if thingType != "" {
		things, err = s.Queries.ListThingsByType(ctx, db.ThingsType(thingType))
	} else {
		things, err = s.Queries.ListThings(ctx)
	}
	if err != nil {
		return nil, err
	}

	var result []*Thing
	for _, t := range things {
		result = append(result, &Thing{Thing: t})
	}

	return result, nil
}

func (s Storage) Set(ctx context.Context, t *Thing) error {
	_, err := s.Queries.CreateThing(ctx, db.CreateThingParams{
		Address:     t.Address,
		Type:        db.ThingsType(t.Type),
		Created:     time.Now(),
		Description: t.Description,
		Reason:      t.Reason,
		Remove:      t.Remove,
	})

	return err
}

func (s Storage) Delete(ctx context.Context, idStr string) error {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return err
	}

	return s.Queries.DeleteThing(ctx, uint32(id))
}

func (s *Storage) Apply(api *babyapi.API[*Thing]) error {
	config, err := getSQLConfigFromEnv()
	if err != nil {
		log.Fatal(err)
	}

	database, err := sql.Open(sqlDriverName, config.FormatDSN())
	if err != nil {
		return fmt.Errorf("error opening database: %w", err)
	}

	go func() {
		<-api.Done()
		database.Close()
	}()

	s.Queries = db.New(database)
	api.SetStorage(s)

	return nil
}
