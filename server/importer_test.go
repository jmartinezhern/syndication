/*
  Copyright (C) 2017 Jorge Martinez Hernandez

  This program is free software: you can redistribute it and/or modify
  it under the terms of the GNU Affero General Public License as published by
  the Free Software Foundation, either version 3 of the License, or
  (at your option) any later version.

  This program is distributed in the hope that it will be useful,
  but WITHOUT ANY WARRANTY; without even the implied warranty of
  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
  GNU Affero General Public License for more details.

  You should have received a copy of the GNU Affero General Public License
  along with this program.  If not, see <http://www.gnu.org/licenses/>.
*/

package server

import (
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/labstack/echo"
)

func (t *ServerTestSuite) TestOPMLImport() {
	data := `
	<opml>
		<body>
			<outline text="Sports" title="Sports">
				<outline
					type="rss"
					text="Basketball"
					title="Basketball"
					xmlUrl="http://example.com/basketball"
					htmlUrl="http://example.com/basketball"
					/>
			</outline>
			<outline
				type="rss"
				text="Baseball"
				title="Baseball"
				xmlUrl="http://example.com/baseball"
				htmlUrl="http://example.com/baseball"
				/>
			</outline>
		</body>
	</opml>
	`

	req := httptest.NewRequest(echo.POST, "/", strings.NewReader(data))
	req.Header.Set("Content-Type", "application/xml")

	c := t.e.NewContext(req, t.rec)
	c.Set(userContextKey, t.user)

	c.SetPath("/v1/import")

	t.NoError(t.server.Import(c))
	t.Equal(http.StatusNoContent, t.rec.Code)
}
