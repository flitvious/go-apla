// Copyright 2016 The go-daylight Authors
// This file is part of the go-daylight library.
//
// The go-daylight library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-daylight library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-daylight library. If not, see <http://www.gnu.org/licenses/>.

package api

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/GenesisKernel/go-genesis/packages/conf"
	"github.com/GenesisKernel/go-genesis/packages/consts"
	"github.com/GenesisKernel/go-genesis/packages/notificator"
	"github.com/GenesisKernel/go-genesis/packages/publisher"
	msgpack "gopkg.in/vmihailenco/msgpack.v2"

	"github.com/GenesisKernel/go-genesis/packages/converter"
	"github.com/GenesisKernel/go-genesis/packages/crypto"
	"github.com/GenesisKernel/go-genesis/packages/model"

	"encoding/hex"
	"encoding/json"

	"github.com/GenesisKernel/go-genesis/packages/script"
	"github.com/GenesisKernel/go-genesis/packages/smart"
	"github.com/GenesisKernel/go-genesis/packages/utils"
	"github.com/GenesisKernel/go-genesis/packages/utils/tx"
	"github.com/dgrijalva/jwt-go"
	log "github.com/sirupsen/logrus"
)

// Special word used by frontend to sign UID generated by /getuid API command, sign is performed for contcatenated word and UID
const nonceSalt = "LOGIN"

type loginResult struct {
	Token       string        `json:"token,omitempty"`
	Refresh     string        `json:"refresh,omitempty"`
	EcosystemID string        `json:"ecosystem_id,omitempty"`
	KeyID       string        `json:"key_id,omitempty"`
	Address     string        `json:"address,omitempty"`
	NotifyKey   string        `json:"notify_key,omitempty"`
	IsNode      bool          `json:"isnode,omitempty"`
	IsOwner     bool          `json:"isowner,omitempty"`
	IsVDE       bool          `json:"vde,omitempty"`
	Timestamp   string        `json:"timestamp,omitempty"`
	Roles       []rolesResult `json:"roles,omitempty"`
}

type rolesResult struct {
	RoleId   int64  `json:"role_id"`
	RoleName string `json:"role_name"`
}

func login(w http.ResponseWriter, r *http.Request, data *apiData, logger *log.Entry) error {
	var (
		pubkey []byte
		wallet int64
		msg    string
		err    error
	)

	if data.token != nil && data.token.Valid {
		if claims, ok := data.token.Claims.(*JWTClaims); ok {
			msg = claims.UID
		}
	}
	if len(msg) == 0 {
		logger.WithFields(log.Fields{"type": consts.EmptyObject}).Error("UID is empty")
		return errorAPI(w, `E_UNKNOWNUID`, http.StatusBadRequest)
	}

	ecosystemID := data.ecosystemId
	if data.params[`ecosystem`].(int64) > 0 {
		ecosystemID = data.params[`ecosystem`].(int64)
	}

	if ecosystemID == 0 {
		logger.WithFields(log.Fields{"type": consts.EmptyObject}).Warning("state is empty, using 1 as a state")
		ecosystemID = 1
	}

	if len(data.params[`key_id`].(string)) > 0 {
		wallet = converter.StringToAddress(data.params[`key_id`].(string))
	} else if len(data.params[`pubkey`].([]byte)) > 0 {
		wallet = crypto.Address(data.params[`pubkey`].([]byte))
	}

	account := &model.Key{}
	account.SetTablePrefix(ecosystemID)
	isAccount, err := account.Get(wallet)
	if err != nil {
		logger.WithFields(log.Fields{"type": consts.DBError, "error": err}).Error("selecting public key from keys")
		return errorAPI(w, err, http.StatusBadRequest)
	}

	if isAccount {
		pubkey = account.PublicKey
		if account.Deleted == 1 {
			return errorAPI(w, `E_DELETEDKEY`, http.StatusForbidden)
		}
	} else {
		pubkey = data.params[`pubkey`].([]byte)
		fmt.Println(string(pubkey))
		if len(pubkey) == 0 {
			logger.WithFields(log.Fields{"type": consts.EmptyObject}).Error("public key is empty")
			return errorAPI(w, `E_EMPTYPUBLIC`, http.StatusBadRequest)
		}
		NodePrivateKey, NodePublicKey, err := utils.GetNodeKeys()
		if err != nil || len(NodePrivateKey) < 1 {
			if err == nil {
				log.WithFields(log.Fields{"type": consts.EmptyObject}).Error("node private key is empty")
			}
			return err
		}

		pubkey = data.params[`pubkey`].([]byte)
		hexPubKey := hex.EncodeToString(pubkey)
		params := converter.EncodeLength(int64(len(hexPubKey)))
		params = append(params, hexPubKey...)

		contract := smart.GetContract("NewUser", 1)

		sc := tx.SmartContract{
			Header: tx.Header{
				Type:        int(contract.Block.Info.(*script.ContractInfo).ID),
				Time:        time.Now().Unix(),
				EcosystemID: 1,
				KeyID:       conf.Config.KeyID,
				NetworkID:   consts.NETWORK_ID,
				PublicKey:   pubkey,
			},
			SignedBy: smart.PubToID(NodePublicKey),
			Data:     params,
		}

		if conf.Config.IsSupportingVDE() {

			signPrms := []string{sc.ForSign()}
			signPrms = append(signPrms, hexPubKey)
			signData := strings.Join(signPrms, ",")
			signature, err := crypto.Sign(NodePrivateKey, signData)
			if err != nil {
				log.WithFields(log.Fields{"type": consts.CryptoError, "error": err}).Error("signing by node private key")
				return err
			}

			sc.BinSignatures = converter.EncodeLengthPlusData(signature)

			if sc.PublicKey, err = hex.DecodeString(NodePublicKey); err != nil {
				log.WithFields(log.Fields{"type": consts.ConversionError, "error": err}).Error("decoding public key from hex")
				return err
			}

			serializedContract, err := msgpack.Marshal(sc)
			if err != nil {
				logger.WithFields(log.Fields{"type": consts.MarshallingError, "error": err}).Error("marshalling smart contract to msgpack")
				return errorAPI(w, err, http.StatusInternalServerError)
			}

			ret, err := VDEContract(serializedContract, data)
			if err != nil {
				return errorAPI(w, err, http.StatusInternalServerError)
			}
			data.result = ret
		} else {
			err = tx.BuildTransaction(sc, NodePrivateKey, NodePublicKey, hexPubKey)
			if err != nil {
				log.WithFields(log.Fields{"type": consts.ContractError}).Error("Executing contract")
			}
		}

	}

	if ecosystemID > 1 && len(pubkey) == 0 {
		logger.WithFields(log.Fields{"type": consts.EmptyObject}).Error("public key is empty, and state is not default")
		return errorAPI(w, `E_STATELOGIN`, http.StatusForbidden, wallet, ecosystemID)
	}

	if roleParam, ok := data.params["role_id"]; ok && data.roleId == 0 {
		role := roleParam.(int64)
		checkedRole, err := checkRoleFromParam(role, ecosystemID, wallet)
		if err != nil {
			return errorAPI(w, "E_CHECKROLE", http.StatusInternalServerError)
		}

		if checkedRole != role {
			return errorAPI(w, "E_CHECKROLE", http.StatusNotFound)
		}

		data.roleId = checkedRole
	}

	if len(pubkey) == 0 {
		pubkey = data.params[`pubkey`].([]byte)
		if len(pubkey) == 0 {
			logger.WithFields(log.Fields{"type": consts.EmptyObject}).Error("public key is empty")
			return errorAPI(w, `E_EMPTYPUBLIC`, http.StatusBadRequest)
		}
	}

	fmt.Println(string(pubkey))
	verify, err := crypto.CheckSign(pubkey, nonceSalt+msg, data.params[`signature`].([]byte))
	if err != nil {
		logger.WithFields(log.Fields{"type": consts.CryptoError, "pubkey": pubkey, "msg": msg, "signature": string(data.params["signature"].([]byte))}).Error("checking signature")
		return errorAPI(w, err, http.StatusBadRequest)
	}

	if !verify {
		logger.WithFields(log.Fields{"type": consts.InvalidObject, "pubkey": pubkey, "msg": msg, "signature": string(data.params["signature"].([]byte))}).Error("incorrect signature")
		return errorAPI(w, `E_SIGNATURE`, http.StatusBadRequest)
	}

	address := crypto.KeyToAddress(pubkey)

	var (
		sp      model.StateParameter
		founder int64
	)

	sp.SetTablePrefix(converter.Int64ToStr(ecosystemID))
	if ok, err := sp.Get(nil, "founder_account"); err != nil {
		log.WithFields(log.Fields{"type": consts.DBError, "error": err}).Error("getting founder_account parameter")
		return errorAPI(w, `E_SERVER`, http.StatusBadRequest)
	} else if ok {
		founder = converter.StrToInt64(sp.Value)
	}

	result := loginResult{
		EcosystemID: converter.Int64ToStr(ecosystemID),
		KeyID:       converter.Int64ToStr(wallet),
		Address:     address,
		IsOwner:     founder == wallet,
		IsNode:      conf.Config.KeyID == wallet,
		IsVDE:       model.IsTable(fmt.Sprintf(`%d_vde_tables`, consts.DefaultVDE)),
	}

	data.result = &result
	expire := data.params[`expire`].(int64)
	if expire == 0 {
		logger.WithFields(log.Fields{"type": consts.JWTError, "expire": jwtExpire}).Warning("using expire from jwt")
		expire = jwtExpire
	}

	isMobile := "0"
	if mob, ok := data.params[`mobile`]; ok && mob != nil {
		if mob.(string) == `1` || mob.(string) == `true` {
			isMobile = `1`
		}
	}

	claims := JWTClaims{
		KeyID:       result.KeyID,
		EcosystemID: result.EcosystemID,
		IsMobile:    isMobile,
		RoleID:      converter.Int64ToStr(data.roleId),
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(time.Second * time.Duration(expire)).Unix(),
		},
	}

	result.Token, err = jwtGenerateToken(w, claims)
	if err != nil {
		logger.WithFields(log.Fields{"type": consts.JWTError, "error": err}).Error("generating jwt token")
		return errorAPI(w, err, http.StatusInternalServerError)
	}
	claims.StandardClaims.ExpiresAt = time.Now().Add(time.Hour * 30 * 24).Unix()
	result.Refresh, err = jwtGenerateToken(w, claims)
	if err != nil {
		logger.WithFields(log.Fields{"type": consts.JWTError, "error": err}).Error("generating jwt token")
		return errorAPI(w, err, http.StatusInternalServerError)
	}
	result.NotifyKey, result.Timestamp, err = publisher.GetHMACSign(wallet)
	if err != nil {
		return errorAPI(w, err, http.StatusInternalServerError)
	}
	notificator.AddUser(wallet, ecosystemID)
	notificator.UpdateNotifications(ecosystemID, []int64{wallet})

	ra := &model.RolesParticipants{}
	roles, err := ra.SetTablePrefix(ecosystemID).GetActiveMemberRoles(wallet)
	if err != nil {
		log.WithFields(log.Fields{"type": consts.DBError, "error": err}).Error("getting roles")
		return errorAPI(w, `E_SERVER`, http.StatusBadRequest)
	}

	for _, r := range roles {
		var res map[string]string
		if err := json.Unmarshal([]byte(r.Role), &res); err != nil {
			log.WithFields(log.Fields{"type": consts.JSONUnmarshallError, "error": err}).Error("unmarshalling role")
			return errorAPI(w, `E_SERVER`, http.StatusInternalServerError)
		} else {
			result.Roles = append(result.Roles, rolesResult{RoleId: converter.StrToInt64(res["id"]), RoleName: res["name"]})
		}
	}
	notificator.AddUser(wallet, ecosystemID)
	notificator.UpdateNotifications(ecosystemID, []int64{wallet})

	return nil
}

func checkRoleFromParam(role, ecosystemID, wallet int64) (int64, error) {
	if role > 0 {
		ok, err := model.MemberHasRole(nil, ecosystemID, wallet, role)
		if err != nil {
			log.WithFields(log.Fields{
				"type":      consts.DBError,
				"member":    wallet,
				"role":      role,
				"ecosystem": ecosystemID}).Error("check role")

			return 0, err
		}

		if !ok {
			log.WithFields(log.Fields{
				"type":      consts.NotFound,
				"member":    wallet,
				"role":      role,
				"ecosystem": ecosystemID,
			}).Error("member hasn't role")

			return 0, nil
		}
	}
	return role, nil
}
