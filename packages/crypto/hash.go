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

package crypto

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"hash"

	"github.com/AplaProject/go-apla/packages/consts"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/sha3"
)

type hashProvider int

const (
	_SHA256 hashProvider = iota
)

// GetHMAC returns HMAC hash
func GetHMAC(secret string, message string) ([]byte, error) {
	switch hmacProv {
	case _SHA256:
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write([]byte(message))
		return mac.Sum(nil), nil
	default:
		return nil, ErrUnknownProvider
	}
}

// GetHMACWithTimestamp allows add timestamp
func GetHMACWithTimestamp(secret string, message string, timestamp string) ([]byte, error) {
	switch hmacProv {
	case _SHA256:
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write([]byte(message))
		mac.Write([]byte(timestamp))
		return mac.Sum(nil), nil
	default:
		return nil, ErrUnknownProvider
	}
}

// Hash returns hash of passed bytes
func Hash(msg []byte) ([]byte, error) {
	if len(msg) == 0 {
		log.WithFields(log.Fields{"type": consts.CryptoError, "error": ErrHashingEmpty.Error()}).Debug(ErrHashingEmpty.Error())
	}
	switch hashProv {
	case _SHA256:
		return hashSHA256(msg), nil
	default:
		return nil, ErrUnknownProvider
	}
}

// DoubleHash returns double hash of passed bytes
func DoubleHash(msg []byte) ([]byte, error) {
	if len(msg) == 0 {
		log.WithFields(log.Fields{"type": consts.CryptoError, "error": ErrHashingEmpty.Error()}).Debug(ErrHashingEmpty.Error())
	}
	switch hashProv {
	case _SHA256:
		return hashDoubleSHA256(msg), nil
	default:
		return nil, ErrUnknownProvider
	}
}

func hashSHA256(msg []byte) []byte {
	if len(msg) == 0 {
		log.Debug(ErrHashingEmpty.Error())
	}
	hash := sha256.Sum256(msg)
	return hash[:]
}

//TODO Replace hashDoubleSHA256 with this method
func hashDoubleSHA3(msg []byte) ([]byte, error) {
	if len(msg) == 0 {
		log.Debug(ErrHashingEmpty.Error())
	}
	return hashSHA3256(msg), nil
}

//In the previous version of this function (api v 1.0) this func worked in another way.
//First, hash has been calculated from input data
//Second, obtained hash has been converted to hex
//Third, hex value has been hashed once more time
//In this variant second step is omitted.
func hashDoubleSHA256(msg []byte) []byte {
	firstHash := sha256.Sum256(msg)
	secondHash := sha256.Sum256(firstHash[:])
	return secondHash[:]
}

func hashSHA3256(msg []byte) []byte {
	hash := make([]byte, 64)
	sha3.ShakeSum256(hash, msg)
	return hash[:]
}

func NewHash() hash.Hash {
	return sha256.New()
}

func HashHex(input []byte) (string, error) {
	hash, err := Hash(input)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(hash), nil
}
