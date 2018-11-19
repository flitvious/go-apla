package model

import (
	"encoding/json"
	"strconv"

	"github.com/GenesisKernel/go-genesis/packages/types"
	"github.com/GenesisKernel/memdb"
	"github.com/fatih/structs"
	"github.com/mitchellh/mapstructure"
	"github.com/tidwall/buntdb"
)

// Key is model TODO rename to key
type KeySchema struct {
	ID        int64  `json:"id"`
	PublicKey []byte `json:"public_key"`
	Amount    string `json:"amount"`
	Deleted   bool   `json:"deleted"`
	Blocked   bool   `json:"blocked"`
}

func (ks KeySchema) ModelName() string {
	return "keys"
}

func (ks KeySchema) GetIndexes() []types.Index {
	registry := &types.Registry{Name: ks.ModelName()}
	return []types.Index{
		{
			Registry: registry,
			Name:     "amount",
			SortFn:   buntdb.IndexJSON("amount"),
		},
		{
			Registry: registry,
			Name:     "amount+blocked",
			SortFn:   memdb.СompositeIndex(buntdb.IndexJSON("amount"), buntdb.IndexJSON("blocked")),
		},
	}
}

func (ks KeySchema) CreateFromData(data map[string]interface{}) (types.RegistryModel, error) {
	k := &KeySchema{}
	err := mapstructure.Decode(data, &k)
	return k, err
}

func (ks KeySchema) UpdateFromData(model types.RegistryModel, data map[string]interface{}) error {
	oldStruct := model.(*KeySchema)
	return mapstructure.Decode(data, oldStruct)
}

func (ks KeySchema) GetData() map[string]interface{} {
	return structs.Map(ks)
}

func (ks KeySchema) GetPrimaryKey() string {
	return strconv.FormatInt(ks.ID, 10)
}

func (ks *KeySchema) UnmarshalJSON(b []byte) error {
	type schema *KeySchema
	err := json.Unmarshal(b, schema(ks))
	return err
}