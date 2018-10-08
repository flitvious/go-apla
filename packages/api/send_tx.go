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
	"bytes"
	"io/ioutil"
	"net/http"

	"github.com/GenesisKernel/go-genesis/packages/conf/syspar"
	"github.com/GenesisKernel/go-genesis/packages/consts"
	"github.com/GenesisKernel/go-genesis/packages/converter"
	"github.com/GenesisKernel/go-genesis/packages/model"
	"github.com/GenesisKernel/go-genesis/packages/transaction"

	log "github.com/sirupsen/logrus"
)

type sendTxResult struct {
	Hashes map[string]string `json:"hashes"`
}

func sendTx(w http.ResponseWriter, r *http.Request, data *apiData, logger *log.Entry) error {
	err := r.ParseMultipartForm(multipartBuf)
	if err != nil {
		return errorAPI(w, err, http.StatusBadRequest)
	}

	result := &sendTxResult{Hashes: make(map[string]string)}
	for key := range r.MultipartForm.File {
		hash, err := handlerTx(w, r, data, logger, key)
		if err != nil {
			return err
		}
		result.Hashes[key] = hash
	}
	data.result = result

	return nil
}

type contractResult struct {
	Hash string `json:"hash"`
	// These fields are used for VDE
	Message *txstatusError `json:"errmsg,omitempty"`
	Result  string         `json:"result,omitempty"`
}

func handlerTx(w http.ResponseWriter, r *http.Request, data *apiData, logger *log.Entry, key string) (string, error) {
	file, _, err := r.FormFile(key)
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("request.FormFile")
		return "", errorAPI(w, err, http.StatusInternalServerError)
	}
	defer file.Close()
	var txData []byte
	if txData, err = ioutil.ReadAll(file); err != nil {
		logger.WithFields(log.Fields{"type": consts.IOError, "error": err}).Error("reading multipart file")
		return "", err
	}

	if int64(len(txData)) > syspar.GetMaxTxSize() {
		logger.WithFields(log.Fields{"type": consts.ParameterExceeded, "max_size": syspar.GetMaxTxSize(), "size": len(txData)}).Error("transaction size exceeds max size")
		return "", errorAPI(w, "E_LIMITTXSIZE", http.StatusBadRequest, len(txData))
	}

	rtx := &transaction.RawTransaction{}
	if err = rtx.Unmarshall(bytes.NewBuffer(txData)); err != nil {
		return "", errorAPI(w, err, http.StatusInternalServerError)
	}

	if err = model.SendTx(rtx, data.keyId); err != nil {
		logger.WithFields(log.Fields{"type": consts.DBError, "error": err}).Error("sending tx")
		return "", errorAPI(w, err, http.StatusInternalServerError)
	}

	return string(converter.BinToHex(rtx.Hash())), nil
}