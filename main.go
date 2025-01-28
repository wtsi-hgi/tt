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
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/calvinmclean/babyapi"
	"github.com/calvinmclean/babyapi/extensions"
	"github.com/calvinmclean/babyapi/html"
	"github.com/go-chi/render"
	"github.com/wtsi-hgi/tt/cmd"
	"github.com/wtsi-hgi/tt/database"
	"github.com/wtsi-hgi/tt/database/mysql"
)

const (
	perPage = 50
)

func main() {
	cmd.Execute()

	// api := createAPI()
	// api.RunCLI()
}

const (
	rootHTML string = `<!doctype html>
<html>
	<head>
		<meta charset="UTF-8">
		<meta name="viewport" content="width=device-width, initial-scale=1.0">
		<title>Temporary Things</title>
        <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/uikit@3.22.0/dist/css/uikit.min.css" />
        <script src="https://cdn.jsdelivr.net/npm/uikit@3.22.0/dist/js/uikit.min.js"></script>
        <script src="https://cdn.jsdelivr.net/npm/uikit@3.22.0/dist/js/uikit-icons.min.js"></script>
		<script src="https://unpkg.com/htmx.org@2.0.4" integrity="sha384-HGfztofotfshcF7+8n44JQL2oJmowVChPTg48S+jvZoztPfvwD79OC/LTtG6dMp+" crossorigin="anonymous"></script>
		<script src="https://unpkg.com/htmx-ext-sse@2.2.2/sse.js"></script>
	</head>

	<style>
        tr.htmx-swapping td {
            opacity: 0;
            transition: opacity 1s ease-out;
        }
	</style>

	<body>
		Username: <input class="uk-input" name="username" type="text"/>

		<table class="uk-table uk-table-divider uk-table-striped uk-margin-left uk-margin-right">
			<colgroup>
				<col>
				<col>
				<col>
				<col>
				<col>
				<col style="width: 300px;">
			</colgroup>

			<thead>
				<tr hx-headers='{"Accept": "text/html"}' hx-trigger="click" hx-target="tbody#things-list">
					<th>
                        Address<span uk-icon="arrow-up" hx-get="/things?sort=address&dir=DESC"></span><span uk-icon="arrow-down" hx-get="/things?sort=address&dir=ASC"></span>
                    </th>
					<th>
                        Type<span uk-icon="arrow-up" hx-get="/things?sort=type&dir=DESC"></span><span uk-icon="arrow-down" hx-get="/things?sort=type&dir=ASC"></span>
                    </th>
					<th>
                        Reason<span uk-icon="arrow-up" hx-get="/things?sort=reason&dir=DESC"></span><span uk-icon="arrow-down" hx-get="/things?sort=reason&dir=ASC"></span>
                    </th>
					<th>Description</th>
					<th>
                        Removal Date<span uk-icon="arrow-up" hx-get="/things?sort=remove&dir=DESC"></span><span uk-icon="arrow-down" hx-get="/things?sort=remove&dir=ASC"></span>
                    </th>
					<th></th>
				</tr>
			</thead>

			<tbody>
				<form hx-post="/things" hx-include="[name='username']" hx-swap="none" hx-on::after-request="this.reset()">
					<td>
						<input class="uk-input" name="Address" type="text" required>
					</td>
					<td>
						<input class="uk-input" name="Type" type="text" required>
					</td>
					<td>
						<input class="uk-input" name="Reason" type="text" required>
					</td>
					<td>
						<input class="uk-input" name="Description" type="text">
					</td>
					<td>
						<input class="uk-input" name="Remove" type="date" required>
					</td>
					<td>
						<button type="submit" class="uk-button uk-button-primary">Add Thing</button>
					</td>
				</form>
            </tbody>

            <tbody hx-get="/things" hx-headers='{"Accept": "text/html"}' hx-trigger="load" id="things-list" hx-ext="sse" sse-connect="/things/listen" sse-swap="newThing" hx-swap="afterbegin">
			</tbody>
		</table>
	</body>
</html>`

	allThings         html.Template = "allThings"
	allThingsTemplate string        = `
                {{ range . }}
                {{ template "thingRow" . }}
                {{ end }}
`

	thingRow      html.Template = "thingRow"
	thingTemplate string        = `<tr hx-target="this" hx-swap="outerHTML">
	<td>{{ .Address }}</td>
	<td>{{ .Type }}</td>
	<td>{{ .Reason }}</td>
	<td>{{ .Description }}</td>
	<td>{{ .Remove.Format "2006-01-02" }}</td>
	<td>
		<button class="uk-button uk-button-danger" hx-delete="/things/{{ .ID }}" hx-swap="swap:1s">
			Delete
		</button>
	</td>
</tr>`
)

type Thing struct {
	database.Thing
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

	storage := &Storage{}
	err := storage.Apply(api)
	if err != nil {
		log.Fatal(err)
	}

	api.AddCustomRootRoute(http.MethodGet, "/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, rootHTML)
	}))

	// Use AllThings in the GetAll response since it implements HTMLer
	api.SetGetAllResponseWrapper(func(things []*Thing) render.Renderer {
		return AllThings{ResourceList: babyapi.ResourceList[*Thing]{Items: things}}
	})

	api.ApplyExtension(extensions.HTMX[*Thing]{})

	// Add SSE handler endpoint which will receive events on the returned channel and write them to the front-end
	newThingChan := api.AddServerSentEventHandler("/listen")

	// Push events onto the SSE channel when new Things are created
	api.SetAfterCreateOrUpdate(func(w http.ResponseWriter, r *http.Request, t *Thing) *babyapi.ErrResponse {
		if r.Method != http.MethodPost {
			return nil
		}

		select {
		case newThingChan <- &babyapi.ServerSentEvent{Event: "newThing", Data: t.HTML(w, r)}:
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

	return api
}

type Storage struct {
	database.Queries
}

func (s Storage) Get(ctx context.Context, idStr string) (*Thing, error) {
	// id, err := strconv.Atoi(idStr)
	// if err != nil {
	// 	return nil, err
	// }

	//TODO: do we actually need a GetThingByID?

	result, err := s.Queries.GetThings(database.GetThingsParams{
		Page:          1,
		ThingsPerPage: 1,
	})
	if err != nil {
		return nil, err
	}

	return &Thing{Thing: result.Things[0]}, nil
}

func (s Storage) GetAll(ctx context.Context, query url.Values) ([]*Thing, error) {
	dir := query.Get("dir")

	sortCol := query.Get("sort")
	if sortCol == "" {
		sortCol = "remove"
	}

	page, err := strconv.Atoi(query.Get("page"))
	if err != nil || page < 1 {
		page = 1
	}

	var thingType database.ThingsType

	switch query.Get("type") {
	case string(database.ThingsTypeDir):
		thingType = database.ThingsTypeDir
	}

	result, err := s.Queries.GetThings(database.GetThingsParams{
		FilterOnType:   thingType,
		OrderBy:        database.OrderBy(sortCol),
		OrderDirection: database.OrderDirection(dir),
		Page:           page,
		ThingsPerPage:  perPage,
	})
	if err != nil {
		return nil, err
	}

	results := make([]*Thing, len(result.Things))
	for i, t := range result.Things {
		results[i] = &Thing{Thing: t}
	}

	return results, nil
}

func (s Storage) Set(ctx context.Context, t *Thing) error {
	thing, err := s.Queries.CreateThing(database.CreateThingParams{
		Address:     t.Address,
		Type:        t.Type,
		Description: t.Description,
		Reason:      t.Reason,
		Remove:      t.Remove,
		Creator:     "",
	})
	if err != nil {
		return err
	}

	t.ID = thing.ID

	return err
}

func (s Storage) Delete(ctx context.Context, idStr string) error {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return err
	}

	return s.Queries.DeleteThing(uint32(id))
}

func (s *Storage) Apply(api *babyapi.API[*Thing]) error {
	config, err := mysql.ConfigFromEnv()
	if err != nil {
		log.Fatal(err)
	}

	database, err := mysql.New(config)
	if err != nil {
		return fmt.Errorf("error opening database: %w", err)
	}

	// go func() {
	// 	<-api.Done()
	// 	database.Close()
	// }()

	s.Queries = database
	api.SetStorage(s)

	return nil
}
