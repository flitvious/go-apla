// Apla Software includes an integrated development
// environment with a multi-level system for the management
// of access rights to data, interfaces, and Smart contracts. The
// technical characteristics of the Apla Software are indicated in
// Apla Technical Paper.
//
// Apla Users are granted a permission to deal in the Apla
// Software without restrictions, including without limitation the
// rights to use, copy, modify, merge, publish, distribute, sublicense,
// and/or sell copies of Apla Software, and to permit persons
// to whom Apla Software is furnished to do so, subject to the
// following conditions:
// * the copyright notice of GenesisKernel and EGAAS S.A.
// and this permission notice shall be included in all copies or
// substantial portions of the software;
// * a result of the dealing in Apla Software cannot be
// implemented outside of the Apla Platform environment.
//
// THE APLA SOFTWARE IS PROVIDED “AS IS”, WITHOUT WARRANTY
// OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED
// TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A
// PARTICULAR PURPOSE, ERROR FREE AND NONINFRINGEMENT. IN
// NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE
// LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
// FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR
// THE USE OR OTHER DEALINGS IN THE APLA SOFTWARE.

package api

import (
	"encoding/json"
	"net/http"

	"github.com/AplaProject/go-apla/packages/conf/syspar"
	"github.com/AplaProject/go-apla/packages/consts"
	"github.com/AplaProject/go-apla/packages/converter"
	"github.com/AplaProject/go-apla/packages/model"

	log "github.com/sirupsen/logrus"
)

type roleInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type keyInfoResult struct {
	Ecosystem string     `json:"ecosystem"`
	Name      string     `json:"name"`
	Roles     []roleInfo `json:"roles,omitempty"`
}

func keyInfo(w http.ResponseWriter, r *http.Request, data *apiData, logger *log.Entry) (err error) {

	keysList := make([]keyInfoResult, 0)
	keyID := converter.StringToAddress(data.params[`wallet`].(string))
	if keyID == 0 {
		return errorAPI(w, `E_INVALIDWALLET`, http.StatusBadRequest, data.params[`wallet`].(string))
	}
	ids, names, err := model.GetAllSystemStatesIDs()
	if err != nil {
		return errorAPI(w, err, http.StatusInternalServerError)
	}

	var (
		found bool
	)
	for i, ecosystemID := range ids {
		key := &model.Key{}
		key.SetTablePrefix(ecosystemID)
		found, err = key.Get(keyID)
		if err != nil {
			return errorAPI(w, err, http.StatusInternalServerError)
		}
		if !found {
			continue
		}
		keyRes := keyInfoResult{Ecosystem: converter.Int64ToStr(ecosystemID),
			Name: names[i]}
		ra := &model.RolesParticipants{}
		roles, err := ra.SetTablePrefix(ecosystemID).GetActiveMemberRoles(keyID)
		if err != nil {
			return errorAPI(w, err, http.StatusInternalServerError)
		}
		for _, r := range roles {
			var role roleInfo
			if err := json.Unmarshal([]byte(r.Role), &role); err != nil {
				log.WithFields(log.Fields{"type": consts.JSONUnmarshallError, "error": err}).Error("unmarshalling role")
				return errorAPI(w, `E_SERVER`, http.StatusInternalServerError)
			} else {
				keyRes.Roles = append(keyRes.Roles, role)
			}
		}
		keysList = append(keysList, keyRes)
	}

	if len(keysList) == 0 && syspar.IsTestMode() {
		keysList = append(keysList, keyInfoResult{
			Ecosystem: converter.Int64ToStr(ids[0]),
			Name:      names[0],
		})
	}

	data.result = &keysList
	return
}
